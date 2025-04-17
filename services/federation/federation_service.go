// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/auth/password"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/validation"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
)

func Init() error {
	if err := initDeliveryQueue(); err != nil {
		return err
	}
	if err := initRefreshQueue(); err != nil {
		return err
	}
	return initPendingQueue()
}

func FollowRemoteActor(ctx *context_service.APIContext, localUser *user.User, actorURI string) error {
	_, federatedUser, _, err := findOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		return err
	}

	// TODO: Encapsulate Factory and add validation
	followReq := ap.FollowNew(
		ap.IRI(localUser.APActorID()+"/follows/"+uuid.New().String()),
		ap.IRI(actorURI),
	)
	followReq.Actor = ap.IRI(localUser.APActorID())
	followReq.Target = ap.IRI(actorURI)
	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).
		Marshal(followReq)
	if err != nil {
		return err
	}

	return pendingQueue.Push(pendingQueueItem{
		FederatedUserID: federatedUser.ID,
		Doer:            localUser,
		Payload:         payload,
	})
}

func findFederatedUser(ctx *context_service.APIContext, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, *fm.PersonID, error) {
	federationHost, err := GetFederationHostForURI(ctx.Base, actorURI)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Wrong FederationHost", err)
		return nil, nil, nil, nil, err
	}
	actorID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "Invalid PersonID", err)
		return nil, nil, nil, nil, err
	}

	user, federatedUser, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Searching for user failed", err)
		return nil, nil, nil, nil, err
	}

	return user, federatedUser, federationHost, &actorID, nil
}

func findOrCreateFederatedUser(ctx *context_service.APIContext, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	// TODO: align this function
	user, federatedUser, federationHost, _, err := findFederatedUser(ctx, actorURI)
	if err != nil {
		return nil, nil, nil, err
	}
	personID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return nil, nil, nil, err
	}

	if user != nil {
		log.Trace("Found local federatedUser: %#v", user)
	} else {
		user, federatedUser, err = CreateUserFromAP(ctx.Base, personID, federationHost.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "Error creating federatedUser", err)
			return nil, nil, nil, err
		}
		log.Trace("Created federatedUser from ap: %#v", user)
	}
	log.Trace("Got user: %v", user.Name)

	return user, federatedUser, federationHost, nil
}

func CreateFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*forgefed.FederationHost, error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, err
	}

	client, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.APActorKeyID())
	if err != nil {
		return nil, err
	}

	body, err := client.GetBody(actorID.AsWellKnownNodeInfoURI())
	if err != nil {
		return nil, err
	}

	nodeInfoWellKnown, err := forgefed.NewNodeInfoWellKnown(body)
	if err != nil {
		return nil, err
	}

	body, err = client.GetBody(nodeInfoWellKnown.Href)
	if err != nil {
		return nil, err
	}

	nodeInfo, err := forgefed.NewNodeInfo(body)
	if err != nil {
		return nil, err
	}

	result, err := forgefed.NewFederationHost(actorID.Host, nodeInfo, actorID.HostPort, actorID.HostSchema)
	if err != nil {
		return nil, err
	}

	err = forgefed.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func GetFederationHostForURI(ctx *context_service.Base, actorURI string) (*forgefed.FederationHost, error) {
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdnAndPort(ctx, rawActorID.Host, rawActorID.HostPort)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := CreateFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}

func CreateUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.APActorKeyID())
	if err != nil {
		return nil, nil, err
	}

	// TODO: readd new kind of signature checks
	// var idIRI string

	// Grab the keyID from the signature
	// v, err := httpsig.NewVerifier(ctx.Req)
	// if err != nil {
	//   idIRI = personID.AsURI()
	// } else {
	// 	idIRIURL, err := url.Parse(v.KeyId())
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	// 	idIRI = idIRIURL.String()
	// }

	body, err := apClient.GetBody(personID.AsURI())
	if err != nil {
		return nil, nil, err
	}

	person := fm.ForgePerson{}
	err = person.UnmarshalJSON(body)
	if err != nil {
		return nil, nil, err
	}

	if res, err := validation.IsValid(person); !res {
		return nil, nil, err
	}

	log.Info("Fetched valid person:%q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, nil, err
	}

	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	fullName := person.Name.String()

	if len(person.Name) == 0 {
		fullName = name
	}

	password, err := password.Generate(32)
	if err != nil {
		return nil, nil, err
	}

	newUser := user.User{
		LowerName:                    strings.ToLower(name),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         user.UserTypeRemoteUser,
		IsAdmin:                      false,
	}

	federatedUser := user.FederatedUser{
		ExternalID:            personID.ID,
		FederationHostID:      federationHostID,
		NormalizedOriginalURL: personID.AsURI(),
	}

	err = user.CreateFederatedUser(ctx, &newUser, &federatedUser)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Created federatedUser:%q", federatedUser)
	return &newUser, &federatedUser, nil
}
