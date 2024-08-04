// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"io"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/queue"
)

type deliveryQueueItem struct {
	Doer     *user.User
	InboxURL string
	Payload  []byte
}

var deliveryQueue *queue.WorkerPoolQueue[deliveryQueueItem]

func initDeliveryQueue() error {
	deliveryQueue = queue.CreateUniqueQueue(graceful.GetManager().ShutdownContext(), "activitypub_inbox_delivery", deliveryQueueHandler)
	if deliveryQueue == nil {
		return fmt.Errorf("unable to create activitypub_inbox_delivery queue")
	}
	go graceful.GetManager().RunWithCancel(deliveryQueue)

	return nil
}

func deliveryQueueHandler(items ...deliveryQueueItem) (unhandled []deliveryQueueItem) {
	for _, item := range items {
		if err := deliverToInbox(item); err != nil {
			unhandled = append(unhandled, item)
		}
	}
	return unhandled
}

func deliverToInbox(item deliveryQueueItem) error {
	ctx, _, finished := process.GetManager().AddContext(graceful.GetManager().HammerContext(),
		fmt.Sprintf("Delivering an Activity via user[%d] (%s), to %s", item.Doer.ID, item.Doer.Name, item.InboxURL))
	defer finished()

	clientFactory, err := activitypub.GetClientFactory(ctx)
	if err != nil {
		return err
	}
	apclient, err := clientFactory.WithKeys(ctx, item.Doer, item.Doer.APActorID()+"#main-key")
	if err != nil {
		return err
	}

	res, err := apclient.Post(item.Payload, item.InboxURL)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)

		log.Warn("Delivering to %s failed: %d %s", item.InboxURL, res.StatusCode, string(body))
		return fmt.Errorf("Delivery failed")
	}

	return nil
}
