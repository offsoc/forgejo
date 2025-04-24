// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/queue"
)

type refreshQueueItem struct {
	Doer                    *user.User
	FederationHostID        int64
	FederatedUserExternalID string
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
		fmt.Sprintf("Refreshing FederatedUser.ExternalID[%v] on FederationHost[%v]", item.FederatedUserExternalID, item.FederationHostID))
	defer finished()

	federationHost, err := forgefed.GetFederationHost(ctx, item.FederationHostID)
	if err != nil {
		log.Error("GetFederationHost: %v", err)
		return err
	}
	_, federatedUser, err := user.GetFederatedUser(ctx, item.FederatedUserExternalID, federationHost.ID)
	if err != nil {
		log.Error("FindFederatedUser: ", err)
		return err
	}

	personID, err := fm.NewPersonIDFromModel(
		federationHost.HostFqdn, federationHost.HostSchema,
		federationHost.HostPort, string(federationHost.NodeInfo.SoftwareName),
		federatedUser.ExternalID,
	)
	if err != nil {
		return err
	}
	_, refetchFederatedUser, err := fetchUserFromAP(ctx, personID, federatedUser.FederationHostID)
	if err != nil {
		return err
	}

	if federatedUser.InboxPath != refetchFederatedUser.InboxPath {
		federatedUser.InboxPath = refetchFederatedUser.InboxPath
		if err := federatedUser.UpdateFederatedUser(ctx); err != nil {
			return err
		}
	}
	return nil
}
