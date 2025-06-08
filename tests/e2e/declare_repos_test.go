// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/indexer/stats"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"
	issue_service "forgejo.org/services/issue"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/convert"
)

// first entry represents filename
// the following entries define the full file content over time
type FileChanges struct {
	Filename  string
	CommitMsg string
	Versions  []string
}

// performs additional repo setup as needed
type SetupRepo func(*user_model.User, *repo_model.Repository)

// put your Git repo declarations in here
// feel free to amend the helper function below or use the raw variant directly
func DeclareGitRepos(t *testing.T) func() {
	cleanupFunctions := []func(){
		newRepo(t, 2, "diff-test", nil, []FileChanges{{
			Filename: "testfile",
			Versions: []string{"hello", "hallo", "hola", "native", "ubuntu-latest", "- runs-on: ubuntu-latest", "- runs-on: debian-latest"},
		}}, nil),
		newRepo(t, 2, "language-stats-test", nil, []FileChanges{{
			Filename: "main.rs",
			Versions: []string{"fn main() {", "println!(\"Hello World!\");", "}"},
		}}, nil),
		newRepo(t, 2, "mentions-highlighted", nil, []FileChanges{
			{
				Filename:  "history1.md",
				Versions:  []string{""},
				CommitMsg: "A commit message which mentions @user2 in the title\nand has some additional text which mentions @user1",
			},
			{
				Filename:  "history2.md",
				Versions:  []string{""},
				CommitMsg: "Another commit which mentions @user1 in the title\nand @user2 in the text",
			},
		}, nil),
		newRepo(t, 2, "unicode-escaping", nil, []FileChanges{{
			Filename: "a-file",
			Versions: []string{"{a}{а}"},
		}}, nil),
		newRepo(t, 11, "dependency-test", &tests.DeclarativeRepoOptions{
			UnitConfig: optional.Some(map[unit_model.Type]convert.Conversion{
				unit_model.TypeIssues: &repo_model.IssuesConfig{
					EnableDependencies: true,
				},
			}),
		}, []FileChanges{}, func(user *user_model.User, repo *repo_model.Repository) {
			now := timeutil.TimeStampNow() - 3600
			post := func(title, content string) {
				issue := &issues_model.Issue{
					RepoID:      repo.ID,
					PosterID:    user.ID,
					Title:       title,
					Content:     content,
					CreatedUnix: now - 300,
				}
				require.NoError(t, issue_service.NewIssue(db.DefaultContext, repo, issue, nil, nil, nil))
			}
			post("first issue here", "an issue created earlier")
			post("second issue here (not 101)", "not the seventh issue, but in the right repo")
			post("third issue here", "depends on things")
		}),
		// add your repo declarations here
	}

	return func() {
		for _, cleanup := range cleanupFunctions {
			cleanup()
		}
	}
}

func newRepo(t *testing.T, userID int64, repoName string, initOpts *tests.DeclarativeRepoOptions, fileChanges []FileChanges, setup SetupRepo) func() {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})

	opts := tests.DeclarativeRepoOptions{}
	if initOpts != nil {
		opts = *initOpts
	}
	opts.Name = optional.Some(repoName)
	if !opts.EnabledUnits.Has() {
		opts.EnabledUnits = optional.Some([]unit_model.Type{unit_model.TypeCode, unit_model.TypeIssues})
	}
	somerepo, _, cleanupFunc := tests.CreateDeclarativeRepoWithOptions(t, user, opts)

	for _, file := range fileChanges {
		for i, version := range file.Versions {
			operation := "update"
			if i == 0 {
				operation = "create"
			}

			// default to unique commit messages
			commitMsg := file.CommitMsg
			if commitMsg == "" {
				commitMsg = fmt.Sprintf("Patch: %s-%d", file.Filename, i+1)
			}

			resp, err := files_service.ChangeRepoFiles(git.DefaultContext, somerepo, user, &files_service.ChangeRepoFilesOptions{
				Files: []*files_service.ChangeRepoFile{{
					Operation:     operation,
					TreePath:      file.Filename,
					ContentReader: strings.NewReader(version),
				}},
				Message:   commitMsg,
				OldBranch: "main",
				NewBranch: "main",
				Author: &files_service.IdentityOptions{
					Name:  user.Name,
					Email: user.Email,
				},
				Committer: &files_service.IdentityOptions{
					Name:  user.Name,
					Email: user.Email,
				},
				Dates: &files_service.CommitDateOptions{
					Author:    time.Now(),
					Committer: time.Now(),
				},
			})
			require.NoError(t, err)
			assert.NotEmpty(t, resp)
		}
	}

	if setup != nil {
		setup(user, somerepo)
	}

	err := stats.UpdateRepoIndexer(somerepo)
	require.NoError(t, err)

	return cleanupFunc
}
