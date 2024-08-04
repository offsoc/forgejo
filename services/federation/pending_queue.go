// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/queue"
)

type pendingQueueItem struct {
	Doer            *user.User
	FederatedUserID int64
	Payload         []byte
}

var pendingQueue *queue.WorkerPoolQueue[pendingQueueItem]

func initPendingQueue() error {
	pendingQueue = queue.CreateUniqueQueue(graceful.GetManager().ShutdownContext(), "activitypub_pending_delivery", pendingQueueHandler)
	if pendingQueue == nil {
		return fmt.Errorf("unable to create activitypub_pending_delivery queue")
	}
	go graceful.GetManager().RunWithCancel(pendingQueue)

	return nil
}

func pendingQueueHandler(items ...pendingQueueItem) (unhandled []pendingQueueItem) {
	for _, item := range items {
		if err := handlePending(item); err != nil {
			unhandled = append(unhandled, item)
		}
	}
	return unhandled
}

func handlePending(item pendingQueueItem) error {
	ctx, _, finished := process.GetManager().AddContext(graceful.GetManager().HammerContext(),
		fmt.Sprintf("Checking delivery eligibility for Activity via user[%d] (%s), to federated user[%d]", item.Doer.ID, item.Doer.Name, item.FederatedUserID))
	defer finished()

	federatedUser, err := user.GetFederatedUserByID(ctx, item.FederatedUserID)
	if err != nil {
		return err
	}

	inbox := federatedUser.InboxURL

	// If we have no inbox, queue up an inbox refresh, and requeue via error
	if inbox == nil || *inbox == "" {
		if err := refreshQueue.Push(refreshQueueItem{
			FederatedUserID: item.FederatedUserID,
			Doer:            item.Doer,
		}); err != nil {
			return err
		}
		log.Debug("[follow] no inbox found for user[%d]", item.FederatedUserID)
		return fmt.Errorf("No Inbox URL found for federated user[%d]", item.FederatedUserID)
	}

	// If we have an inbox, queue it into delivery, and don't requeue here
	return deliveryQueue.Push(deliveryQueueItem{
		Doer:     item.Doer,
		Payload:  item.Payload,
		InboxURL: *inbox,
	})
}
