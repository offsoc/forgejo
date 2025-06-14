// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"forgejo.org/models"
	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	"forgejo.org/services/attachment"
	notify_service "forgejo.org/services/notify"
)

type AttachmentChange struct {
	Action      string // "add", "delete", "update
	Type        string // "attachment", "external"
	UUID        string
	Name        string
	ExternalURL string
}

func createTag(ctx context.Context, gitRepo *git.Repository, rel *repo_model.Release, msg string) (bool, error) {
	err := rel.LoadAttributes(ctx)
	if err != nil {
		return false, err
	}

	err = rel.Repo.MustNotBeArchived()
	if err != nil {
		return false, err
	}

	var created bool
	// Only actual create when publish.
	if !rel.IsDraft {
		if !gitRepo.IsTagExist(rel.TagName) {
			if err := rel.LoadAttributes(ctx); err != nil {
				log.Error("LoadAttributes: %v", err)
				return false, err
			}

			protectedTags, err := git_model.GetProtectedTags(ctx, rel.Repo.ID)
			if err != nil {
				return false, fmt.Errorf("GetProtectedTags: %w", err)
			}

			// Trim '--' prefix to prevent command line argument vulnerability.
			rel.TagName = strings.TrimPrefix(rel.TagName, "--")
			isAllowed, err := git_model.IsUserAllowedToControlTag(ctx, protectedTags, rel.TagName, rel.PublisherID)
			if err != nil {
				return false, err
			}
			if !isAllowed {
				return false, models.ErrProtectedTagName{
					TagName: rel.TagName,
				}
			}

			commit, err := gitRepo.GetCommit(rel.Target)
			if err != nil {
				return false, err
			}

			if len(msg) > 0 {
				if err = gitRepo.CreateAnnotatedTag(rel.TagName, msg, commit.ID.String()); err != nil {
					if strings.Contains(err.Error(), "is not a valid tag name") {
						return false, models.ErrInvalidTagName{
							TagName: rel.TagName,
						}
					}
					return false, err
				}
			} else if err = gitRepo.CreateTag(rel.TagName, commit.ID.String()); err != nil {
				if strings.Contains(err.Error(), "is not a valid tag name") {
					return false, models.ErrInvalidTagName{
						TagName: rel.TagName,
					}
				}
				return false, err
			}
			created = true
			rel.LowerTagName = strings.ToLower(rel.TagName)

			objectFormat := git.ObjectFormatFromName(rel.Repo.ObjectFormatName)
			commits := repository.NewPushCommits()
			commits.HeadCommit = repository.CommitToPushCommit(commit)
			commits.CompareURL = rel.Repo.ComposeCompareURL(objectFormat.EmptyObjectID().String(), commit.ID.String())

			refFullName := git.RefNameFromTag(rel.TagName)
			notify_service.PushCommits(
				ctx, rel.Publisher, rel.Repo,
				&repository.PushUpdateOptions{
					RefFullName: refFullName,
					OldCommitID: objectFormat.EmptyObjectID().String(),
					NewCommitID: commit.ID.String(),
				}, commits)
			notify_service.CreateRef(ctx, rel.Publisher, rel.Repo, refFullName, commit.ID.String())
			rel.CreatedUnix = timeutil.TimeStampNow()
		}
		commit, err := gitRepo.GetTagCommit(rel.TagName)
		if err != nil {
			return false, fmt.Errorf("GetTagCommit: %w", err)
		}

		rel.Sha1 = commit.ID.String()
		rel.NumCommits, err = commit.CommitsCount()
		if err != nil {
			return false, fmt.Errorf("CommitsCount: %w", err)
		}

		if rel.PublisherID <= 0 {
			u, err := user_model.GetUserByEmail(ctx, commit.Author.Email)
			if err == nil {
				rel.PublisherID = u.ID
			}
		}
	} else {
		rel.CreatedUnix = timeutil.TimeStampNow()
	}
	return created, nil
}

// CreateRelease creates a new release of repository.
func CreateRelease(gitRepo *git.Repository, rel *repo_model.Release, msg string, attachmentChanges []*AttachmentChange) error {
	has, err := repo_model.IsReleaseExist(gitRepo.Ctx, rel.RepoID, rel.TagName)
	if err != nil {
		return err
	} else if has {
		return repo_model.ErrReleaseAlreadyExist{
			TagName: rel.TagName,
		}
	}

	if _, err = createTag(gitRepo.Ctx, gitRepo, rel, msg); err != nil {
		return err
	}

	rel.Title, _ = util.SplitStringAtByteN(rel.Title, 255)
	rel.LowerTagName = strings.ToLower(rel.TagName)
	if err = db.Insert(gitRepo.Ctx, rel); err != nil {
		return err
	}

	addAttachmentUUIDs := make(container.Set[string])

	for _, attachmentChange := range attachmentChanges {
		if attachmentChange.Action != "add" {
			return errors.New("can only create new attachments when creating release")
		}
		switch attachmentChange.Type {
		case "attachment":
			if attachmentChange.UUID == "" {
				return errors.New("new attachment should have a uuid")
			}
			addAttachmentUUIDs.Add(attachmentChange.UUID)
		case "external":
			if attachmentChange.Name == "" || attachmentChange.ExternalURL == "" {
				return errors.New("new external attachment should have a name and external url")
			}

			_, err = attachment.NewExternalAttachment(gitRepo.Ctx, &repo_model.Attachment{
				Name:        attachmentChange.Name,
				UploaderID:  rel.PublisherID,
				RepoID:      rel.RepoID,
				ReleaseID:   rel.ID,
				ExternalURL: attachmentChange.ExternalURL,
			})
			if err != nil {
				return err
			}
		default:
			if attachmentChange.Type == "" {
				return errors.New("missing attachment type")
			}
			return fmt.Errorf("unknown attachment type: '%q'", attachmentChange.Type)
		}
	}

	if err = repo_model.AddReleaseAttachments(gitRepo.Ctx, rel.ID, addAttachmentUUIDs.Values()); err != nil {
		return err
	}

	if !rel.IsDraft {
		notify_service.NewRelease(gitRepo.Ctx, rel)
	}

	return nil
}

// CreateNewTag creates a new repository tag
func CreateNewTag(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, commit, tagName, msg string) error {
	has, err := repo_model.IsReleaseExist(ctx, repo.ID, tagName)
	if err != nil {
		return err
	} else if has {
		return models.ErrTagAlreadyExists{
			TagName: tagName,
		}
	}

	gitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, repo)
	if err != nil {
		return err
	}
	defer closer.Close()

	rel := &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  doer.ID,
		Publisher:    doer,
		TagName:      tagName,
		Target:       commit,
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        true,
	}

	if _, err = createTag(ctx, gitRepo, rel, msg); err != nil {
		return err
	}

	return db.Insert(ctx, rel)
}

// UpdateRelease updates information, attachments of a release and will create tag if it's not a draft and tag not exist.
// addAttachmentUUIDs accept a slice of new created attachments' uuids which will be reassigned release_id as the created release
// delAttachmentUUIDs accept a slice of attachments' uuids which will be deleted from the release
// editAttachments accept a map of attachment uuid to new attachment name which will be updated with attachments.
func UpdateRelease(ctx context.Context, doer *user_model.User, gitRepo *git.Repository, rel *repo_model.Release, createdFromTag bool, attachmentChanges []*AttachmentChange,
) error {
	if rel.ID == 0 {
		return errors.New("UpdateRelease only accepts an exist release")
	}
	isCreated, err := createTag(gitRepo.Ctx, gitRepo, rel, "")
	if err != nil {
		return err
	}
	rel.LowerTagName = strings.ToLower(rel.TagName)

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err = repo_model.UpdateRelease(ctx, rel); err != nil {
		return err
	}

	addAttachmentUUIDs := make(container.Set[string])
	delAttachmentUUIDs := make(container.Set[string])
	updateAttachmentUUIDs := make(container.Set[string])
	updateAttachments := make(container.Set[*AttachmentChange])

	for _, attachmentChange := range attachmentChanges {
		switch attachmentChange.Action {
		case "add":
			switch attachmentChange.Type {
			case "attachment":
				if attachmentChange.UUID == "" {
					return fmt.Errorf("new attachment should have a uuid (%s)}", attachmentChange.Name)
				}
				addAttachmentUUIDs.Add(attachmentChange.UUID)
			case "external":
				if attachmentChange.Name == "" || attachmentChange.ExternalURL == "" {
					return errors.New("new external attachment should have a name and external url")
				}
				_, err := attachment.NewExternalAttachment(ctx, &repo_model.Attachment{
					Name:        attachmentChange.Name,
					UploaderID:  doer.ID,
					RepoID:      rel.RepoID,
					ReleaseID:   rel.ID,
					ExternalURL: attachmentChange.ExternalURL,
				})
				if err != nil {
					return err
				}
			default:
				if attachmentChange.Type == "" {
					return errors.New("missing attachment type")
				}
				return fmt.Errorf("unknown attachment type: %q", attachmentChange.Type)
			}
		case "delete":
			if attachmentChange.UUID == "" {
				return errors.New("attachment deletion should have a uuid")
			}
			delAttachmentUUIDs.Add(attachmentChange.UUID)
		case "update":
			updateAttachmentUUIDs.Add(attachmentChange.UUID)
			updateAttachments.Add(attachmentChange)
		default:
			if attachmentChange.Action == "" {
				return errors.New("missing attachment action")
			}
			return fmt.Errorf("unknown attachment action: %q", attachmentChange.Action)
		}
	}

	if err = repo_model.AddReleaseAttachments(ctx, rel.ID, addAttachmentUUIDs.Values()); err != nil {
		return fmt.Errorf("AddReleaseAttachments: %w", err)
	}

	deletedUUIDs := make(container.Set[string])
	if len(delAttachmentUUIDs) > 0 {
		// Check attachments
		attachments, err := repo_model.GetAttachmentsByUUIDs(ctx, delAttachmentUUIDs.Values())
		if err != nil {
			return fmt.Errorf("GetAttachmentsByUUIDs [uuids: %v]: %w", delAttachmentUUIDs, err)
		}
		for _, attach := range attachments {
			if attach.ReleaseID != rel.ID {
				return util.SilentWrap{
					Message: "delete attachment of release permission denied",
					Err:     util.ErrPermissionDenied,
				}
			}
			deletedUUIDs.Add(attach.UUID)
		}

		if _, err := repo_model.DeleteAttachments(ctx, attachments, true); err != nil {
			return fmt.Errorf("DeleteAttachments [uuids: %v]: %w", delAttachmentUUIDs, err)
		}
	}

	if len(updateAttachmentUUIDs) > 0 {
		// Check attachments
		attachments, err := repo_model.GetAttachmentsByUUIDs(ctx, updateAttachmentUUIDs.Values())
		if err != nil {
			return fmt.Errorf("GetAttachmentsByUUIDs [uuids: %v]: %w", updateAttachmentUUIDs, err)
		}
		for _, attach := range attachments {
			if attach.ReleaseID != rel.ID {
				return util.SilentWrap{
					Message: "update attachment of release permission denied",
					Err:     util.ErrPermissionDenied,
				}
			}
		}
	}

	for attachmentChange := range updateAttachments {
		if !deletedUUIDs.Contains(attachmentChange.UUID) {
			if err = repo_model.UpdateAttachmentByUUID(ctx, &repo_model.Attachment{
				UUID:        attachmentChange.UUID,
				Name:        attachmentChange.Name,
				ExternalURL: attachmentChange.ExternalURL,
			}, "name", "external_url"); err != nil {
				return err
			}
		}
	}

	if err := committer.Commit(); err != nil {
		return err
	}

	for uuid := range delAttachmentUUIDs.Seq() {
		if err := storage.Attachments.Delete(repo_model.AttachmentRelativePath(uuid)); err != nil {
			// Even delete files failed, but the attachments has been removed from database, so we
			// should not return error but only record the error on logs.
			// users have to delete this attachments manually or we should have a
			// synchronize between database attachment table and attachment storage
			log.Error("delete attachment[uuid: %s] failed: %v", uuid, err)
		}
	}

	if !rel.IsDraft {
		if createdFromTag || isCreated {
			notify_service.NewRelease(gitRepo.Ctx, rel)
			return nil
		}
		notify_service.UpdateRelease(gitRepo.Ctx, doer, rel)
	}
	return nil
}

// DeleteReleaseByID deletes a release and corresponding Git tag by given ID.
func DeleteReleaseByID(ctx context.Context, repo *repo_model.Repository, rel *repo_model.Release, doer *user_model.User, delTag bool) error {
	if delTag {
		protectedTags, err := git_model.GetProtectedTags(ctx, rel.RepoID)
		if err != nil {
			return fmt.Errorf("GetProtectedTags: %w", err)
		}
		isAllowed, err := git_model.IsUserAllowedToControlTag(ctx, protectedTags, rel.TagName, rel.PublisherID)
		if err != nil {
			return err
		}
		if !isAllowed {
			return models.ErrProtectedTagName{
				TagName: rel.TagName,
			}
		}

		err = repo_model.DeleteArchiveDownloadCountForRelease(ctx, rel.ID)
		if err != nil {
			return err
		}

		if stdout, _, err := git.NewCommand(ctx, "tag", "-d").AddDashesAndList(rel.TagName).
			SetDescription(fmt.Sprintf("DeleteReleaseByID (git tag -d): %d", rel.ID)).
			RunStdString(&git.RunOpts{Dir: repo.RepoPath()}); err != nil && !strings.Contains(err.Error(), "not found") {
			log.Error("DeleteReleaseByID (git tag -d): %d in %v Failed:\nStdout: %s\nError: %v", rel.ID, repo, stdout, err)
			return fmt.Errorf("git tag -d: %w", err)
		}

		refName := git.RefNameFromTag(rel.TagName)
		objectFormat := git.ObjectFormatFromName(repo.ObjectFormatName)
		notify_service.PushCommits(
			ctx, doer, repo,
			&repository.PushUpdateOptions{
				RefFullName: refName,
				OldCommitID: rel.Sha1,
				NewCommitID: objectFormat.EmptyObjectID().String(),
			}, repository.NewPushCommits())
		notify_service.DeleteRef(ctx, doer, repo, refName)

		if _, err := db.DeleteByID[repo_model.Release](ctx, rel.ID); err != nil {
			return fmt.Errorf("DeleteReleaseByID: %w", err)
		}
	} else {
		rel.IsTag = true

		if err := repo_model.UpdateRelease(ctx, rel); err != nil {
			return fmt.Errorf("Update: %w", err)
		}
	}

	rel.Repo = repo
	if err := rel.LoadAttributes(ctx); err != nil {
		return fmt.Errorf("LoadAttributes: %w", err)
	}

	if err := repo_model.DeleteAttachmentsByRelease(ctx, rel.ID); err != nil {
		return fmt.Errorf("DeleteAttachments: %w", err)
	}

	for i := range rel.Attachments {
		attachment := rel.Attachments[i]
		if err := storage.Attachments.Delete(attachment.RelativePath()); err != nil {
			log.Error("Delete attachment %s of release %s failed: %v", attachment.UUID, rel.ID, err)
		}
	}

	if !rel.IsDraft {
		notify_service.DeleteRelease(ctx, doer, rel)
	}
	return nil
}

// Init start release service
func Init() error {
	return initTagSyncQueue(graceful.GetManager().ShutdownContext())
}
