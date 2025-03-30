// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchRuleMatch(t *testing.T) {
	kases := []struct {
		Rule          string
		BranchName    string
		ExpectedMatch bool
	}{
		{
			Rule:          "release/*",
			BranchName:    "release/v1.17",
			ExpectedMatch: true,
		},
		{
			Rule:          "release/**/v1.17",
			BranchName:    "release/test/v1.17",
			ExpectedMatch: true,
		},
		{
			Rule:          "release/**/v1.17",
			BranchName:    "release/test/1/v1.17",
			ExpectedMatch: true,
		},
		{
			Rule:          "release/*/v1.17",
			BranchName:    "release/test/1/v1.17",
			ExpectedMatch: false,
		},
		{
			Rule:          "release/v*",
			BranchName:    "release/v1.16",
			ExpectedMatch: true,
		},
		{
			Rule:          "*",
			BranchName:    "release/v1.16",
			ExpectedMatch: false,
		},
		{
			Rule:          "**",
			BranchName:    "release/v1.16",
			ExpectedMatch: true,
		},
		{
			Rule:          "main",
			BranchName:    "main",
			ExpectedMatch: true,
		},
		{
			Rule:          "master",
			BranchName:    "main",
			ExpectedMatch: false,
		},
	}

	for _, kase := range kases {
		pb := ProtectedBranch{RuleName: kase.Rule}
		assert.Equal(t, kase.ExpectedMatch, pb.Match(kase.BranchName), "%s - %s", kase.BranchName, kase.Rule)
	}
}
