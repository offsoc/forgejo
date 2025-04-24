// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/services/convert"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

func SendUserActivity(ctx context.Context, doer *user.User, activity *activities_model.Action) error {
	followers, err := user.GetFollowersForUser(ctx, doer)
	if err != nil {
		return err
	}

	userActivity, err := convert.ActionToForgeUserActivity(ctx, activity)
	if err != nil {
		return err
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(ap.ActivityBaseURI),
	).Marshal(userActivity)
	if err != nil {
		return err
	}

	for _, follower := range followers {
		// TODO: switch to user id instead of federatedUserID
		federatedUserFollower, err := user.GetFederatedUserByID(ctx, follower.FollowingUserID)
		if err != nil {
			return err
		}

		federationHost, err := forgefed.GetFederationHost(ctx, federatedUserFollower.FederationHostID)
		if err != nil {
			return err
		}

		hostURL := federationHost.AsURL()
		if err := pendingQueue.Push(pendingQueueItem{
			InboxURL: hostURL.JoinPath(federatedUserFollower.InboxPath).String(),
			Doer:     doer,
			Payload:  payload,
		}); err != nil {
			return err
		}
	}

	return nil
}

func NotifyActivityPubFollowers(ctx context.Context, actions []activities_model.Action) error {
	if !setting.Federation.Enabled {
		return nil
	}
	for _, act := range actions {
		if act.Repo != nil {
			if act.Repo.IsPrivate {
				continue
			}
			if act.Repo.Owner.KeepActivityPrivate || act.Repo.Owner.Visibility != structs.VisibleTypePublic {
				continue
			}
		}
		if act.ActUser.KeepActivityPrivate || act.ActUser.Visibility != structs.VisibleTypePublic {
			continue
		}
		if err := SendUserActivity(ctx, act.ActUser, &act); err != nil {
			return err
		}
	}
	return nil
}
