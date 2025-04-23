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

func FindOrCreateFederationHost(ctx *context_service.Base, actorURI string) (*forgefed.FederationHost, error) {
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdnAndPort(ctx, rawActorID.Host, rawActorID.HostPort)
	if err != nil {
		return nil, err
	}
	if federationHost == nil {
		result, err := createFederationHostFromAP(ctx, rawActorID)
		if err != nil {
			return nil, err
		}
		federationHost = result
	}
	return federationHost, nil
}

func FindOrCreateFederatedUser(ctx *context_service.APIContext, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	user, federatedUser, federationHost, err := findFederatedUser(ctx, actorURI)
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
		user, federatedUser, err = createUserFromAP(ctx.Base, personID, federationHost.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "Error creating federatedUser", err)
			return nil, nil, nil, err
		}
		log.Trace("Created federatedUser from ap: %#v", user)
	}
	log.Trace("Got user: %v", user.Name)

	return user, federatedUser, federationHost, nil
}

func FollowRemoteActor(ctx *context_service.APIContext, localUser *user.User, actorURI string) error {
	_, federatedUser, federationHost, err := FindOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		return err
	}

	followReq, err := fm.NewForgeFollow(localUser, actorURI)
	if err != nil {
		return err
	}

	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).
		Marshal(followReq)
	if err != nil {
		return err
	}

	hostURL := federationHost.AsURL()
	return pendingQueue.Push(pendingQueueItem{
		InboxURL: hostURL.JoinPath(federatedUser.InboxPath).String(),
		Doer:     localUser,
		Payload:  payload,
	})
}

func findFederatedUser(ctx *context_service.APIContext, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	federationHost, err := FindOrCreateFederationHost(ctx.Base, actorURI)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Wrong FederationHost", err)
		return nil, nil, nil, err
	}
	actorID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "Invalid PersonID", err)
		return nil, nil, nil, err
	}

	user, federatedUser, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Searching for user failed", err)
		return nil, nil, nil, err
	}

	return user, federatedUser, federationHost, nil
}

func createFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*forgefed.FederationHost, error) {
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

func fetchUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.APActorKeyID())
	if err != nil {
		return nil, nil, err
	}

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

	inbox, err := url.ParseRequestURI(person.Inbox.GetLink().String())
	if err != nil {
		return nil, nil, err
	}

	federatedUser := user.FederatedUser{
		ExternalID:            personID.ID,
		FederationHostID:      federationHostID,
		InboxPath:             inbox.Path,
		NormalizedOriginalURL: personID.AsURI(),
	}

	log.Info("Fetch federatedUser:%q", federatedUser)
	return &newUser, &federatedUser, nil
}

func createUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	newUser, federatedUser, err := fetchUserFromAP(ctx, personID, federationHostID)
	err = user.CreateFederatedUser(ctx, newUser, federatedUser)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Created federatedUser:%q", federatedUser)
	return newUser, federatedUser, nil
}
