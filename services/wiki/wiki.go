// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package wiki

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"unsafe"

	repo_model "code.gitea.io/gitea/models/repo"
	system_model "code.gitea.io/gitea/models/system"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sync"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	repo_service "code.gitea.io/gitea/services/repository"
)

// TODO: use clustered lock (unique queue? or *abuse* cache)
var wikiWorkingPool = sync.NewExclusivePool()

const (
	DefaultRemote = "origin"
)

// InitWiki initializes a wiki for repository,
// it does nothing when repository already has wiki.
func InitWiki(ctx context.Context, repo *repo_model.Repository) error {
	if repo.HasWiki() {
		return nil
	}

	branch := repo.GetWikiBranchName()

	if err := git.InitRepository(ctx, repo.WikiPath(), true, repo.ObjectFormatName); err != nil {
		return fmt.Errorf("InitRepository: %w", err)
	} else if err = repo_module.CreateDelegateHooks(repo.WikiPath()); err != nil {
		return fmt.Errorf("createDelegateHooks: %w", err)
	} else if _, _, err = git.NewCommand(ctx, "symbolic-ref", "HEAD").AddDynamicArguments(git.BranchPrefix + branch).RunStdString(&git.RunOpts{Dir: repo.WikiPath()}); err != nil {
		return fmt.Errorf("unable to set default wiki branch to %s: %w", branch, err)
	}
	return nil
}

// NormalizeWikiBranch renames a repository wiki's branch to `setting.Repository.DefaultBranch`
func NormalizeWikiBranch(ctx context.Context, repo *repo_model.Repository, to string) error {
	from := repo.GetWikiBranchName()

	if err := repo.MustNotBeArchived(); err != nil {
		return err
	}

	updateDB := func() error {
		repo.WikiBranch = to
		return repo_model.UpdateRepositoryCols(ctx, repo, "wiki_branch")
	}

	if !repo.HasWiki() {
		return updateDB()
	}

	if from == to {
		return nil
	}

	gitRepo, err := git.OpenRepository(ctx, repo.WikiPath())
	if err != nil {
		return err
	}
	defer gitRepo.Close()

	if gitRepo.IsBranchExist(to) {
		return nil
	}

	if !gitRepo.IsBranchExist(from) {
		return nil
	}

	if err := gitRepo.RenameBranch(from, to); err != nil {
		return err
	}

	if err := gitrepo.SetDefaultBranch(ctx, repo, to); err != nil {
		return err
	}

	return updateDB()
}

// prepareGitPath try to find a suitable file path with file name by the given raw wiki name.
// return: existence, prepared file path with name, error
func prepareGitPath(gitRepo *git.Repository, branch string, wikiPath WebPath) (bool, string, error) {
	unescaped := string(wikiPath) + ".md"
	gitPath, err := SanitizeWikiPath(string(wikiPath))
	if err != nil {
		return false, string(gitPath), err
	}

	// Look for both files
	filesInIndex, err := gitRepo.LsTree(branch, unescaped, string(gitPath))
	if err != nil {
		if strings.Contains(err.Error(), "Not a valid object name "+branch) {
			return false, string(gitPath), nil
		}
		log.Error("%v", err)
		return false, string(gitPath), err
	}

	foundEscaped := false
	for _, filename := range filesInIndex {
		switch filename {
		case unescaped:
			// if we find the unescaped file return it
			return true, unescaped, nil
		case string(gitPath):
			foundEscaped = true
		}
	}

	// If not return whether the escaped file exists, and the escaped filename to keep backwards compatibility.
	return foundEscaped, string(gitPath), nil
}

// updateWikiPage adds a new page or edits an existing page in repository wiki.
func updateWikiPage(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, oldWikiName, newWikiName WebPath, content, message string, isNew bool) (err error) {
	err = repo.MustNotBeArchived()
	if err != nil {
		return err
	}

	sanitizedWikiPath, err := SanitizeWikiPath(string(newWikiName))
	if err != nil {
		return err
	}
	sanitizedWikiPath += ".md"
	if err = validateWebPath(WebPath(sanitizedWikiPath)); err != nil {
		return err
	}
	wikiWorkingPool.CheckIn(fmt.Sprint(repo.ID))
	defer wikiWorkingPool.CheckOut(fmt.Sprint(repo.ID))

	if err = InitWiki(ctx, repo); err != nil {
		return fmt.Errorf("InitWiki: %w", err)
	}

	hasMasterBranch := git.IsBranchExist(ctx, repo.WikiPath(), repo.GetWikiBranchName())

	basePath, err := repo_module.CreateTemporaryPath("update-wiki")
	if err != nil {
		return err
	}
	defer func() {
		if err := repo_module.RemoveTemporaryPath(basePath); err != nil {
			log.Error("Merge: RemoveTemporaryPath: %s", err)
		}
	}()

	cloneOpts := git.CloneRepoOptions{
		Bare:   true,
		Shared: true,
	}

	if hasMasterBranch {
		cloneOpts.Branch = repo.GetWikiBranchName()
	}

	if err := git.Clone(ctx, repo.WikiPath(), basePath, cloneOpts); err != nil {
		log.Error("Failed to clone repository: %s (%v)", repo.FullName(), err)
		return fmt.Errorf("failed to clone repository: %s (%w)", repo.FullName(), err)
	}

	gitRepo, err := git.OpenRepository(ctx, basePath)
	if err != nil {
		log.Error("Unable to open temporary repository: %s (%v)", basePath, err)
		return fmt.Errorf("failed to open new temporary repository in: %s %w", basePath, err)
	}
	defer gitRepo.Close()

	if hasMasterBranch {
		if err := gitRepo.ReadTreeToIndex("HEAD"); err != nil {
			log.Error("Unable to read HEAD tree to index in: %s %v", basePath, err)
			return fmt.Errorf("fnable to read HEAD tree to index in: %s %w", basePath, err)
		}
	}

	isWikiExist, newWikiPath, err := prepareGitPath(gitRepo, repo.GetWikiBranchName(), WebPath(sanitizedWikiPath))
	if err != nil {
		return err
	}

	if isNew {
		if isWikiExist {
			return repo_model.ErrWikiAlreadyExist{
				Title: newWikiPath,
			}
		}
	} else {
		// avoid check existence again if wiki name is not changed since gitRepo.LsFiles(...) is not free.
		isOldWikiExist := true
		oldWikiPath := newWikiPath
		if oldWikiName != WebPath(sanitizedWikiPath) {
			isOldWikiExist, oldWikiPath, err = prepareGitPath(gitRepo, repo.GetWikiBranchName(), oldWikiName)
			if err != nil {
				return err
			}
		}

		if isOldWikiExist {
			err := gitRepo.RemoveFilesFromIndex(oldWikiPath)
			if err != nil {
				log.Error("RemoveFilesFromIndex failed: %v", err)
				return err
			}
		}
	}

	// FIXME: The wiki doesn't have lfs support at present - if this changes need to check attributes here

	objectHash, err := gitRepo.HashObject(strings.NewReader(content))
	if err != nil {
		log.Error("HashObject failed: %v", err)
		return err
	}

	if err := gitRepo.AddObjectToIndex("100644", objectHash, newWikiPath); err != nil {
		log.Error("AddObjectToIndex failed: %v", err)
		return err
	}

	tree, err := gitRepo.WriteTree()
	if err != nil {
		log.Error("WriteTree failed: %v", err)
		return err
	}

	commitTreeOpts := git.CommitTreeOpts{
		Message: message,
	}

	committer := doer.NewGitSig()
	sign, signingKey, signer, _ := asymkey_service.SignWikiCommit(ctx, repo, doer)
	if sign {
		commitTreeOpts.KeyID = signingKey
		if repo.GetTrustModel() == repo_model.CommitterTrustModel || repo.GetTrustModel() == repo_model.CollaboratorCommitterTrustModel {
			committer = signer
		}
	} else {
		commitTreeOpts.NoGPGSign = true
	}
	if hasMasterBranch {
		commitTreeOpts.Parents = []string{"HEAD"}
	}

	commitHash, err := gitRepo.CommitTree(doer.NewGitSig(), committer, tree, commitTreeOpts)
	if err != nil {
		log.Error("CommitTree failed: %v", err)
		return err
	}
	if err := git.Push(gitRepo.Ctx, basePath, git.PushOptions{
		Remote: DefaultRemote,
		Branch: fmt.Sprintf("%s:%s%s", commitHash.String(), git.BranchPrefix, repo.GetWikiBranchName()),
		Env: repo_module.FullPushingEnvironment(
			doer,
			doer,
			repo,
			repo.Name+".wiki",
			0,
		),
	}); err != nil {
		log.Error("Push failed: %v", err)
		if git.IsErrPushOutOfDate(err) || git.IsErrPushRejected(err) {
			return err
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// AddWikiPage adds a new wiki page with a given wikiPath.
func AddWikiPage(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, wikiName WebPath, content, message string) error {
	return updateWikiPage(ctx, doer, repo, "", wikiName, content, message, true)
}

// EditWikiPage updates a wiki page identified by its wikiPath,
// optionally also changing wikiPath.
func EditWikiPage(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, oldWikiName, newWikiName WebPath, content, message string) error {
	return updateWikiPage(ctx, doer, repo, oldWikiName, newWikiName, content, message, false)
}

// DeleteWikiPage deletes a wiki page identified by its path.
func DeleteWikiPage(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, wikiName WebPath) (err error) {
	err = repo.MustNotBeArchived()
	if err != nil {
		return err
	}

	wikiWorkingPool.CheckIn(fmt.Sprint(repo.ID))
	defer wikiWorkingPool.CheckOut(fmt.Sprint(repo.ID))

	if err = InitWiki(ctx, repo); err != nil {
		return fmt.Errorf("InitWiki: %w", err)
	}

	basePath, err := repo_module.CreateTemporaryPath("update-wiki")
	if err != nil {
		return err
	}
	defer func() {
		if err := repo_module.RemoveTemporaryPath(basePath); err != nil {
			log.Error("Merge: RemoveTemporaryPath: %s", err)
		}
	}()

	if err := git.Clone(ctx, repo.WikiPath(), basePath, git.CloneRepoOptions{
		Bare:   true,
		Shared: true,
		Branch: repo.GetWikiBranchName(),
	}); err != nil {
		log.Error("Failed to clone repository: %s (%v)", repo.FullName(), err)
		return fmt.Errorf("failed to clone repository: %s (%w)", repo.FullName(), err)
	}

	gitRepo, err := git.OpenRepository(ctx, basePath)
	if err != nil {
		log.Error("Unable to open temporary repository: %s (%v)", basePath, err)
		return fmt.Errorf("failed to open new temporary repository in: %s %w", basePath, err)
	}
	defer gitRepo.Close()

	if err := gitRepo.ReadTreeToIndex("HEAD"); err != nil {
		log.Error("Unable to read HEAD tree to index in: %s %v", basePath, err)
		return fmt.Errorf("unable to read HEAD tree to index in: %s %w", basePath, err)
	}

	found, wikiPath, err := prepareGitPath(gitRepo, repo.GetWikiBranchName(), wikiName)
	if err != nil {
		return err
	}
	if found {
		err := gitRepo.RemoveFilesFromIndex(wikiPath)
		if err != nil {
			return err
		}
	} else {
		return os.ErrNotExist
	}

	// FIXME: The wiki doesn't have lfs support at present - if this changes need to check attributes here

	tree, err := gitRepo.WriteTree()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Delete page %q", wikiName)
	commitTreeOpts := git.CommitTreeOpts{
		Message: message,
		Parents: []string{"HEAD"},
	}

	committer := doer.NewGitSig()

	sign, signingKey, signer, _ := asymkey_service.SignWikiCommit(ctx, repo, doer)
	if sign {
		commitTreeOpts.KeyID = signingKey
		if repo.GetTrustModel() == repo_model.CommitterTrustModel || repo.GetTrustModel() == repo_model.CollaboratorCommitterTrustModel {
			committer = signer
		}
	} else {
		commitTreeOpts.NoGPGSign = true
	}

	commitHash, err := gitRepo.CommitTree(doer.NewGitSig(), committer, tree, commitTreeOpts)
	if err != nil {
		return err
	}

	if err := git.Push(gitRepo.Ctx, basePath, git.PushOptions{
		Remote: DefaultRemote,
		Branch: fmt.Sprintf("%s:%s%s", commitHash.String(), git.BranchPrefix, repo.GetWikiBranchName()),
		Env: repo_module.FullPushingEnvironment(
			doer,
			doer,
			repo,
			repo.Name+".wiki",
			0,
		),
	}); err != nil {
		if git.IsErrPushOutOfDate(err) || git.IsErrPushRejected(err) {
			return err
		}
		return fmt.Errorf("Push: %w", err)
	}

	return nil
}

// DeleteWiki removes the actual and local copy of repository wiki.
func DeleteWiki(ctx context.Context, repo *repo_model.Repository) error {
	if err := repo_service.UpdateRepositoryUnits(ctx, repo, nil, []unit.Type{unit.TypeWiki}); err != nil {
		return err
	}

	system_model.RemoveAllWithNotice(ctx, "Delete repository wiki", repo.WikiPath())
	return nil
}

type SearchContentsResult struct {
	*git.GrepResult
	Title string
}

func SearchWikiContents(ctx context.Context, repo *repo_model.Repository, keyword string) ([]SearchContentsResult, error) {
	gitRepo, err := git.OpenRepository(ctx, repo.WikiPath())
	if err != nil {
		return nil, err
	}
	defer gitRepo.Close()

	grepRes, err := git.GrepSearch(ctx, gitRepo, keyword, git.GrepOptions{
		ContextLineNumber: 0,
		Mode:              git.FixedAnyGrepMode,
		RefName:           repo.GetWikiBranchName(),
		MaxResultLimit:    10,
		MatchesPerFile:    3,
	})
	if err != nil {
		return nil, err
	}

	res := make([]SearchContentsResult, 0, len(grepRes))
	for _, entry := range grepRes {
		dispplayName, err := fullpathToDisplayName(entry.Filename)
		if err != nil {
			continue
		}
		res = append(res, SearchContentsResult{
			GrepResult: entry,
			Title:      dispplayName,
		})
	}

	return res, nil
}

type Page struct {
	DisplayName string
	GitPath     string
	SubURL      string
	Commit      *git.Commit
}

// ListWikiPages returns a list of all pages in a wiki
// This takes a sorter interface compatible to the old sorters
// TODO: Display/return an error, if a reserved name is in the pages.
// this can happen if a user pushes such a file directly per git
func ListWikiPages(
	ctx context.Context,
	commit *git.Commit,
	sorter func(s1, s2 string) bool,
) ([]Page, error) {
	entries, err := commit.ListEntriesRecursiveFast()
	if err != nil {
		return nil, err
	}

	dirPages := make(map[string][]string, 0)
	for _, entry := range entries {
		if !entry.IsRegular() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		dir, name := pathToDirAndFile(entry.Name())
		subPages, exists := dirPages[dir]
		if !exists {
			subPages = make([]string, 0)
		}
		dirPages[dir] = append(subPages, name)
	}

	pages := make([]Page, 0)
	for dir := range dirPages {
		currentDirPages := dirPages[dir]

		dirCommits, err := git.GetLastCommitForPaths(ctx, commit, dir, currentDirPages)
		if err != nil {
			log.Error("Failed to read Subdir commit! dir: %s err: %s", dir, err)
			continue
		}

		for _, page := range currentDirPages {
			fullPath := dirAndFileToFullpath(dir, page)
			subURL, err := fullpathToSubURL(fullPath)
			if err != nil {
				continue
			}
			displayName, err := fullpathToDisplayName(fullPath)
			if err != nil {
				continue
			}

			pages = append(pages, Page{
				DisplayName: displayName,
				SubURL:      subURL,
				GitPath:     fullPath,
				Commit:      dirCommits[page],
			})
		}
	}

	slices.SortFunc(pages, func(s1, s2 Page) int {
		if s1.DisplayName == s2.DisplayName {
			return 0
		}
		res := sorter(s1.DisplayName, s2.DisplayName)
		// This is about 2 times faster than a normal if statement
		// On Wikis with a lot of Pages this will have a big impact
		// The Only way to get even faster would be taking a sort-
		// function compatible with slices.SortFuc
		// Allternatively Pagination should be added to the Wiki view, or a Tree like structure should be shown, which does on demand loading
		return int(*(*byte)(unsafe.Pointer(&res)))*-2 + 1
	})

	return pages, nil
}

func fullpathToDisplayName(fullpath string) (string, error) {
	return fullpathToSubURL(fullpath)
}

func fullpathToSubURL(fullpath string) (string, error) {
	subURL, _ := strings.CutPrefix(fullpath, "/")
	subURL, _ = strings.CutSuffix(subURL, ".md")
	return subURL, nil
}

func dirAndFileToFullpath(dir, file string) string {
	if dir != "" {
		return fmt.Sprintf("%s/%s", dir, file)
	}
	return file
}

func pathToDirAndFile(path string) (string, string) {
	index := strings.LastIndex(path, "/")
	if index == -1 {
		return "", path
	}
	return path[:index], path[index+1:]
}
