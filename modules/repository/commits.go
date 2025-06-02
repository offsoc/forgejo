// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"net/url"
	"time"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/avatars"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
)

// PushCommit represents a commit in a push operation.
type PushCommit struct {
	Sha1           string
	Message        string
	AuthorEmail    string
	AuthorName     string
	CommitterEmail string
	CommitterName  string
	Signature      *git.ObjectSignature
	Verification   *asymkey_model.ObjectVerification
	Timestamp      time.Time
}

// PushCommits represents list of commits in a push operation.
type PushCommits struct {
	Commits    []*PushCommit
	HeadCommit *PushCommit
	CompareURL string
	Len        int
}

// NewPushCommits creates a new PushCommits object.
func NewPushCommits() *PushCommits {
	return &PushCommits{}
}

// toAPIPayloadCommit converts a single PushCommit to an api.PayloadCommit object.
func (pc *PushCommits) toAPIPayloadCommit(ctx context.Context, emailUsers map[string]*user_model.User, repoPath, repoLink string, commit *PushCommit) (*api.PayloadCommit, error) {
	var err error
	authorUsername := ""
	author, ok := emailUsers[commit.AuthorEmail]
	if !ok {
		author, err = user_model.GetUserByEmail(ctx, commit.AuthorEmail)
		if err == nil {
			authorUsername = author.Name
			emailUsers[commit.AuthorEmail] = author
		}
	} else {
		authorUsername = author.Name
	}

	committerUsername := ""
	committer, ok := emailUsers[commit.CommitterEmail]
	if !ok {
		committer, err = user_model.GetUserByEmail(ctx, commit.CommitterEmail)
		if err == nil {
			// TODO: check errors other than email not found.
			committerUsername = committer.Name
			emailUsers[commit.CommitterEmail] = committer
		}
	} else {
		committerUsername = committer.Name
	}

	fileStatus, err := git.GetCommitFileStatus(ctx, repoPath, commit.Sha1)
	if err != nil {
		return nil, fmt.Errorf("FileStatus [commit_sha1: %s]: %w", commit.Sha1, err)
	}

	return &api.PayloadCommit{
		ID:      commit.Sha1,
		Message: commit.Message,
		URL:     fmt.Sprintf("%s/commit/%s", repoLink, url.PathEscape(commit.Sha1)),
		Author: &api.PayloadUser{
			Name:     commit.AuthorName,
			Email:    commit.AuthorEmail,
			UserName: authorUsername,
		},
		Committer: &api.PayloadUser{
			Name:     commit.CommitterName,
			Email:    commit.CommitterEmail,
			UserName: committerUsername,
		},
		Added:     fileStatus.Added,
		Removed:   fileStatus.Removed,
		Modified:  fileStatus.Modified,
		Timestamp: commit.Timestamp,
	}, nil
}

// ToAPIPayloadCommits converts a PushCommits object to api.PayloadCommit format.
// It returns all converted commits and, if provided, the head commit or an error otherwise.
func (pc *PushCommits) ToAPIPayloadCommits(ctx context.Context, repoPath, repoLink string) ([]*api.PayloadCommit, *api.PayloadCommit, error) {
	commits := make([]*api.PayloadCommit, len(pc.Commits))
	var headCommit *api.PayloadCommit

	emailUsers := make(map[string]*user_model.User)

	for i, commit := range pc.Commits {
		apiCommit, err := pc.toAPIPayloadCommit(ctx, emailUsers, repoPath, repoLink, commit)
		if err != nil {
			return nil, nil, err
		}

		commits[i] = apiCommit
		if pc.HeadCommit != nil && pc.HeadCommit.Sha1 == commits[i].ID {
			headCommit = apiCommit
		}
	}
	if pc.HeadCommit != nil && headCommit == nil {
		var err error
		headCommit, err = pc.toAPIPayloadCommit(ctx, emailUsers, repoPath, repoLink, pc.HeadCommit)
		if err != nil {
			return nil, nil, err
		}
	}
	return commits, headCommit, nil
}

// AvatarLink tries to match user in database with e-mail
// in order to show custom avatar, and falls back to general avatar link.
func (pc *PushCommits) AvatarLink(ctx context.Context, email string) string {
	size := avatars.DefaultAvatarPixelSize * setting.Avatar.RenderedSizeFactor

	v, _ := cache.GetWithContextCache(ctx, "push_commits", email, func() (string, error) {
		u, err := user_model.GetUserByEmail(ctx, email)
		if err != nil {
			if !user_model.IsErrUserNotExist(err) {
				log.Error("GetUserByEmail: %v", err)
				return "", err
			}
			return avatars.GenerateEmailAvatarFastLink(ctx, email, size), nil
		}
		return u.AvatarLinkWithSize(ctx, size), nil
	})

	return v
}

// PushCommitToCommit transforms a PushCommit to a git.Commit type on a best effort basis.
//
// Attention: Converting a Commit to a PushCommit and converting back to a Commit will not result in an identical object!
func PushCommitToCommit(commit *PushCommit) (*git.Commit, error) {
	id, err := git.NewIDFromString(commit.Sha1)
	if err != nil {
		return nil, err
	}
	return &git.Commit{
		ID: id,
		Author: &git.Signature{
			Name:  commit.AuthorName,
			Email: commit.AuthorEmail,
			When:  commit.Timestamp,
		},
		Committer: &git.Signature{
			Name:  commit.CommitterName,
			Email: commit.CommitterEmail,
			When:  commit.Timestamp,
		},
		CommitMessage: commit.Message,
		Signature:     commit.Signature,
		Parents:       []git.ObjectID{},
	}, nil
}

// CommitToPushCommit transforms a git.Commit to PushCommit type.
func CommitToPushCommit(commit *git.Commit) *PushCommit {
	return &PushCommit{
		Sha1:           commit.ID.String(),
		Message:        commit.Message(),
		AuthorEmail:    commit.Author.Email,
		AuthorName:     commit.Author.Name,
		CommitterEmail: commit.Committer.Email,
		CommitterName:  commit.Committer.Name,
		Signature:      commit.Signature,
		Timestamp:      commit.Author.When,
	}
}

// GitToPushCommits transforms a list of git.Commits to PushCommits type.
func GitToPushCommits(gitCommits []*git.Commit) *PushCommits {
	commits := make([]*PushCommit, 0, len(gitCommits))
	for _, commit := range gitCommits {
		commits = append(commits, CommitToPushCommit(commit))
	}
	return &PushCommits{
		Commits:    commits,
		HeadCommit: nil,
		CompareURL: "",
		Len:        len(commits),
	}
}
