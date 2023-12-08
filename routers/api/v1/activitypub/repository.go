// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/gitea/models/activitypub"
	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"github.com/google/uuid"

	ap "github.com/go-ap/activitypub"
	pwd_gen "github.com/sethvargo/go-password/password"
)

var actionsUser = user_model.NewActionsUser()

func generateUUIDMail(person ap.Actor) (string, error) {
	// UUID@remote.host
	id := uuid.New().String()

	//url, err := url.Parse(person.URL.GetID().String())

	//host := url.Host

	return strings.Join([]string{id, "example.com"}, "@"), nil
}

func generateRemoteUserName(person ap.Actor) (string, error) {
	u, err := url.Parse(person.URL.GetID().String())
	if err != nil {
		return "", err
	}

	host := strings.Split(u.Host, ":")[0] // no port in username

	if name := person.PreferredUsername.String(); name != "" {
		return strings.Join([]string{name, host}, "_"), nil
	}
	if name := person.Name.String(); name != "" {
		return strings.Join([]string{name, host}, "_"), nil
	}

	return "", fmt.Errorf("empty name, preferredUsername field")
}

func generateRandomPassword() (string, error) {
	// Generate a password that is 64 characters long with 10 digits, 10 symbols,
	// allowing upper and lower case letters, disallowing repeat characters.
	res, err := pwd_gen.Generate(32, 10, 10, false, false)
	if err != nil {
		return "", err
	}
	return res, err
}

func searchUsersByPerson(actorId string) ([]*user_model.User, error) {

	actionsUser.IsAdmin = true

	options := &user_model.SearchUserOptions{
		LoginName:       actorId,
		Actor:           actionsUser,
		Type:            user_model.UserTypeRemoteUser,
		OrderBy:         db.SearchOrderByAlphabetically,
		ListOptions:     db.ListOptions{PageSize: 1},
		IsActive:        util.OptionalBoolFalse,
		IncludeReserved: true,
	}
	users, _, err := user_model.SearchUsers(db.DefaultContext, options)
	if err != nil {
		return []*user_model.User{}, fmt.Errorf("search failed: %v", err)
	}

	log.Info("local found users: %v", len(users))

	return users, nil

}

func getBody(remoteStargazer, starReceiver string, ctx *context.APIContext) ([]byte, error) { // ToDo: We could split this: move body reading to unmarshall

	// TODO: The star receiver signs the http get request will maybe not work.
	// The remote repo has probably diferent keys as the local one.
	// > The local user signs the request with their private key, the public key is publicly available to anyone. I do not see an issue here.
	// Why should we use a signed request here at all?
	// > To provide an extra layer of security against in flight tampering: https://github.com/go-fed/httpsig/blob/55836744818e/httpsig.go#L116

	client, err := api.NewClient(ctx, actionsUser, starReceiver) // ToDo: Do we get a publicKeyId of owner or repo?
	if err != nil {
		return []byte{0}, err
	}
	// get_person_by_rest
	response, err := client.Get([]byte{0}, remoteStargazer)
	if err != nil {
		return []byte{0}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return []byte{0}, err
	}

	log.Info("remoteStargazer: %v", remoteStargazer)
	log.Info("http client. %v", client)
	log.Info("response: %v\n error: ", response, err)

	return body, nil
}

func unmarshallPersonJSON(body []byte) (ap.Person, error) {

	// parse response
	person := ap.Person{}
	err := person.UnmarshalJSON(body)
	if err != nil {
		return ap.Person{}, err
	}

	log.Info("Person is: %v", person)
	log.Info("Person Name is: %v", person.PreferredUsername)
	log.Info("Person URL is: %v", person.URL)

	return person, nil

}

func createFederatedUserFromPerson(person ap.Person, remoteStargazer string) (*user_model.User, error) {
	email, err := generateUUIDMail(person)
	if err != nil {
		return &user_model.User{}, err
	}
	username, err := generateRemoteUserName(person)
	if err != nil {
		return &user_model.User{}, err
	}
	password, err := generateRandomPassword()
	if err != nil {
		return &user_model.User{}, err
	}
	user := &user_model.User{
		LowerName:                    strings.ToLower(username),
		Name:                         username,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    remoteStargazer,
		Type:                         user_model.UserTypeRemoteUser,
		IsAdmin:                      false,
	}
	return user, nil
}

func saveFederatedUserRecord(ctx *context.APIContext, user *user_model.User) error {
	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive:     util.OptionalBoolFalse,
		IsRestricted: util.OptionalBoolFalse,
	}
	if err := user_model.CreateUser(ctx, user, overwriteDefault); err != nil {
		return err
	}
	log.Info("User created!")
	return nil
}

// Repository function returns the Repository actor for a repo
func Repository(ctx *context.APIContext) {
	// swagger:operation GET /activitypub/repository-id/{repository-id} activitypub activitypubRepository
	// ---
	// summary: Returns the Repository actor for a repo
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActivityPub"

	link := fmt.Sprintf("%s/api/v1/activitypub/repository-id/%d", strings.TrimSuffix(setting.AppURL, "/"), ctx.Repo.Repository.ID)
	repo := forgefed.RepositoryNew(ap.IRI(link))

	repo.Name = ap.NaturalLanguageValuesNew()
	err := repo.Name.Set("en", ap.Content(ctx.Repo.Repository.Name))
	if err != nil {
		ctx.ServerError("Set Name", err)
		return
	}

	response(ctx, repo)
}

// PersonInbox function handles the incoming data for a repository inbox
func RepositoryInbox(ctx *context.APIContext) {
	// swagger:operation POST /activitypub/repository-id/{repository-id}/inbox activitypub activitypubRepository
	// ---
	// summary: Send to the inbox
	// produces:
	// - application/json
	// parameters:
	// - name: repository-id
	//   in: path
	//   description: repository ID of the repo
	//   type: integer
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/Star"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	var user *user_model.User

	repository := ctx.Repo.Repository
	log.Info("RepositoryInbox: repo: %v, owned by:  %v", repository.Name, repository.OwnerName)

	activity := web.GetForm(ctx).(*forgefed.Star)
	log.Info("RepositoryInbox: Activity.Source: %v, Activity.Actor %v, Activity.Actor.Id %v", activity.Source, activity.Actor, activity.Actor.GetID().String())

	// assume actor is: "actor": "https://codeberg.org/api/v1/activitypub/user-id/12345" - NB: This might be actually the ID? Maybe check vocabulary.
	//    "https://Codeberg.org/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org:443/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345"
	//    "https://user:password@codeberg.org/api/v1/activitypub/user-id/12345"
	//    "https://codeberg.org/api/v1/activitypub//user-id/12345"

	// parse senderActorId
	// senderActorId holds the data to construct the sender of the star
	validatedActorId, err := activitypub.ValidateAndParseIRI(activity.Actor.GetID().String())
	if err != nil {
		ctx.ServerError("Set Name", err)
		return
	}
	actorId := activitypub.ParseActorID(validatedActorId, string(activity.Source))

	// Is the ActorID Struct valid?
	actorId.PanicIfInvalid()
	log.Info("RepositoryInbox: Actor parsed. %v", actorId)

	remoteStargazer := actorId.GetNormalizedUri() // used as LoginName in newly created user
	log.Info("remotStargazer: %v", remoteStargazer)

	// Check if user already exists
	// TODO: If the usesrs-id points to our current host, we've to use an alterantive search ...
	// > We might need to discuss this further with the community, because when we execute this bit of code here, the federated api has been called.
	// > Thus the searching for non-federated users could facilitate spoofing of already existing user-ids for some (malicious) purposes.
	users, err := searchUsersByPerson(remoteStargazer)
	if err != nil {
		panic(fmt.Errorf("searching for user failed: %v", err))
	}

	switch len(users) {
	case 0:
		{
			body, err := getBody(remoteStargazer, "does not exist yet", ctx) // ToDo: We would need to insert the repo or its owners key here
			if err != nil {
				panic(fmt.Errorf("http get failed: %v", err))
			}
			person, err := unmarshallPersonJSON(body)
			if err != nil {
				panic(fmt.Errorf("getting user failed: %v", err))
			}
			user, err = createFederatedUserFromPerson(person, remoteStargazer)
			if err != nil {
				panic(fmt.Errorf("create federated user: %w", err))
			}
			err = saveFederatedUserRecord(ctx, user)
			if err != nil {
				panic(fmt.Errorf("save user: %w", err))
			}
		}
	case 1:
		{
			user = users[0]
			log.Info("Found user full name was: %v", user.FullName)
			log.Info("Found user name was: %v", user.Name)
			log.Info("Found user name was: %v", user.LoginName)
			log.Info("%v", user)
		}
	default:
		{
			panic(fmt.Errorf("found more than one matches for federated users"))
		}
	}

	remoteUser, err := user_model.GetUserByEmail(ctx, user.Email)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "StarRepo", err)
		return
	}

	// check if already starred by this user
	alreadyStared := repo_model.IsStaring(ctx, remoteUser.ID, repository.ID)
	switch alreadyStared {
	case true: // execute unstar action
		{
			err = repo_model.StarRepo(ctx, remoteUser.ID, repository.ID, false)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "StarRepo", err)
				return
			}
		}
	case false: // execute star action
		{
			err = repo_model.StarRepo(ctx, remoteUser.ID, repository.ID, true)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "StarRepo", err)
				return
			}
		}
	}

	// wait 5 sec.
	time.Sleep(5 * time.Second)

	ctx.Status(http.StatusNoContent)
}
