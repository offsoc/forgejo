// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package templates

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	asymkey_model "forgejo.org/models/asymkey"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pushCommits() *repository.PushCommits {
	pushCommits := repository.NewPushCommits()
	pushCommits.Commits = []*repository.PushCommit{
		{
			Sha1:           "x",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "invalid sha1",
		},
		{
			Sha1:           "2c54faec6c45d31c1abfaecdab471eac6633738a",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "not signed commit",
		},
		{
			Sha1:           "2d491b2985a7ff848d5c02748e7ea9f9f7619f9f",
			CommitterEmail: "non-existent",
			CommitterName:  "user2",
			AuthorEmail:    "non-existent",
			AuthorName:     "user2",
			Message:        "Using email that isn't known to Forgejo",
			Signature: &git.ObjectSignature{
				Payload: `tree 2d491b2985a7ff848d5c02748e7ea9f9f7619f9f
parent 45b03601635a1f463b81963a4022c7f87ce96ef9
author user2 <non-existent> 1699710556 +0100
committer user2 <non-existent> 1699710556 +0100

Using email that isn't known to Forgejo
`,
				Signature: `-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgoGSe9Zy7Ez9bSJcaTNjh/Y7p95
f5DujjqkpzFRtw6CEAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQIMufOuSjZeDUujrkVK4sl7ICa0WwEftas8UAYxx0Thdkiw2qWjR1U1PKfTLm16/w8
/bS1LX1lZNuzm2LR2qEgw=
-----END SSH SIGNATURE-----
`,
			},
		},
		{
			Sha1:           "853694aae8816094a0d875fee7ea26278dbf5d0f",
			CommitterEmail: "user2@example.com",
			CommitterName:  "user2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "user2",
			Message:        "Add content",
			Signature: &git.ObjectSignature{
				Payload: `tree 853694aae8816094a0d875fee7ea26278dbf5d0f
parent c2780d5c313da2a947eae22efd7dacf4213f4e7f
author user2 <user2@example.com> 1699707877 +0100
committer user2 <user2@example.com> 1699707877 +0100

Add content
`,
				Signature: `-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgoGSe9Zy7Ez9bSJcaTNjh/Y7p95
f5DujjqkpzFRtw6CEAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQBe2Fwk/FKY3SBCnG6jSYcO6ucyahp2SpQ/0P+otslzIHpWNW8cQ0fGLdhhaFynJXQ
fs9cMpZVM9BfIKNUSO8QY=
-----END SSH SIGNATURE-----
`,
			},
		},
	}
	return pushCommits
}

func TestActionContent2Commits_VerificationState(t *testing.T) {
	defer unittest.OverrideFixtures("models/fixtures/TestParseCommitWithSSHSignature/")()
	require.NoError(t, unittest.PrepareTestDatabase())
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user2.ID})
	commits, err := json.Marshal(pushCommits())
	require.NoError(t, err)

	act := &activities_model.Action{
		OpType:  activities_model.ActionCommitRepo,
		Repo:    repo,
		Content: string(commits),
	}
	push := ActionContent2Commits(t.Context(), act)

	assert.Equal(t, 4, push.Len)
	assert.False(t, push.Commits[0].Verification.Verified)
	assert.Empty(t, push.Commits[0].Verification.TrustStatus)
	assert.Equal(t, "git.error.invalid_commit_id", push.Commits[0].Verification.Reason)

	assert.False(t, push.Commits[1].Verification.Verified)
	assert.Empty(t, push.Commits[1].Verification.TrustStatus)
	assert.Equal(t, asymkey_model.NotSigned, push.Commits[1].Verification.Reason)

	assert.False(t, push.Commits[2].Verification.Verified)
	assert.Empty(t, push.Commits[2].Verification.TrustStatus)
	assert.Equal(t, asymkey_model.NoKeyFound, push.Commits[2].Verification.Reason)

	assert.True(t, push.Commits[3].Verification.Verified)
	assert.Equal(t, "user2 / SHA256:TKfwbZMR7e9OnlV2l1prfah1TXH8CmqR0PvFEXVCXA4", push.Commits[3].Verification.Reason)
	assert.Equal(t, "trusted", push.Commits[3].Verification.TrustStatus)
}
