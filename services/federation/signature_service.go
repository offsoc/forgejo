// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	fm "forgejo.org/modules/forgefed"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

// Factory function for ActorID. Created struct is asserted to be valid
func NewActorIDFromKeyId(ctx *context_service.Base, uri string) (fm.ActorID, error) {
	parsedURI, err := url.Parse(uri)
	parsedURI.Fragment = ""
	if err != nil {
		return fm.ActorID{}, err
	}

	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return fm.ActorID{}, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
	if err != nil {
		return fm.ActorID{}, err
	}

	userResponse, err := apClient.GetBody(parsedURI.String())
	if err != nil {
		return fm.ActorID{}, err
	}

	var actor ap.Actor
	err = actor.UnmarshalJSON(userResponse)
	if err != nil {
		return fm.ActorID{}, err
	}

	result, err := fm.NewActorID(actor.PublicKey.Owner.String())
	return result, err
}

func FindOrCreateFederatedUserKey(ctx *context_service.Base, keyID string) (pubKey any, err error) {
	var federatedUser *user.FederatedUser
	var keyURL *url.URL

	keyURL, err = url.Parse(keyID)
	if err != nil {
		return nil, err
	}

	// Try if the signing actor is an already known federated user
	_, federatedUser, err = user.FindFederatedUserByKeyID(ctx, keyURL.String())
	if err != nil {
		return nil, err
	}

	if federatedUser == nil {
		rawActorID, err := NewActorIDFromKeyId(ctx, keyID)
		if err != nil {
			return nil, err
		}

		_, federatedUser, _, err = FindOrCreateFederatedUser(ctx, rawActorID.AsURI())
		if err != nil {
			return nil, err
		}
	} else {
		_, err = forgefed.GetFederationHost(ctx, federatedUser.FederationHostID)
		if err != nil {
			return nil, err
		}
	}

	if federatedUser.PublicKey.Valid {
		pubKey, err := x509.ParsePKIXPublicKey(federatedUser.PublicKey.V)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}

	// Fetch missing public key
	pubKey, pubKeyBytes, apPerson, err := fetchKeyFromAp(ctx, *keyURL)
	if err != nil {
		return nil, err
	}
	if apPerson.Type == ap.ActivityVocabularyType("Person") {
		// Check federatedUser.id = person.id
		if federatedUser.ExternalID != apPerson.ID.String() {
			return nil, fmt.Errorf("federated user fetched (%v) does not match the stored one %v", apPerson, federatedUser)
		}
		// update federated user
		federatedUser.KeyID = sql.NullString{
			String: apPerson.PublicKey.ID.String(),
			Valid:  true,
		}
		federatedUser.PublicKey = sql.Null[sql.RawBytes]{
			V:     pubKeyBytes,
			Valid: true,
		}
		err = user.UpdateFederatedUser(ctx, federatedUser)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}
	return nil, nil
}

func FindOrCreateFederationHostKey(ctx *context_service.Base, keyID string) (pubKey any, err error) {
	keyURL, err := url.Parse(keyID)
	if err != nil {
		return nil, err
	}
	rawActorID, err := NewActorIDFromKeyId(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Is there an already known federation host?
	federationHost, err := forgefed.FindFederationHostByKeyID(ctx, keyURL.String())
	if err != nil {
		return nil, err
	}

	if federationHost == nil {
		federationHost, err = FindOrCreateFederationHost(ctx, rawActorID.AsURI())
		if err != nil {
			return nil, err
		}
	}

	// Is there an already an key?
	if federationHost.PublicKey.Valid {
		pubKey, err := x509.ParsePKIXPublicKey(federationHost.PublicKey.V)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}

	// If not, fetch missing public key
	pubKey, pubKeyBytes, apPerson, err := fetchKeyFromAp(ctx, *keyURL)
	if err != nil {
		return nil, err
	}
	if apPerson.Type == ap.ActivityVocabularyType("Application") {
		// Check federationhost.id = person.id
		if federationHost.HostPort != rawActorID.HostPort || federationHost.HostFqdn != rawActorID.Host ||
			federationHost.HostSchema != rawActorID.HostSchema {
			return nil, fmt.Errorf("federation host fetched (%v) does not match the stored one %v", apPerson, federationHost)
		}
		// update federation host
		federationHost.KeyID = sql.NullString{
			String: apPerson.PublicKey.ID.String(),
			Valid:  true,
		}
		federationHost.PublicKey = sql.Null[sql.RawBytes]{
			V:     pubKeyBytes,
			Valid: true,
		}
		err = forgefed.UpdateFederationHost(ctx, federationHost)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}
	return nil, nil
}

func fetchKeyFromAp(ctx *context_service.Base, keyURL url.URL) (pubKey any, pubKeyBytes []byte, apPerson *ap.Person, err error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.KeyID())
	if err != nil {
		return nil, nil, nil, err
	}

	b, err := apClient.GetBody(keyURL.String())
	if err != nil {
		return nil, nil, nil, err
	}

	person := ap.PersonNew(ap.IRI(keyURL.String()))
	err = person.UnmarshalJSON(b)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ActivityStreams type cannot be converted to one known to have publicKey property: %w", err)
	}

	pubKeyFromAp := person.PublicKey
	if pubKeyFromAp.ID.String() != keyURL.String() {
		return nil, nil, nil, fmt.Errorf("cannot find publicKey with id: %v in %v", keyURL, string(b))
	}

	pubKeyBytes, err = decodePublicKeyPem(pubKeyFromAp.PublicKeyPem)
	if err != nil {
		return nil, nil, nil, err
	}

	pubKey, err = x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	return pubKey, pubKeyBytes, person, err
}

func decodePublicKeyPem(pubKeyPem string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pubKeyPem))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("could not decode publicKeyPem to PUBLIC KEY pem block type")
	}

	return block.Bytes, nil
}
