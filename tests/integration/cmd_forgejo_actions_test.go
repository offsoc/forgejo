// SPDX-License-Identifier: MIT

package integration

import (
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CmdForgejo_Actions(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		token, err := runMainApp("forgejo-cli", "actions", "generate-runner-token")
		require.NoError(t, err)
		assert.Len(t, token, 40)

		secret, err := runMainApp("forgejo-cli", "actions", "generate-secret")
		require.NoError(t, err)
		assert.Len(t, secret, 40)

		_, err = runMainApp("forgejo-cli", "actions", "register")
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Contains(t, string(exitErr.Stderr), "at least one of the --secret")

		for _, testCase := range []struct {
			testName     string
			scope        string
			secret       string
			errorMessage string
		}{
			{
				testName:     "bad user",
				scope:        "baduser",
				secret:       "0123456789012345678901234567890123456789",
				errorMessage: "user does not exist",
			},
			{
				testName:     "bad repo",
				scope:        "org25/badrepo",
				secret:       "0123456789012345678901234567890123456789",
				errorMessage: "repository does not exist",
			},
			{
				testName:     "secret length != 40",
				scope:        "org25",
				secret:       "0123456789",
				errorMessage: "40 characters long",
			},
			{
				testName:     "secret is not a hexadecimal string",
				scope:        "org25",
				secret:       "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
				errorMessage: "must be an hexadecimal string",
			},
		} {
			t.Run(testCase.testName, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				output, err := runMainApp("forgejo-cli", "actions", "register", "--secret", testCase.secret, "--scope", testCase.scope)
				assert.EqualValues(t, "", output)

				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				assert.Contains(t, string(exitErr.Stderr), testCase.errorMessage)
			})
		}

		secret = "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"
		expecteduuid := "44444444-4444-4444-4444-444444444444"

		for _, testCase := range []struct {
			testName     string
			secretOption func() string
			stdin        io.Reader
		}{
			{
				testName: "secret from argument",
				secretOption: func() string {
					return "--secret=" + secret
				},
			},
			{
				testName: "secret from stdin",
				secretOption: func() string {
					return "--secret-stdin"
				},
				stdin: strings.NewReader(secret),
			},
			{
				testName: "secret from file",
				secretOption: func() string {
					secretFile := t.TempDir() + "/secret"
					require.NoError(t, os.WriteFile(secretFile, []byte(secret), 0o644))
					return "--secret-file=" + secretFile
				},
			},
		} {
			t.Run(testCase.testName, func(t *testing.T) {
				uuid, err := runMainAppWithStdin(testCase.stdin, "forgejo-cli", "actions", "register", testCase.secretOption(), "--scope=org26")
				require.NoError(t, err)
				assert.EqualValues(t, expecteduuid, uuid)
			})
		}

		secret = "0123456789012345678901234567890123456789"
		expecteduuid = "30313233-3435-3637-3839-303132333435"

		for _, testCase := range []struct {
			testName string
			scope    string
			secret   string
			name     string
			labels   string
			version  string
			uuid     string
		}{
			{
				testName: "org",
				scope:    "org25",
				secret:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				uuid:     "41414141-4141-4141-4141-414141414141",
			},
			{
				testName: "user and repo",
				scope:    "user2/repo2",
				secret:   "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
				uuid:     "42424242-4242-4242-4242-424242424242",
			},
			{
				testName: "labels",
				scope:    "org25",
				name:     "runnerName",
				labels:   "label1,label2,label3",
				version:  "v1.2.3",
				secret:   "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
				uuid:     "43434343-4343-4343-4343-434343434343",
			},
			{
				testName: "insert a runner",
				scope:    "user10/repo6",
				name:     "runnerName",
				labels:   "label1,label2,label3",
				version:  "v1.2.3",
				secret:   secret,
				uuid:     expecteduuid,
			},
			{
				testName: "update an existing runner",
				scope:    "user5/repo4",
				name:     "runnerNameChanged",
				labels:   "label1,label2,label3,more,label",
				version:  "v1.2.3-suffix",
				secret:   secret,
				uuid:     expecteduuid,
			},
		} {
			t.Run(testCase.testName, func(t *testing.T) {
				cmd := []string{
					"actions", "register",
					"--secret", testCase.secret, "--scope", testCase.scope,
				}
				if testCase.name != "" {
					cmd = append(cmd, "--name", testCase.name)
				}
				if testCase.labels != "" {
					cmd = append(cmd, "--labels", testCase.labels)
				}
				if testCase.version != "" {
					cmd = append(cmd, "--version", testCase.version)
				}
				//
				// Run twice to verify it is idempotent
				//
				for i := 0; i < 2; i++ {
					uuid, err := runMainApp("forgejo-cli", cmd...)
					require.NoError(t, err)
					if assert.EqualValues(t, testCase.uuid, uuid) {
						ownerName, repoName, found := strings.Cut(testCase.scope, "/")
						action, err := actions_model.GetRunnerByUUID(t.Context(), uuid)
						require.NoError(t, err)

						user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: action.OwnerID})
						assert.Equal(t, ownerName, user.Name, action.OwnerID)

						if found {
							repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: action.RepoID})
							assert.Equal(t, repoName, repo.Name, action.RepoID)
						}
						if testCase.name != "" {
							assert.EqualValues(t, testCase.name, action.Name)
						}
						if testCase.labels != "" {
							labels := strings.Split(testCase.labels, ",")
							assert.EqualValues(t, labels, action.AgentLabels)
						}
						if testCase.version != "" {
							assert.EqualValues(t, testCase.version, action.Version)
						}
					}
				}
			})
		}
	})
}
