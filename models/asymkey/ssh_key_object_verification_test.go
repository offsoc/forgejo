// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"os"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestParseCommitWithSSHSignature(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	sshKey := unittest.AssertExistsAndLoadBean(t, &PublicKey{ID: 1000, OwnerID: 2})

	t.Run("No commiter", func(t *testing.T) {
		o := commitToGitObject(&git.Commit{})
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, &user_model.User{})
		assert.False(t, commitVerification.Verified)
		assert.Equal(t, NoKeyFound, commitVerification.Reason)
	})

	t.Run("Commiter without keys", func(t *testing.T) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		o := commitToGitObject(&git.Commit{Committer: &git.Signature{Email: user.Email}})
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user)
		assert.False(t, commitVerification.Verified)
		assert.Equal(t, NoKeyFound, commitVerification.Reason)
	})

	t.Run("Correct signature with wrong email", func(t *testing.T) {
		gitCommit := &git.Commit{
			Committer: &git.Signature{
				Email: "non-existent",
			},
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
		}
		o := commitToGitObject(gitCommit)
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user2)
		assert.False(t, commitVerification.Verified)
		assert.Equal(t, NoKeyFound, commitVerification.Reason)
	})

	t.Run("Incorrect signature with correct email", func(t *testing.T) {
		gitCommit := &git.Commit{
			Committer: &git.Signature{
				Email: "user2@example.com",
			},
			Signature: &git.ObjectSignature{
				Payload: `tree 853694aae8816094a0d875fee7ea26278dbf5d0f
parent c2780d5c313da2a947eae22efd7dacf4213f4e7f
author user2 <user2@example.com> 1699707877 +0100
committer user2 <user2@example.com> 1699707877 +0100

Add content
`,
				Signature: `-----BEGIN SSH SIGNATURE-----`,
			},
		}

		o := commitToGitObject(gitCommit)
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user2)
		assert.False(t, commitVerification.Verified)
		assert.Equal(t, NoKeyFound, commitVerification.Reason)
	})

	t.Run("Valid signature with correct email", func(t *testing.T) {
		gitCommit := &git.Commit{
			Committer: &git.Signature{
				Email: "user2@example.com",
			},
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
		}

		o := commitToGitObject(gitCommit)
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user2)
		assert.True(t, commitVerification.Verified)
		assert.Equal(t, "user2 / SHA256:TKfwbZMR7e9OnlV2l1prfah1TXH8CmqR0PvFEXVCXA4", commitVerification.Reason)
		assert.Equal(t, sshKey, commitVerification.SigningSSHKey)
	})

	t.Run("Valid signature with noreply email", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.NoReplyAddress, "noreply.example.com")()

		gitCommit := &git.Commit{
			Committer: &git.Signature{
				Email: "user2@noreply.example.com",
			},
			Signature: &git.ObjectSignature{
				Payload: `tree 4836c7f639f37388bab4050ef5c97bbbd54272fc
parent 795be1b0117ea5c65456050bb9fd84744d4fd9c6
author user2 <user2@noreply.example.com> 1699709594 +0100
committer user2 <user2@noreply.example.com> 1699709594 +0100

Commit with noreply
`,
				Signature: `-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgoGSe9Zy7Ez9bSJcaTNjh/Y7p95
f5DujjqkpzFRtw6CEAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQJz83KKxD6Bz/ZvNpqkA3RPOSQ4LQ5FfEItbtoONkbwV9wAWMnmBqgggo/lnXCJ3oq
muPLbvEduU+Ze/1Ol1pgk=
-----END SSH SIGNATURE-----
`,
			},
		}

		o := commitToGitObject(gitCommit)
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user2)
		assert.True(t, commitVerification.Verified)
		assert.Equal(t, "user2 / SHA256:TKfwbZMR7e9OnlV2l1prfah1TXH8CmqR0PvFEXVCXA4", commitVerification.Reason)
		assert.Equal(t, sshKey, commitVerification.SigningSSHKey)
	})

	t.Run("Instance key", func(t *testing.T) {
		pubKeyContent, err := os.ReadFile("../../tests/integration/ssh-signing-key.pub")
		require.NoError(t, err)
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyContent)
		require.NoError(t, err)

		defer test.MockVariableValue(&setting.Repository.Signing.SigningName, "UwU")()
		defer test.MockVariableValue(&setting.Repository.Signing.SigningEmail, "fox@example.com")()
		defer test.MockVariableValue(&setting.SSHInstanceKey, pubKey)()

		gitCommit := &git.Commit{
			Committer: &git.Signature{
				Email: "fox@example.com",
			},
			Signature: &git.ObjectSignature{
				Payload: `tree f96f1a4f1a51dc42e2983592f503980b60b8849c
parent 93f84db542dd8c6e952c8130bc2fcbe2e299b8b4
author OwO <instance@example.com> 1738961379 +0100
committer UwU <fox@example.com> 1738961379 +0100

Fox
`,
				Signature: `-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgV5ELwZ8XJe2LLR/UTuEu/vsFdb
t7ry0W8hyzz/b1iocAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQCnyMRkWVVNoZxZkvi/ZoknUhs4LNBmEwZs9e9214WIt+mhKfc6BiHoE2qeluR2McD
Y5RzHnA8Ke9wXddEePCQE=
-----END SSH SIGNATURE-----
`,
			},
		}

		o := commitToGitObject(gitCommit)
		commitVerification := ParseObjectWithSSHSignature(db.DefaultContext, &o, user2)
		assert.True(t, commitVerification.Verified)
		assert.Equal(t, "UwU / SHA256:QttK41r/zMUeAW71b5UgVSb8xGFF/DlZJ6TyADW+uoI", commitVerification.Reason)
		assert.Equal(t, "SHA256:QttK41r/zMUeAW71b5UgVSb8xGFF/DlZJ6TyADW+uoI", commitVerification.SigningSSHKey.Fingerprint)
	})
}
