// Copyright 2015 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"net/http"

	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/validation"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	user_service "forgejo.org/services/user"
)

// ListEmails list all of the authenticated user's email addresses
// see https://github.com/gogits/go-gogs-client/wiki/Users-Emails#list-email-addresses-for-a-user
func ListEmails(ctx *context.APIContext) {
	// swagger:operation GET /user/emails user userListEmails
	// ---
	// summary: List the authenticated user's email addresses
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/EmailList"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	emails, err := user_model.GetEmailAddresses(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetEmailAddresses", err)
		return
	}
	apiEmails := make([]*api.Email, len(emails))
	for i := range emails {
		apiEmails[i] = convert.ToEmail(emails[i])
	}
	ctx.JSON(http.StatusOK, &apiEmails)
}

// AddEmail add an email address
func AddEmail(ctx *context.APIContext) {
	// swagger:operation POST /user/emails user userAddEmail
	// ---
	// summary: Add email addresses
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateEmailOption"
	// responses:
	//   '201':
	//     "$ref": "#/responses/EmailList"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateEmailOption)
	if len(form.Emails) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "", "Email list empty")
		return
	}

	if err := user_service.AddEmailAddresses(ctx, ctx.Doer, form.Emails); err != nil {
		if user_model.IsErrEmailAlreadyUsed(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", "Email address has been used: "+err.(user_model.ErrEmailAlreadyUsed).Email)
		} else if validation.IsErrEmailInvalid(err) {
			email := ""
			if typedError, ok := err.(validation.ErrEmailInvalid); ok {
				email = typedError.Email
			}

			errMsg := fmt.Sprintf("Email address %q invalid", email)
			ctx.Error(http.StatusUnprocessableEntity, "", errMsg)
		} else {
			ctx.Error(http.StatusInternalServerError, "AddEmailAddresses", err)
		}
		return
	}

	emails, err := user_model.GetEmailAddresses(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetEmailAddresses", err)
		return
	}

	apiEmails := make([]*api.Email, 0, len(emails))
	for _, email := range emails {
		apiEmails = append(apiEmails, convert.ToEmail(email))
	}
	ctx.JSON(http.StatusCreated, apiEmails)
}

// DeleteEmail delete email
func DeleteEmail(ctx *context.APIContext) {
	// swagger:operation DELETE /user/emails user userDeleteEmail
	// ---
	// summary: Delete email addresses
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/DeleteEmailOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.DeleteEmailOption)
	if len(form.Emails) == 0 {
		ctx.Status(http.StatusNoContent)
		return
	}

	if err := user_service.DeleteEmailAddresses(ctx, ctx.Doer, form.Emails); err != nil {
		if user_model.IsErrEmailAddressNotExist(err) {
			ctx.Error(http.StatusNotFound, "DeleteEmailAddresses", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteEmailAddresses", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}
