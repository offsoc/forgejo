// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/setting"
)

// UploadRepoFileOptions contains the uploaded repository file options
type UploadRepoFileOptions struct {
	LastCommitID string
	OldBranch    string
	NewBranch    string
	TreePath     string
	Message      string
	Author       *IdentityOptions
	Committer    *IdentityOptions
	Files        []string // In UUID format.
	FullPaths    []string
	Signoff      bool
}

type uploadInfo struct {
	upload        *repo_model.Upload
	lfsMetaObject *git_model.LFSMetaObject
}

func cleanUpAfterFailure(ctx context.Context, infos *[]uploadInfo, t *TemporaryUploadRepository, original error) error {
	for _, info := range *infos {
		if info.lfsMetaObject == nil {
			continue
		}
		if !info.lfsMetaObject.Existing {
			if _, err := git_model.RemoveLFSMetaObjectByOid(ctx, t.repo.ID, info.lfsMetaObject.Oid); err != nil {
				original = fmt.Errorf("%w, %v", original, err) // We wrap the original error - as this is the underlying error that required the fallback
			}
		}
	}
	return original
}

// UploadRepoFiles uploads files to the given repository
func UploadRepoFiles(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, opts *UploadRepoFileOptions) error {
	if len(opts.Files) == 0 {
		return nil
	}

	if len(opts.Files) != len(opts.FullPaths) {
		return fmt.Errorf("the length of opts.Files and opts.FullPaths is not the same")
	}

	uploads := make([]*repo_model.Upload, len(opts.Files))
	names := make([]string, len(opts.Files))
	infos := make([]uploadInfo, len(opts.Files))
	var err error
	for i := 0; i < len(opts.Files); i++ {
		uploads[i], err = repo_model.GetUploadByUUID(ctx, opts.Files[i])
		if err != nil {
			return fmt.Errorf("GetUploadByUUID [uuids: %v]: %w", opts.Files[i], err)
		}
		uploads[i].Name, err = SanitizePath(opts.FullPaths[i])
		if err != nil {
			return fmt.Errorf("SanitizePath [path: %v]: %w", opts.FullPaths[i], err)
		}
		// Check file is not lfs locked, will return nil if lock setting not enabled
		filepath := path.Join(opts.TreePath, uploads[i].Name)
		lfsLock, err := git_model.GetTreePathLock(ctx, repo.ID, filepath)
		if err != nil {
			return err
		}
		if lfsLock != nil && lfsLock.OwnerID != doer.ID {
			u, err := user_model.GetUserByID(ctx, lfsLock.OwnerID)
			if err != nil {
				return err
			}
			return git_model.ErrLFSFileLocked{RepoID: repo.ID, Path: filepath, UserName: u.Name}
		}

		names[i] = uploads[i].Name
		infos[i] = uploadInfo{upload: uploads[i]}
	}

	t, err := NewTemporaryUploadRepository(ctx, repo)
	if err != nil {
		return err
	}
	defer t.Close()

	hasOldBranch := true
	if err = t.Clone(opts.OldBranch, true); err != nil {
		if !git.IsErrBranchNotExist(err) || !repo.IsEmpty {
			return err
		}
		if err = t.Init(repo.ObjectFormatName); err != nil {
			return err
		}
		hasOldBranch = false
		opts.LastCommitID = ""
	}
	if hasOldBranch {
		if err = t.SetDefaultIndex(); err != nil {
			return err
		}
	}

	// Copy uploaded files into repository.
	if err := copyUploadedLFSFilesIntoRepository(infos, t, opts.TreePath); err != nil {
		return err
	}

	// Now write the tree
	treeHash, err := t.WriteTree()
	if err != nil {
		return err
	}

	author, committer := GetAuthorAndCommitterUsers(opts.Author, opts.Committer, doer)

	// Now commit the tree
	commitHash, err := t.CommitTree(opts.LastCommitID, author, committer, treeHash, opts.Message, opts.Signoff)
	if err != nil {
		return err
	}

	// Now deal with LFS objects
	for i := range infos {
		if infos[i].lfsMetaObject == nil {
			continue
		}
		infos[i].lfsMetaObject, err = git_model.NewLFSMetaObject(ctx, infos[i].lfsMetaObject.RepositoryID, infos[i].lfsMetaObject.Pointer)
		if err != nil {
			// OK Now we need to cleanup
			return cleanUpAfterFailure(ctx, &infos, t, err)
		}
		// Don't move the files yet - we need to ensure that
		// everything can be inserted first
	}

	// OK now we can insert the data into the store - there's no way to clean up the store
	// once it's in there, it's in there.
	contentStore := lfs.NewContentStore()
	for _, info := range infos {
		if err := uploadToLFSContentStore(info, contentStore); err != nil {
			return cleanUpAfterFailure(ctx, &infos, t, err)
		}
	}

	// Then push this tree to NewBranch
	if err := t.Push(doer, commitHash, opts.NewBranch); err != nil {
		return err
	}

	return repo_model.DeleteUploads(ctx, uploads...)
}

func copyUploadedLFSFilesIntoRepository(infos []uploadInfo, t *TemporaryUploadRepository, treePath string) error {
	var storeInLFSFunc func(string) (bool, error)

	if setting.LFS.StartServer {
		checker, err := t.gitRepo.GitAttributeChecker("", "filter")
		if err != nil {
			return err
		}
		defer checker.Close()

		storeInLFSFunc = func(name string) (bool, error) {
			attrs, err := checker.CheckPath(name)
			if err != nil {
				return false, fmt.Errorf("could not CheckPath(%s): %w", name, err)
			}
			return attrs["filter"] == "lfs", nil
		}
	}

	// Copy uploaded files into repository.
	for i, info := range infos {
		storeInLFS := false
		if storeInLFSFunc != nil {
			var err error
			storeInLFS, err = storeInLFSFunc(info.upload.Name)
			if err != nil {
				return err
			}
		}

		if err := copyUploadedLFSFileIntoRepository(&infos[i], storeInLFS, t, treePath); err != nil {
			return err
		}
	}
	return nil
}

func copyUploadedLFSFileIntoRepository(info *uploadInfo, storeInLFS bool, t *TemporaryUploadRepository, treePath string) error {
	file, err := os.Open(info.upload.LocalPath())
	if err != nil {
		return err
	}
	defer file.Close()

	var objectHash string
	if storeInLFS {
		// Handle LFS
		// FIXME: Inefficient! this should probably happen in models.Upload
		pointer, err := lfs.GeneratePointer(file)
		if err != nil {
			return err
		}

		info.lfsMetaObject = &git_model.LFSMetaObject{Pointer: pointer, RepositoryID: t.repo.ID}

		if objectHash, err = t.HashObject(strings.NewReader(pointer.StringContent())); err != nil {
			return err
		}
	} else if objectHash, err = t.HashObject(file); err != nil {
		return err
	}

	// Add the object to the index
	return t.AddObjectToIndex("100644", objectHash, path.Join(treePath, info.upload.Name))
}

func uploadToLFSContentStore(info uploadInfo, contentStore *lfs.ContentStore) error {
	if info.lfsMetaObject == nil {
		return nil
	}
	exist, err := contentStore.Exists(info.lfsMetaObject.Pointer)
	if err != nil {
		return err
	}
	if !exist {
		file, err := os.Open(info.upload.LocalPath())
		if err != nil {
			return err
		}

		defer file.Close()
		// FIXME: Put regenerates the hash and copies the file over.
		// I guess this strictly ensures the soundness of the store but this is inefficient.
		if err := contentStore.Put(info.lfsMetaObject.Pointer, file); err != nil {
			// OK Now we need to cleanup
			// Can't clean up the store, once uploaded there they're there.
			return err
		}
	}
	return nil
}
