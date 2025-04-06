// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	driver_options "forgejo.org/services/f3/driver/options"

	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	"code.forgejo.org/f3/gof3/v3/options"
	forge_test "code.forgejo.org/f3/gof3/v3/tree/tests/f3/forge"
)

type forgeTest struct {
	forge_test.Base
}

func (o *forgeTest) NewOptions(t *testing.T) options.Interface {
	return newTestOptions(t)
}

func (o *forgeTest) GetExceptions() []f3_kind.Kind {
	return []f3_kind.Kind{}
}

func (o *forgeTest) GetNonTestUsers() []string {
	return []string{
		"user1",
	}
}

func newForgeTest() forge_test.Interface {
	t := &forgeTest{}
	t.SetName(driver_options.Name)
	return t
}
