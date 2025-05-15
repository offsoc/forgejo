// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"database/sql"
	"fmt"
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

func FindOrCreateFederatedUser(ctx *context_service.Base, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
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
		user, federatedUser, err = createUserFromAP(ctx, personID, federationHost.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		log.Trace("Created federatedUser from ap: %#v", user)
	}
	log.Trace("Got user: %v", user.Name)

	return user, federatedUser, federationHost, nil
}

func findFederatedUser(ctx *context_service.Base, actorURI string) (*user.User, *user.FederatedUser, *forgefed.FederationHost, error) {
	federationHost, err := FindOrCreateFederationHost(ctx, actorURI)
	if err != nil {
		return nil, nil, nil, err
	}
	actorID, err := fm.NewPersonID(actorURI, string(federationHost.NodeInfo.SoftwareName))
	if err != nil {
		return nil, nil, nil, err
	}

	user, federatedUser, err := user.FindFederatedUser(ctx, actorID.ID, federationHost.ID)
	if err != nil {
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

	client, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
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

	// TODO: we should get key material here also to have it immediately
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

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
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

	inbox, err := url.ParseRequestURI(person.Inbox.GetLink().String())
	if err != nil {
		return nil, nil, err
	}

	// TODO: in case of gts we will need an extra request here ?
	pubKeyBytes, err := decodePublicKeyPem(person.PublicKey.PublicKeyPem)
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
		InboxPath:             inbox.Path,
		NormalizedOriginalURL: personID.AsURI(),
		KeyID: sql.NullString{
			String: person.PublicKey.ID.String(),
			Valid:  true,
		},
		PublicKey: sql.Null[sql.RawBytes]{
			V:     pubKeyBytes,
			Valid: true,
		},
	}

	log.Info("Fetch federatedUser:%q", federatedUser)
	return &newUser, &federatedUser, nil
}

func createUserFromAP(ctx context.Context, personID fm.PersonID, federationHostID int64) (*user.User, *user.FederatedUser, error) {
	newUser, federatedUser, err := fetchUserFromAP(ctx, personID, federationHostID)
	if err != nil {
		return nil, nil, err
	}
	err = user.CreateFederatedUser(ctx, newUser, federatedUser)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Created federatedUser:%q", federatedUser)
	return newUser, federatedUser, nil
}
