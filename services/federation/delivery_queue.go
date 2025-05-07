// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"io"

	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/queue"
)

type deliveryQueueItem struct {
	Doer          *user.User
	InboxURL      string
	Payload       []byte
	DeliveryCount int
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
		item.DeliveryCount++
		err := deliverToInbox(item)
		if err != nil && item.DeliveryCount < 10 {
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

	log.Debug("Delivering %s to %s", item.Payload, item.InboxURL)
	res, err := apclient.Post(item.Payload, item.InboxURL)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 16*1024))

		log.Warn("Delivering to %s failed: %d %s, %v times", item.InboxURL, res.StatusCode, string(body), item.DeliveryCount)
		return fmt.Errorf("Delivery failed")
	}

	return nil
}
