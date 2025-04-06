// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package graceful

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAbstractUnixSocket(t *testing.T) {
	_, err := DefaultGetListener("unix", "@abc")
	require.NoError(t, err)
}
