// Copyright 2019 The Gitea Authors.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package pull

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"

	"github.com/gobwas/glob"
)

// DownloadDiffOrPatch will write the patch for the pr to the writer
func DownloadDiffOrPatch(pr *models.PullRequest, w io.Writer, patch bool) error {
	if err := pr.LoadBaseRepo(); err != nil {
		log.Error("Unable to load base repository ID %d for pr #%d [%d]", pr.BaseRepoID, pr.Index, pr.ID)
		return err
	}

	gitRepo, err := git.OpenRepository(pr.BaseRepo.RepoPath())
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}
	defer gitRepo.Close()
	if err := gitRepo.GetDiffOrPatch(pr.MergeBase, pr.GetGitRefName(), w, patch); err != nil {
		log.Error("Unable to get patch file from %s to %s in %s Error: %v", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
		return fmt.Errorf("Unable to get patch file from %s to %s in %s Error: %v", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
	}
	return nil
}

var patchErrorSuffices = []string{
	": already exists in index",
	": patch does not apply",
	": already exists in working directory",
	"unrecognized input",
}

// TestPatch will test whether a simple patch will apply
func TestPatch(pr *models.PullRequest) error {
	// Clone base repo.
	tmpBasePath, err := createTemporaryRepo(pr)
	if err != nil {
		log.Error("CreateTemporaryPath: %v", err)
		return err
	}
	defer func() {
		if err := models.RemoveTemporaryPath(tmpBasePath); err != nil {
			log.Error("Merge: RemoveTemporaryPath: %s", err)
		}
	}()

	gitRepo, err := git.OpenRepository(tmpBasePath)
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}
	defer gitRepo.Close()

	pr.MergeBase, err = git.NewCommand("merge-base", "--", "base", "tracking").RunInDir(tmpBasePath)
	if err != nil {
		var err2 error
		pr.MergeBase, err2 = gitRepo.GetRefCommitID(git.BranchPrefix + "base")
		if err2 != nil {
			return fmt.Errorf("GetMergeBase: %v and can't find commit ID for base: %v", err, err2)
		}
	}
	pr.MergeBase = strings.TrimSpace(pr.MergeBase)
	tmpPatchFile, err := ioutil.TempFile("", "patch")
	if err != nil {
		log.Error("Unable to create temporary patch file! Error: %v", err)
		return fmt.Errorf("Unable to create temporary patch file! Error: %v", err)
	}
	defer func() {
		_ = util.Remove(tmpPatchFile.Name())
	}()

	if err := gitRepo.GetDiff(pr.MergeBase, "tracking", tmpPatchFile); err != nil {
		tmpPatchFile.Close()
		log.Error("Unable to get patch file from %s to %s in %s Error: %v", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
		return fmt.Errorf("Unable to get patch file from %s to %s in %s Error: %v", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
	}
	stat, err := tmpPatchFile.Stat()
	if err != nil {
		tmpPatchFile.Close()
		return fmt.Errorf("Unable to stat patch file: %v", err)
	}
	patchPath := tmpPatchFile.Name()
	tmpPatchFile.Close()

	if stat.Size() == 0 {
		log.Debug("PullRequest[%d]: Patch is empty - ignoring", pr.ID)
		pr.Status = models.PullRequestStatusMergeable
		pr.ConflictedFiles = []string{}
		return nil
	}

	log.Trace("PullRequest[%d].testPatch (patchPath): %s", pr.ID, patchPath)

	pr.Status = models.PullRequestStatusChecking

	_, err = git.NewCommand("read-tree", "base").RunInDir(tmpBasePath)
	if err != nil {
		return fmt.Errorf("git read-tree %s: %v", pr.BaseBranch, err)
	}

	prUnit, err := pr.BaseRepo.GetUnit(models.UnitTypePullRequests)
	if err != nil {
		return err
	}
	prConfig := prUnit.PullRequestsConfig()

	args := []string{"apply", "--check", "--cached"}
	if prConfig.IgnoreWhitespaceConflicts {
		args = append(args, "--ignore-whitespace")
	}
	args = append(args, patchPath)
	pr.ConflictedFiles = make([]string, 0, 5)

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		log.Error("Unable to open stderr pipe: %v", err)
		return fmt.Errorf("Unable to open stderr pipe: %v", err)
	}
	defer func() {
		_ = stderrReader.Close()
		_ = stderrWriter.Close()
	}()
	conflict := false
	err = git.NewCommand(args...).
		RunInDirTimeoutEnvFullPipelineFunc(
			nil, -1, tmpBasePath,
			nil, stderrWriter, nil,
			func(ctx context.Context, cancel context.CancelFunc) error {
				_ = stderrWriter.Close()
				const prefix = "error: patch failed:"
				const errorPrefix = "error: "
				conflictMap := map[string]bool{}

				scanner := bufio.NewScanner(stderrReader)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, prefix) {
						conflict = true
						filepath := strings.TrimSpace(strings.Split(line[len(prefix):], ":")[0])
						conflictMap[filepath] = true
					} else if strings.HasPrefix(line, errorPrefix) {
						conflict = true
						for _, suffix := range patchErrorSuffices {
							if strings.HasSuffix(line, suffix) {
								filepath := strings.TrimSpace(strings.TrimSuffix(line[len(errorPrefix):], suffix))
								if filepath != "" {
									conflictMap[filepath] = true
								}
								break
							}
						}
					}
					// only list 10 conflicted files
					if len(conflictMap) >= 10 {
						break
					}
				}
				if len(conflictMap) > 0 {
					pr.ConflictedFiles = make([]string, 0, len(conflictMap))
					for key := range conflictMap {
						pr.ConflictedFiles = append(pr.ConflictedFiles, key)
					}
				}
				_ = stderrReader.Close()
				return nil
			})

	if err != nil {
		if conflict {
			pr.Status = models.PullRequestStatusConflict
			log.Trace("Found %d files conflicted: %v", len(pr.ConflictedFiles), pr.ConflictedFiles)

			return nil
		}
		return fmt.Errorf("git apply --check: %v", err)
	}

	if pr.Index != 0 {
		if err = CheckPullFilesProtection(pr); err != nil {
			return fmt.Errorf("pr.CheckPullFilesProtection(): %v", err)
		}
	}

	if len(pr.ChangedProtectedFiles) > 0 {
		log.Trace("Found %d protected files changed", len(pr.ChangedProtectedFiles))
	}

	pr.Status = models.PullRequestStatusMergeable

	return nil
}

// CheckFileProtection check file Protection
func CheckFileProtection(oldCommitID, newCommitID string, patterns []glob.Glob, limit int, env []string, repo *git.Repository) ([]string, error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		log.Error("Unable to create os.Pipe for %s", repo.Path)
		return nil, err
	}
	defer func() {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
	}()

	changedProtectedFiles := make([]string, 0, limit)

	// This use of ...  is safe as force-pushes have already been ruled out.
	err = git.NewCommand("diff", "--name-only", oldCommitID+"..."+newCommitID).
		RunInDirTimeoutEnvFullPipelineFunc(env, -1, repo.Path,
			stdoutWriter, nil, nil,
			func(ctx context.Context, cancel context.CancelFunc) error {
				_ = stdoutWriter.Close()
				counter := 0

				scanner := bufio.NewScanner(stdoutReader)
				for scanner.Scan() {
					path := strings.TrimSpace(scanner.Text())
					if len(path) == 0 {
						continue
					}
					lpath := strings.ToLower(path)
					for _, pat := range patterns {
						if pat.Match(lpath) {
							if counter < limit {
								counter++
								changedProtectedFiles = append(changedProtectedFiles, path)
								continue
							}
							cancel()
							return models.ErrFilePathProtected{
								Path: path,
							}
						}
					}
					if counter >= limit {
						break
					}
				}
				err := scanner.Err()
				return err
			})
	if err != nil && !models.IsErrFilePathProtected(err) {
		log.Error("Unable to check file protection for commits from %s to %s in %s: %v", oldCommitID, newCommitID, repo.Path, err)
	}

	return changedProtectedFiles, err
}

// CheckPullFilesProtection check if pr changed protected files and save results
func CheckPullFilesProtection(pr *models.PullRequest) (err error) {
	if err = pr.LoadProtectedBranch(); err != nil {
		return
	}

	if pr.ProtectedBranch == nil {
		pr.ChangedProtectedFiles = nil
		return nil
	}

	if err = pr.LoadBaseRepo(); err != nil {
		return
	}

	gitRepo, err := git.OpenRepository(pr.BaseRepo.RepoPath())
	if err != nil {
		return err
	}
	defer gitRepo.Close()

	headCommitID, err := gitRepo.GetRefCommitID(pr.GetGitRefName())
	if err != nil {
		return err
	}

	pr.ChangedProtectedFiles, err = CheckFileProtection(pr.MergeBase, headCommitID, pr.ProtectedBranch.GetProtectedFilePatterns(), 10, os.Environ(), gitRepo)
	if models.IsErrFilePathProtected(err) {
		err = nil
	}
	return
}
