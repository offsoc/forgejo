// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/queue"
)

type refreshQueueItem struct {
	Doer            *user.User
	FederatedUserID int64
}

var refreshQueue *queue.WorkerPoolQueue[refreshQueueItem]

func initRefreshQueue() error {
	refreshQueue = queue.CreateUniqueQueue(graceful.GetManager().ShutdownContext(), "activitypub_user_data_refresh", refreshQueueHandler)
	if refreshQueue == nil {
		return fmt.Errorf("unable to create activitypub_user_data_refresh queue")
	}
	go graceful.GetManager().RunWithCancel(refreshQueue)

	return nil
}

func refreshQueueHandler(items ...refreshQueueItem) (unhandled []refreshQueueItem) {
	for _, item := range items {
		if err := refreshSingleItem(item); err != nil {
			unhandled = append(unhandled, item)
		}
	}
	return unhandled
}

func refreshSingleItem(item refreshQueueItem) error {
	ctx, _, finished := process.GetManager().AddContext(graceful.GetManager().HammerContext(),
		fmt.Sprintf("Refreshing IndexURL for federated user[%d], via user[%d]", item.FederatedUserID, item.Doer.ID))
	defer finished()

	federatedUser, err := user.GetFederatedUserByID(ctx, item.FederatedUserID)
	if err != nil {
		log.Error("GetFederatedUserByID: %v", err)
		return err
	}

	localUser, err := user.GetUserByID(ctx, federatedUser.UserID)
	if err != nil {
		log.Error("GetUserByID: %v", err)
		return err
	}

	if localUser.NormalizedFederatedURI == "" {
		return fmt.Errorf("Federated user[%d] (user[%d]) has no NormalizedFederatedURI", item.FederatedUserID, localUser.ID)
	}

	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return err
	}
	client, err := clientFactory.WithKeys(ctx, item.Doer, item.Doer.APActorID()+"#main-key")
	if err != nil {
		return err
	}

	body, err := client.GetBody(localUser.NormalizedFederatedURI)
	if err != nil {
		return err
	}

	type personWithInbox struct {
		Inbox string `json:"inbox"`
	}
	var payload personWithInbox
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}

	return federatedUser.SetInboxURL(ctx, &payload.Inbox)
}
