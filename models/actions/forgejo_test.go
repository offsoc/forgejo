// SPDX-License-Identifier: MIT

package actions

import (
	"crypto/subtle"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_RegisterRunner_Token(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	labels := []string{}
	name := "runner"
	version := "v1.2.3"
	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, &labels, name, version)
	require.NoError(t, err)
	assert.Equal(t, name, runner.Name)

	assert.Equal(t, 1, subtle.ConstantTimeCompare([]byte(runner.TokenHash), []byte(auth_model.HashToken(token, runner.TokenSalt))), "the token cannot be verified with the same method as routers/api/actions/runner/interceptor.go as of 8228751c55d6a4263f0fec2932ca16181c09c97d")
}

// TestActions_RegisterRunner_TokenUpdate tests that a token's secret is updated
// when a runner already exists and RegisterRunner is called with a token
// parameter whose first 16 bytes match that record but where the last 24 bytes
// do not match.
func TestActions_RegisterRunner_TokenUpdate(t *testing.T) {
	const recordID = 12345678
	oldToken := "7e577e577e577e57feedfacefeedfacefeedface"
	newToken := "7e577e577e577e57deadbeefdeadbeefdeadbeef"
	require.NoError(t, unittest.PrepareTestDatabase())
	before := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	require.Equal(t,
		before.TokenHash, auth_model.HashToken(oldToken, before.TokenSalt),
		"the initial token should match the runner's secret",
	)

	RegisterRunner(db.DefaultContext, before.OwnerID, before.RepoID, newToken, nil, before.Name, before.Version)

	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	assert.Equal(t, before.UUID, after.UUID)
	assert.NotEqual(t,
		after.TokenHash, auth_model.HashToken(oldToken, after.TokenSalt),
		"the old token can still be verified",
	)
	assert.Equal(t,
		after.TokenHash, auth_model.HashToken(newToken, after.TokenSalt),
		"the new token cannot be verified",
	)
}

func TestActions_RegisterRunner_CreateWithLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	name := "runner"
	version := "v1.2.3"
	labels := []string{"woop", "doop"}
	labelsCopy := labels // labels may be affected by the tested function so we copy them

	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, &labels, name, version)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, ownerID, runner.OwnerID)
	assert.Equal(t, repoID, runner.RepoID)
	assert.Equal(t, name, runner.Name)
	assert.Equal(t, version, runner.Version)
	assert.Equal(t, labelsCopy, runner.AgentLabels)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: runner.ID})
	assert.Equal(t, ownerID, after.OwnerID)
	assert.Equal(t, repoID, after.RepoID)
	assert.Equal(t, name, after.Name)
	assert.Equal(t, version, after.Version)
	assert.Equal(t, labelsCopy, after.AgentLabels)
}

func TestActions_RegisterRunner_CreateWithoutLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	name := "runner"
	version := "v1.2.3"

	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, nil, name, version)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, ownerID, runner.OwnerID)
	assert.Equal(t, repoID, runner.RepoID)
	assert.Equal(t, name, runner.Name)
	assert.Equal(t, version, runner.Version)
	assert.Equal(t, []string{}, runner.AgentLabels)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: runner.ID})
	assert.Equal(t, ownerID, after.OwnerID)
	assert.Equal(t, repoID, after.RepoID)
	assert.Equal(t, name, after.Name)
	assert.Equal(t, version, after.Version)
	assert.Equal(t, []string{}, after.AgentLabels)
}

func TestActions_RegisterRunner_UpdateWithLabels(t *testing.T) {
	const recordID = 12345678
	token := "7e577e577e577e57feedfacefeedfacefeedface"
	require.NoError(t, unittest.PrepareTestDatabase())
	unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	newOwnerID := int64(1)
	newRepoID := int64(1)
	newName := "rennur"
	newVersion := "v4.5.6"
	newLabels := []string{"warp", "darp"}
	labelsCopy := newLabels // labels may be affected by the tested function so we copy them

	runner, err := RegisterRunner(db.DefaultContext, newOwnerID, newRepoID, token, &newLabels, newName, newVersion)
	require.NoError(t, err)

	// Check that the returned record has been updated
	assert.Equal(t, newOwnerID, runner.OwnerID)
	assert.Equal(t, newRepoID, runner.RepoID)
	assert.Equal(t, newName, runner.Name)
	assert.Equal(t, newVersion, runner.Version)
	assert.Equal(t, labelsCopy, runner.AgentLabels)

	// Check that whatever is in the DB has been updated
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	assert.Equal(t, newOwnerID, after.OwnerID)
	assert.Equal(t, newRepoID, after.RepoID)
	assert.Equal(t, newName, after.Name)
	assert.Equal(t, newVersion, after.Version)
	assert.Equal(t, labelsCopy, after.AgentLabels)
}

func TestActions_RegisterRunner_UpdateWithoutLabels(t *testing.T) {
	const recordID = 12345678
	token := "7e577e577e577e57feedfacefeedfacefeedface"
	require.NoError(t, unittest.PrepareTestDatabase())
	before := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	newOwnerID := int64(1)
	newRepoID := int64(1)
	newName := "rennur"
	newVersion := "v4.5.6"

	runner, err := RegisterRunner(db.DefaultContext, newOwnerID, newRepoID, token, nil, newName, newVersion)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, newOwnerID, runner.OwnerID)
	assert.Equal(t, newRepoID, runner.RepoID)
	assert.Equal(t, newName, runner.Name)
	assert.Equal(t, newVersion, runner.Version)
	assert.Equal(t, before.AgentLabels, runner.AgentLabels)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	assert.Equal(t, newOwnerID, after.OwnerID)
	assert.Equal(t, newRepoID, after.RepoID)
	assert.Equal(t, newName, after.Name)
	assert.Equal(t, newVersion, after.Version)
	assert.Equal(t, before.AgentLabels, after.AgentLabels)
}
