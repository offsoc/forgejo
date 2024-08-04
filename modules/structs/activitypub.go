// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// ActivityPub type
type ActivityPub struct {
	Context string `json:"@context"`
}

type APRemoteFollowOption struct {
	Target string `json:"target"`
}

type APPersonFollowItem struct {
	ActorID string `json:"actor_id"`
	Note    string `json:"note"`

	OriginalURL  string `json:"original_url"`
	OriginalItem string `json:"original_item"`
}
