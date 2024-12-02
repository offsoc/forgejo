// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"context"
)

type GistList []*Gist

func (gistList GistList) LoadOwner(ctx context.Context) error {
	for _, gist := range gistList {
		err := gist.LoadOwner(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
