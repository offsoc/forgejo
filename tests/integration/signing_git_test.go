// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestInstanceSigning(t *testing.T) {
	t.Cleanup(func() {
		// Cannot use t.Context(), it is in the done state.
		require.NoError(t, git.InitFull(context.Background())) //nolint:usetesting
	})

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Repository.Signing.SigningName, "UwU")()
		defer test.MockVariableValue(&setting.Repository.Signing.SigningEmail, "fox@example.com")()
		defer test.MockProtect(&setting.Repository.Signing.InitialCommit)()
		defer test.MockProtect(&setting.Repository.Signing.CRUDActions)()

		t.Run("SSH", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pubKeyContent, err := os.ReadFile("tests/integration/ssh-signing-key.pub")
			require.NoError(t, err)

			pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyContent)
			require.NoError(t, err)
			signingKeyPath, err := filepath.Abs("tests/integration/ssh-signing-key")
			require.NoError(t, err)
			require.NoError(t, os.Chmod(signingKeyPath, 0o600))
			defer test.MockVariableValue(&setting.SSHInstanceKey, pubKey)()
			defer test.MockVariableValue(&setting.Repository.Signing.Format, "ssh")()
			defer test.MockVariableValue(&setting.Repository.Signing.SigningKey, signingKeyPath)()

			// Ensure the git config is updated with the new signing format.
			require.NoError(t, git.InitFull(t.Context()))

			forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
				u2 := *u
				testCRUD(t, &u2, "ssh", objectFormat)
			})
		})

		t.Run("PGP", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Use a new GNUPGPHOME to avoid messing with the existing GPG keyring.
			tmpDir := t.TempDir()
			require.NoError(t, os.Chmod(tmpDir, 0o700))
			t.Setenv("GNUPGHOME", tmpDir)

			rootKeyPair, err := importTestingKey()
			require.NoError(t, err)
			defer test.MockVariableValue(&setting.Repository.Signing.SigningKey, rootKeyPair.PrimaryKey.KeyIdShortString())()
			defer test.MockVariableValue(&setting.Repository.Signing.Format, "openpgp")()

			// Ensure the git config is updated with the new signing format.
			require.NoError(t, git.InitFull(t.Context()))

			forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
				u2 := *u
				testCRUD(t, &u2, "pgp", objectFormat)
			})
		})
	})
}

func testCRUD(t *testing.T, u *url.URL, signingFormat string, objectFormat git.ObjectFormat) {
	t.Helper()
	setting.Repository.Signing.CRUDActions = []string{"never"}
	setting.Repository.Signing.InitialCommit = []string{"never"}

	username := "user2"
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
	baseAPITestContext := NewAPITestContext(t, username, "repo1")
	u.Path = baseAPITestContext.GitPath()

	suffix := "-" + signingFormat + "-" + objectFormat.Name()

	t.Run("Unsigned-Initial", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
		t.Run("CheckMasterBranchUnsigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
			assert.NotNil(t, branch.Commit)
			assert.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		}))
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "master", "never", "unsigned-never.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "never", "never2", "unsigned-never2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-ParentSigned", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "master", "parentsigned", "signed-parent.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned2", "signed-parent2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-Never", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.InitialCommit = []string{"never"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned-never", "unsigned-never2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Always", crudActionCreateFile(
			t, testCtx, user, "master", "always", "signed-always.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
		t.Run("CreateCRUDFile-ParentSigned-always", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned-always", "signed-parent2.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-ParentSigned", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Always-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "always", "always-parentsigned", "signed-always-parentsigned.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("AlwaysSign-Pubkey", func(t *testing.T) {
		setting.Repository.Signing.InitialCommit = []string{"pubkey"}

		t.Run("Has publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "initial-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
			t.Run("CheckMasterBranchSigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.True(t, branch.Commit.Verification.Verified)
				assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
			}))
		})

		t.Run("No publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-no-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
			t.Run("CheckMasterBranchSigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.False(t, branch.Commit.Verification.Verified)
			}))
		})
	})

	t.Run("AlwaysSign-Twofa", func(t *testing.T) {
		setting.Repository.Signing.InitialCommit = []string{"twofa"}

		t.Run("Has 2fa", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Cleanup(func() {
				unittest.AssertSuccessfulDelete(t, &auth_model.WebAuthnCredential{UserID: user.ID})
			})

			testCtx := NewAPITestContext(t, username, "initial-2fa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			unittest.AssertSuccessfulInsert(t, &auth_model.WebAuthnCredential{UserID: user.ID})

			t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
			t.Run("CheckMasterBranchSigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.True(t, branch.Commit.Verification.Verified)
				assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
			}))
		})

		t.Run("No publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-no-2fa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
			t.Run("CheckMasterBranchSigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.False(t, branch.Commit.Verification.Verified)
			}))
		})
	})

	t.Run("AlwaysSign-Initial", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.InitialCommit = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-always"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
		t.Run("CheckMasterBranchSigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.True(t, branch.Commit.Verification.Verified)
			assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
		}))
	})

	t.Run("AlwaysSign-Initial-CRUD-Never", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"never"}

		testCtx := NewAPITestContext(t, username, "initial-always-never"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "master", "never", "unsigned-never.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("AlwaysSign-Initial-CRUD-ParentSigned-On-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-always-parent"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "master", "parentsigned", "signed-parent.txt", func(t *testing.T, response api.FileResponse) {
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("AlwaysSign-Initial-CRUD-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-always-always"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, false, objectFormat))
		t.Run("CreateCRUDFile-Always", crudActionCreateFile(
			t, testCtx, user, "master", "always", "signed-always.txt", func(t *testing.T, response api.FileResponse) {
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("UnsignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"commitssigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "never2")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		}))
	})

	t.Run("BaseSignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"basesigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "parentsigned2")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		}))
	})

	t.Run("CommitsSignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"commitssigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "always-parentsigned")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", doAPIGetBranch(testCtx, "master", func(t *testing.T, branch api.Branch) {
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.True(t, branch.Commit.Verification.Verified)
		}))
	})
}

func crudActionCreateFile(_ *testing.T, ctx APITestContext, user *user_model.User, from, to, path string, callback ...func(*testing.T, api.FileResponse)) func(*testing.T) {
	return doAPICreateFile(ctx, path, &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    from,
			NewBranchName: to,
			Message:       fmt.Sprintf("from:%s to:%s path:%s", from, to, path),
			Author: api.Identity{
				Name:  user.FullName,
				Email: user.Email,
			},
			Committer: api.Identity{
				Name:  user.FullName,
				Email: user.Email,
			},
		},
		ContentBase64: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("This is new text for %s", path))),
	}, callback...)
}

func importTestingKey() (*openpgp.Entity, error) {
	if _, _, err := process.GetManager().Exec("gpg --import tests/integration/private-testing.key", "gpg", "--import", "tests/integration/private-testing.key"); err != nil {
		return nil, err
	}
	keyringFile, err := os.Open("tests/integration/private-testing.key")
	if err != nil {
		return nil, err
	}
	defer keyringFile.Close()

	block, err := armor.Decode(keyringFile)
	if err != nil {
		return nil, err
	}

	keyring, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("Keyring access failed: '%w'", err)
	}

	// There should only be one entity in this file.
	return keyring[0], nil
}
