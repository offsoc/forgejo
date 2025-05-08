// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"net/url"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	fm "forgejo.org/modules/forgefed"
	context_service "forgejo.org/services/context"
	ap "github.com/go-ap/activitypub"
)

func FindOrCreateKey(ctx *context_service.Base, keyID string) (pubKey any, err error) {
	keyURL, err := url.Parse(keyID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch the public key of the other actor
	// Try if the signing actor is an already known federated user
	_, federatedUser, err := user.FindFederatedUserByKeyID(ctx, keyURL.String())
	if err != nil {
		return nil, err
	}

	if federatedUser != nil && federatedUser.PublicKey.Valid {
		pubKey, err := x509.ParsePKIXPublicKey(federatedUser.PublicKey.V)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}

	// Try if the signing actor is an already known federation host
	federationHost, err := forgefed.FindFederationHostByKeyID(ctx, keyURL.String())
	if err != nil {
		return nil, err
	}

	if federationHost != nil && federationHost.PublicKey.Valid {
		pubKey, err := x509.ParsePKIXPublicKey(federationHost.PublicKey.V)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}

	// Fetch missing public key
	pubKey, pubKeyBytes, apPerson, err := fetchKeyFromAp(ctx, *keyURL, federationHost)
	if err != nil {
		return nil, err
	}
	if apPerson.Type == ap.ActivityVocabularyType("Application") {
		rawActorID, err := fm.NewActorID(apPerson.ID.String())
		if err != nil {
			return nil, err
		}

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
	} else if apPerson.Type == ap.ActivityVocabularyType("Person") {
		rawPersonID, err := fm.NewPersonID(apPerson.ID.String(), string(federationHost.NodeInfo.SoftwareName))
		if err != nil {
			return nil, err
		}
		// Check federatedUser.id = person.id
		if federatedUser.ExternalID != rawPersonID.ID {
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
		user.UpdateFederatedUser(ctx, federatedUser)
	}

	return pubKey, err
}

func fetchKeyFromAp(ctx *context_service.Base, keyURL url.URL, federationHost *forgefed.FederationHost) (pubKey any, pubKeyBytes []byte, apPerson *ap.Person, err error) {
	actionsUser := user.NewAPServerActor()
	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	apClient, err := clientFactory.WithKeys(ctx, actionsUser, actionsUser.APActorKeyID())
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
		return nil, fmt.Errorf("could not decode publicKeyPem to PUBLIC KEY pem block type")
	}

	return block.Bytes, nil
}
