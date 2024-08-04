// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/auth/password"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"
	context_service "code.gitea.io/gitea/services/context"

	"github.com/google/uuid"
)

func findOrCreateFederatedUser(ctx *context_service.APIContext, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	federationHost, err := getFederationHostForURI(ctx, actorURI)
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
	if user != nil {
		log.Info("Found local federatedUser: %v", user)
	} else {
		user, federatedUser, err = createUserFromAP(ctx, &actorURI, actorID, federationHost.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "Error creating federatedUser", err)
			return nil, nil, nil, err
		}
		log.Info("Created federatedUser from ap: %v", user)
	}
	log.Info("Got user:%v", user.Name)

	return user, federatedUser, federationHost, nil
}

func createFederationHostFromAP(ctx context.Context, actorID fm.ActorID) (*forgefed.FederationHost, error) {
	actionsUser := user.NewActionsUser()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, err
	}
	client, err := clientFactory.WithKeys(ctx, actionsUser, "no idea where to get key material.")
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
	result, err := forgefed.NewFederationHost(nodeInfo, actorID.Host)
	if err != nil {
		return nil, err
	}
	err = forgefed.CreateFederationHost(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func getFederationHostForURI(ctx context.Context, actorURI string) (*forgefed.FederationHost, error) {
	rawActorID, err := fm.NewActorID(actorURI)
	if err != nil {
		return nil, err
	}
	federationHost, err := forgefed.FindFederationHostByFqdn(ctx, rawActorID.Host)
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

func createUserFromAP(ctx context.Context, actorURL *string, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	// ToDo: Do we get a publicKeyId from server, repo or owner or repo?
	actionsUser := user.NewActionsUser()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, err
	}
	client, err := clientFactory.WithKeys(ctx, actionsUser, "no idea where to get key material.")
	if err != nil {
		return nil, nil, err
	}

	// TODO: grab the stuff from Signature's keyID. This works by accident, but
	// we should not rely on it, really.
	body, err := client.GetBody(personID.AsURI())
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
		NormalizedFederatedURI:       personID.AsURI(),
	}
	federatedUser := user.FederatedUser{
		ExternalID:       personID.ID,
		FederationHostID: federationHostID,
	}
	if actorURL != nil {
		federatedUser.ActorURL = actorURL
	}
	err = user.CreateFederatedUser(ctx, &newUser, &federatedUser)
	if err != nil {
		return nil, nil, err
	}
	log.Info("Created federatedUser:%q", federatedUser)

	return &newUser, &federatedUser, nil
}
