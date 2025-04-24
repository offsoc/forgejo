// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"

	"forgejo.org/models/user"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/process"
	"forgejo.org/modules/queue"
)

type pendingQueueItem struct {
	Doer     *user.User
	InboxURL string
	Payload  []byte
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
	_, _, finished := process.GetManager().AddContext(graceful.GetManager().HammerContext(),
		fmt.Sprintf("Checking delivery eligibility for Activity via user[%d] (%s), to federated user[%s]", item.Doer.ID, item.Doer.Name, item.InboxURL))
	defer finished()

	// TODO: fix linting
	return deliveryQueue.Push(deliveryQueueItem{
		Doer:     item.Doer,
		Payload:  item.Payload,
		InboxURL: item.InboxURL,
	})
}
