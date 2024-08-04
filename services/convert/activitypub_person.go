// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	"code.gitea.io/gitea/models/forgefed"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	api "code.gitea.io/gitea/modules/structs"

	ap "github.com/go-ap/activitypub"
)

func ToActivityPubPersonFeedItem(item *forgefed.FederatedUserActivity) api.APPersonFollowItem {
	return api.APPersonFollowItem{
		ActorID:      item.ExternalID,
		Note:         item.Note,
		OriginalURL:  item.OriginalURL,
		OriginalItem: item.Original,
	}
}

func ToActivityPubPerson(ctx context.Context, user *user_model.User) (*ap.Person, error) {
	link := user.APActorID()
	person := ap.PersonNew(ap.IRI(link))

	person.Name = ap.NaturalLanguageValuesNew()
	err := person.Name.Set("en", ap.Content(user.FullName))
	if err != nil {
		return nil, err
	}

	person.PreferredUsername = ap.NaturalLanguageValuesNew()
	err = person.PreferredUsername.Set("en", ap.Content(user.Name))
	if err != nil {
		return nil, err
	}

	person.URL = ap.IRI(user.HTMLURL())

	person.Icon = ap.Image{
		Type:      ap.ImageType,
		MediaType: "image/png",
		URL:       ap.IRI(user.AvatarLink(ctx)),
	}

	person.Inbox = ap.IRI(link + "/inbox")
	person.Outbox = ap.IRI(link + "/outbox")

	person.PublicKey.ID = ap.IRI(link + "#main-key")
	person.PublicKey.Owner = ap.IRI(link)

	publicKeyPem, err := activitypub.GetPublicKey(ctx, user)
	if err != nil {
		return nil, err
	}
	person.PublicKey.PublicKeyPem = publicKeyPem

	return person, nil
}
