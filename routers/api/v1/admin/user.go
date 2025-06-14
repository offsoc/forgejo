// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"forgejo.org/models"
	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/auth/password"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/user"
	"forgejo.org/routers/api/v1/utils"
	asymkey_service "forgejo.org/services/asymkey"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	"forgejo.org/services/mailer"
	user_service "forgejo.org/services/user"
)

func parseAuthSource(ctx *context.APIContext, u *user_model.User, sourceID int64) {
	if sourceID == 0 {
		return
	}

	source, err := auth.GetSourceByID(ctx, sourceID)
	if err != nil {
		if auth.IsErrSourceNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "auth.GetSourceByID", err)
		}
		return
	}

	u.LoginType = source.Type
	u.LoginSource = source.ID
}

// CreateUser create a user
func CreateUser(ctx *context.APIContext) {
	// swagger:operation POST /admin/users admin adminCreateUser
	// ---
	// summary: Create a user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateUserOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/User"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateUserOption)

	u := &user_model.User{
		Name:               form.Username,
		FullName:           form.FullName,
		Email:              form.Email,
		Passwd:             form.Password,
		MustChangePassword: true,
		LoginType:          auth.Plain,
		LoginName:          form.LoginName,
	}
	if form.MustChangePassword != nil {
		u.MustChangePassword = *form.MustChangePassword
	}

	parseAuthSource(ctx, u, form.SourceID)
	if ctx.Written() {
		return
	}

	if u.LoginType == auth.Plain {
		if len(form.Password) < setting.MinPasswordLength {
			err := errors.New("PasswordIsRequired")
			ctx.Error(http.StatusBadRequest, "PasswordIsRequired", err)
			return
		}

		if !password.IsComplexEnough(form.Password) {
			err := errors.New("PasswordComplexity")
			ctx.Error(http.StatusBadRequest, "PasswordComplexity", err)
			return
		}

		if err := password.IsPwned(ctx, form.Password); err != nil {
			if password.IsErrIsPwnedRequest(err) {
				log.Error(err.Error())
			}
			ctx.Error(http.StatusBadRequest, "PasswordPwned", errors.New("PasswordPwned"))
			return
		}
	}

	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive:     optional.Some(true),
		IsRestricted: optional.FromPtr(form.Restricted),
	}

	if form.Visibility != "" {
		visibility := api.VisibilityModes[form.Visibility]
		overwriteDefault.Visibility = &visibility
	}

	// Update the user creation timestamp. This can only be done after the user
	// record has been inserted into the database; the insert intself will always
	// set the creation timestamp to "now".
	if form.Created != nil {
		u.CreatedUnix = timeutil.TimeStamp(form.Created.Unix())
		u.UpdatedUnix = u.CreatedUnix
	}

	if err := user_model.AdminCreateUser(ctx, u, overwriteDefault); err != nil {
		if user_model.IsErrUserAlreadyExist(err) ||
			user_model.IsErrEmailAlreadyUsed(err) ||
			db.IsErrNameReserved(err) ||
			db.IsErrNameCharsNotAllowed(err) ||
			validation.IsErrEmailInvalid(err) ||
			db.IsErrNamePatternNotAllowed(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateUser", err)
		}
		return
	}

	if !validation.IsEmailDomainAllowed(u.Email) {
		ctx.Resp.Header().Add("X-Gitea-Warning", fmt.Sprintf("the domain of user email %s conflicts with EMAIL_DOMAIN_ALLOWLIST or EMAIL_DOMAIN_BLOCKLIST", u.Email))
	}

	log.Trace("Account created by admin (%s): %s", ctx.Doer.Name, u.Name)

	// Send email notification.
	if form.SendNotify {
		mailer.SendRegisterNotifyMail(u)
	}
	ctx.JSON(http.StatusCreated, convert.ToUser(ctx, u, ctx.Doer))
}

// EditUser api for modifying a user's information
func EditUser(ctx *context.APIContext) {
	// swagger:operation PATCH /admin/users/{username} admin adminEditUser
	// ---
	// summary: Edit an existing user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user to edit
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditUserOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/User"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.EditUserOption)

	// If either LoginSource or LoginName is given, the other must be present too.
	if form.SourceID != nil || form.LoginName != nil {
		if form.SourceID == nil || form.LoginName == nil {
			ctx.Error(http.StatusUnprocessableEntity, "LoginSourceAndLoginName", errors.New("source_id and login_name must be specified together"))
			return
		}
	}

	authOpts := &user_service.UpdateAuthOptions{
		LoginSource:        optional.FromPtr(form.SourceID),
		LoginName:          optional.FromPtr(form.LoginName),
		Password:           optional.FromNonDefault(form.Password),
		MustChangePassword: optional.FromPtr(form.MustChangePassword),
		ProhibitLogin:      optional.FromPtr(form.ProhibitLogin),
	}
	if err := user_service.UpdateAuth(ctx, ctx.ContextUser, authOpts); err != nil {
		switch {
		case errors.Is(err, password.ErrMinLength):
			ctx.Error(http.StatusBadRequest, "PasswordTooShort", fmt.Errorf("password must be at least %d characters", setting.MinPasswordLength))
		case errors.Is(err, password.ErrComplexity):
			ctx.Error(http.StatusBadRequest, "PasswordComplexity", err)
		case errors.Is(err, password.ErrIsPwned), password.IsErrIsPwnedRequest(err):
			ctx.Error(http.StatusBadRequest, "PasswordIsPwned", err)
		default:
			ctx.Error(http.StatusInternalServerError, "UpdateAuth", err)
		}
		return
	}

	if form.Email != nil {
		if err := user_service.AdminAddOrSetPrimaryEmailAddress(ctx, ctx.ContextUser, *form.Email); err != nil {
			switch {
			case validation.IsErrEmailInvalid(err):
				ctx.Error(http.StatusBadRequest, "EmailInvalid", err)
			case user_model.IsErrEmailAlreadyUsed(err):
				ctx.Error(http.StatusBadRequest, "EmailUsed", err)
			default:
				ctx.Error(http.StatusInternalServerError, "AddOrSetPrimaryEmailAddress", err)
			}
			return
		}

		if !validation.IsEmailDomainAllowed(*form.Email) {
			ctx.Resp.Header().Add("X-Gitea-Warning", fmt.Sprintf("the domain of user email %s conflicts with EMAIL_DOMAIN_ALLOWLIST or EMAIL_DOMAIN_BLOCKLIST", *form.Email))
		}
	}

	opts := &user_service.UpdateOptions{
		FullName:                optional.FromPtr(form.FullName),
		Website:                 optional.FromPtr(form.Website),
		Location:                optional.FromPtr(form.Location),
		Description:             optional.FromPtr(form.Description),
		Pronouns:                optional.FromPtr(form.Pronouns),
		IsActive:                optional.FromPtr(form.Active),
		IsAdmin:                 optional.FromPtr(form.Admin),
		Visibility:              optional.FromNonDefault(api.VisibilityModes[form.Visibility]),
		AllowGitHook:            optional.FromPtr(form.AllowGitHook),
		AllowImportLocal:        optional.FromPtr(form.AllowImportLocal),
		MaxRepoCreation:         optional.FromPtr(form.MaxRepoCreation),
		AllowCreateOrganization: optional.FromPtr(form.AllowCreateOrganization),
		IsRestricted:            optional.FromPtr(form.Restricted),
	}

	if err := user_service.UpdateUser(ctx, ctx.ContextUser, opts); err != nil {
		if models.IsErrDeleteLastAdminUser(err) {
			ctx.Error(http.StatusBadRequest, "LastAdmin", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "UpdateUser", err)
		}
		return
	}

	log.Trace("Account profile updated by admin (%s): %s", ctx.Doer.Name, ctx.ContextUser.Name)

	ctx.JSON(http.StatusOK, convert.ToUser(ctx, ctx.ContextUser, ctx.Doer))
}

// DeleteUser api for deleting a user
func DeleteUser(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/users/{username} admin adminDeleteUser
	// ---
	// summary: Delete a user
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user to delete
	//   type: string
	//   required: true
	// - name: purge
	//   in: query
	//   description: purge the user from the system completely
	//   type: boolean
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	if ctx.ContextUser.IsOrganization() {
		ctx.Error(http.StatusUnprocessableEntity, "", fmt.Errorf("%s is an organization not a user", ctx.ContextUser.Name))
		return
	}

	// admin should not delete themself
	if ctx.ContextUser.ID == ctx.Doer.ID {
		ctx.Error(http.StatusUnprocessableEntity, "", errors.New("you cannot delete yourself"))
		return
	}

	if err := user_service.DeleteUser(ctx, ctx.ContextUser, ctx.FormBool("purge")); err != nil {
		if models.IsErrUserOwnRepos(err) ||
			models.IsErrUserHasOrgs(err) ||
			models.IsErrUserOwnPackages(err) ||
			models.IsErrDeleteLastAdminUser(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteUser", err)
		}
		return
	}
	log.Trace("Account deleted by admin(%s): %s", ctx.Doer.Name, ctx.ContextUser.Name)

	ctx.Status(http.StatusNoContent)
}

// CreatePublicKey api for creating a public key to a user
func CreatePublicKey(ctx *context.APIContext) {
	// swagger:operation POST /admin/users/{username}/keys admin adminCreatePublicKey
	// ---
	// summary: Add a public key on behalf of a user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of the user
	//   type: string
	//   required: true
	// - name: key
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateKeyOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/PublicKey"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateKeyOption)

	user.CreateUserPublicKey(ctx, *form, ctx.ContextUser.ID)
}

// DeleteUserPublicKey api for deleting a user's public key
func DeleteUserPublicKey(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/users/{username}/keys/{id} admin adminDeleteUserPublicKey
	// ---
	// summary: Delete a user's public key
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the key to delete
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if err := asymkey_service.DeletePublicKey(ctx, ctx.ContextUser, ctx.ParamsInt64(":id")); err != nil {
		if asymkey_model.IsErrKeyNotExist(err) {
			ctx.NotFound()
		} else if asymkey_model.IsErrKeyAccessDenied(err) {
			ctx.Error(http.StatusForbidden, "", "You do not have access to this key")
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteUserPublicKey", err)
		}
		return
	}
	log.Trace("Key deleted by admin(%s): %s", ctx.Doer.Name, ctx.ContextUser.Name)

	ctx.Status(http.StatusNoContent)
}

// SearchUsers API for getting information of the users according the filter conditions
func SearchUsers(ctx *context.APIContext) {
	// swagger:operation GET /admin/users admin adminSearchUsers
	// ---
	// summary: Search users according filter conditions
	// produces:
	// - application/json
	// parameters:
	// - name: source_id
	//   in: query
	//   description: ID of the user's login source to search for
	//   type: integer
	//   format: int64
	// - name: login_name
	//   in: query
	//   description: user's login name to search for
	//   type: string
	// - name: sort
	//   in: query
	//   description: sort order of results
	//   type: string
	//   enum: [oldest, newest, alphabetically, reversealphabetically, recentupdate, leastupdate]
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	listOptions := utils.GetListOptions(ctx)

	sort := ctx.FormString("sort")
	var orderBy db.SearchOrderBy

	switch sort {
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "recentupdate":
		orderBy = db.SearchOrderByRecentUpdated
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	default:
		orderBy = db.SearchOrderByAlphabetically
	}

	intSource, err := strconv.ParseInt(ctx.FormString("source_id"), 10, 64)
	var sourceID optional.Option[int64]
	if ctx.FormString("source_id") == "" || err != nil {
		sourceID = optional.None[int64]()
	} else {
		sourceID = optional.Some(intSource)
	}

	users, maxResults, err := user_model.SearchUsers(ctx, &user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		Type:        user_model.UserTypeIndividual,
		LoginName:   ctx.FormTrim("login_name"),
		SourceID:    sourceID,
		OrderBy:     orderBy,
		ListOptions: listOptions,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchUsers", err)
		return
	}

	results := make([]*api.User, len(users))
	for i := range users {
		results[i] = convert.ToUser(ctx, users[i], ctx.Doer)
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, &results)
}

// RenameUser api for renaming a user
func RenameUser(ctx *context.APIContext) {
	// swagger:operation POST /admin/users/{username}/rename admin adminRenameUser
	// ---
	// summary: Rename a user
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: existing username of user
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/RenameUserOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	if ctx.ContextUser.IsOrganization() {
		ctx.Error(http.StatusUnprocessableEntity, "", fmt.Errorf("%s is an organization not a user", ctx.ContextUser.Name))
		return
	}

	oldName := ctx.ContextUser.Name
	newName := web.GetForm(ctx).(*api.RenameUserOption).NewName

	// Check if user name has been changed
	if err := user_service.AdminRenameUser(ctx, ctx.ContextUser, newName); err != nil {
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			ctx.Error(http.StatusUnprocessableEntity, "", ctx.Tr("form.username_been_taken"))
		case db.IsErrNameReserved(err):
			ctx.Error(http.StatusUnprocessableEntity, "", ctx.Tr("user.form.name_reserved", newName))
		case db.IsErrNamePatternNotAllowed(err):
			ctx.Error(http.StatusUnprocessableEntity, "", ctx.Tr("user.form.name_pattern_not_allowed", newName))
		case db.IsErrNameCharsNotAllowed(err):
			ctx.Error(http.StatusUnprocessableEntity, "", ctx.Tr("user.form.name_chars_not_allowed", newName))
		default:
			ctx.ServerError("ChangeUserName", err)
		}
		return
	}

	log.Trace("User name changed: %s -> %s", oldName, newName)
	ctx.Status(http.StatusNoContent)
}
