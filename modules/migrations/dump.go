// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/migrations/base"
	"code.gitea.io/gitea/modules/repository"

	"gopkg.in/yaml.v2"
)

var (
	_ base.Uploader = &RepositoryDumper{}
)

// RepositoryDumper implements an Uploader to the local directory
type RepositoryDumper struct {
	ctx             context.Context
	baseDir         string
	repoOwner       string
	repoName        string
	milestoneFile   *os.File
	labelFile       *os.File
	releaseFile     *os.File
	issueFile       *os.File
	commentFile     *os.File
	pullrequestFile *os.File
	reviewFile      *os.File

	gitRepo     *git.Repository
	prHeadCache map[string]struct{}
}

// NewRepositoryDumper creates an gitea Uploader
func NewRepositoryDumper(ctx context.Context, baseDir, repoOwner, repoName string) *RepositoryDumper {
	return &RepositoryDumper{
		ctx:         ctx,
		baseDir:     baseDir,
		repoOwner:   repoOwner,
		repoName:    repoName,
		prHeadCache: make(map[string]struct{}),
	}
}

// MaxBatchInsertSize returns the table's max batch insert size
func (g *RepositoryDumper) MaxBatchInsertSize(tp string) int {
	return 1000
}

func (g *RepositoryDumper) repoPath() string {
	return filepath.Join(g.baseDir, "git")
}

func (g *RepositoryDumper) wikiPath() string {
	return filepath.Join(g.baseDir, "wiki")
}

func (g *RepositoryDumper) topicDir() string {
	return filepath.Join(g.baseDir, "topic")
}

func (g *RepositoryDumper) milestoneDir() string {
	return filepath.Join(g.baseDir, "milestone")
}

func (g *RepositoryDumper) labelDir() string {
	return filepath.Join(g.baseDir, "label")
}

func (g *RepositoryDumper) releaseDir() string {
	return filepath.Join(g.baseDir, "release")
}

func (g *RepositoryDumper) issueDir() string {
	return filepath.Join(g.baseDir, "issue")
}

func (g *RepositoryDumper) commentDir() string {
	return filepath.Join(g.baseDir, "comment")
}

func (g *RepositoryDumper) pullrequestDir() string {
	return filepath.Join(g.baseDir, "pullrequest")
}

func (g *RepositoryDumper) reviewDir() string {
	return filepath.Join(g.baseDir, "review")
}

// CreateRepo creates a repository
func (g *RepositoryDumper) CreateRepo(repo *base.Repository, opts base.MigrateOptions) error {
	repoPath := g.repoPath()
	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return err
	}

	migrateTimeout := 2 * time.Hour

	err := git.Clone(opts.CloneAddr, repoPath, git.CloneRepoOptions{
		Mirror:  true,
		Quiet:   true,
		Timeout: migrateTimeout,
	})
	if err != nil {
		return fmt.Errorf("Clone: %v", err)
	}

	if opts.Wiki {
		wikiPath := g.wikiPath()
		wikiRemotePath := repository.WikiRemoteURL(opts.CloneAddr)
		if len(wikiRemotePath) > 0 {
			if err := os.MkdirAll(wikiPath, os.ModePerm); err != nil {
				return fmt.Errorf("Failed to remove %s: %v", wikiPath, err)
			}

			if err := git.Clone(wikiRemotePath, wikiPath, git.CloneRepoOptions{
				Mirror:  true,
				Quiet:   true,
				Timeout: migrateTimeout,
				Branch:  "master",
			}); err != nil {
				log.Warn("Clone wiki: %v", err)
				if err := os.RemoveAll(wikiPath); err != nil {
					return fmt.Errorf("Failed to remove %s: %v", wikiPath, err)
				}
			}
		}
	}

	g.gitRepo, err = git.OpenRepository(g.repoPath())
	return err
}

// Close closes this uploader
func (g *RepositoryDumper) Close() {
	if g.gitRepo != nil {
		g.gitRepo.Close()
	}
	if g.milestoneFile != nil {
		g.milestoneFile.Close()
	}
	if g.labelFile != nil {
		g.labelFile.Close()
	}
	if g.releaseFile != nil {
		g.releaseFile.Close()
	}
	if g.issueFile != nil {
		g.issueFile.Close()
	}
	if g.pullrequestFile != nil {
		g.pullrequestFile.Close()
	}
	if g.reviewFile != nil {
		g.reviewFile.Close()
	}
}

// CreateTopics creates topics
func (g *RepositoryDumper) CreateTopics(topics ...string) error {
	if err := os.MkdirAll(g.topicDir(), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(g.topicDir(), "data.yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	bs, err := yaml.Marshal(map[string]interface{}{
		"topics": topics,
	})
	if err != nil {
		return err
	}

	if _, err := f.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreateMilestones creates milestones
func (g *RepositoryDumper) CreateMilestones(milestones ...*base.Milestone) error {
	var err error
	if g.milestoneFile == nil {
		if err := os.MkdirAll(g.milestoneDir(), os.ModePerm); err != nil {
			return err
		}
		g.milestoneFile, err = os.Create(filepath.Join(g.milestoneDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(milestones)
	if err != nil {
		return err
	}

	if _, err := g.milestoneFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreateLabels creates labels
func (g *RepositoryDumper) CreateLabels(labels ...*base.Label) error {
	var err error
	if g.labelFile == nil {
		if err := os.MkdirAll(g.labelDir(), os.ModePerm); err != nil {
			return err
		}
		g.labelFile, err = os.Create(filepath.Join(g.labelDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(labels)
	if err != nil {
		return err
	}

	if _, err := g.labelFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreateReleases creates releases
func (g *RepositoryDumper) CreateReleases(releases ...*base.Release) error {
	for _, release := range releases {
		attachDir := filepath.Join(g.releaseDir(), "assets", release.Name)
		if err := os.MkdirAll(attachDir, os.ModePerm); err != nil {
			return err
		}
		for _, asset := range release.Assets {
			attachLocalPath := filepath.Join(attachDir, asset.Name)
			// download attachment
			err := func(attachLocalPath string) error {
				resp, err := http.Get(asset.URL)
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				fw, err := os.Create(attachLocalPath)
				if err != nil {
					return fmt.Errorf("Create: %v", err)
				}
				defer fw.Close()

				_, err = io.Copy(fw, resp.Body)
				return err
			}(attachLocalPath)
			if err != nil {
				return err
			}
		}
	}

	var err error
	if g.releaseFile == nil {
		if err := os.MkdirAll(g.releaseDir(), os.ModePerm); err != nil {
			return err
		}
		g.releaseFile, err = os.Create(filepath.Join(g.releaseDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(releases)
	if err != nil {
		return err
	}

	if _, err := g.releaseFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// SyncTags syncs releases with tags in the database
func (g *RepositoryDumper) SyncTags() error {
	return nil
}

// CreateIssues creates issues
func (g *RepositoryDumper) CreateIssues(issues ...*base.Issue) error {
	var err error
	if g.issueFile == nil {
		if err := os.MkdirAll(g.issueDir(), os.ModePerm); err != nil {
			return err
		}
		g.issueFile, err = os.Create(filepath.Join(g.issueDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(issues)
	if err != nil {
		return err
	}

	if _, err := g.issueFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreateComments creates comments of issues
func (g *RepositoryDumper) CreateComments(comments ...*base.Comment) error {
	var err error
	if g.commentFile == nil {
		if err := os.MkdirAll(g.commentDir(), os.ModePerm); err != nil {
			return err
		}
		g.commentFile, err = os.Create(filepath.Join(g.commentDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(comments)
	if err != nil {
		return err
	}

	if _, err := g.commentFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreatePullRequests creates pull requests
func (g *RepositoryDumper) CreatePullRequests(prs ...*base.PullRequest) error {
	for _, pr := range prs {
		// download patch file
		err := func() error {
			resp, err := http.Get(pr.PatchURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			pullDir := filepath.Join(g.repoPath(), "pulls")
			if err = os.MkdirAll(pullDir, os.ModePerm); err != nil {
				return err
			}
			f, err := os.Create(filepath.Join(pullDir, fmt.Sprintf("%d.patch", pr.Number)))
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, resp.Body)
			return err
		}()
		if err != nil {
			return err
		}

		// set head information
		pullHead := filepath.Join(g.repoPath(), "refs", "pull", fmt.Sprintf("%d", pr.Number))
		if err := os.MkdirAll(pullHead, os.ModePerm); err != nil {
			return err
		}
		p, err := os.Create(filepath.Join(pullHead, "head"))
		if err != nil {
			return err
		}
		_, err = p.WriteString(pr.Head.SHA)
		p.Close()
		if err != nil {
			return err
		}

		if pr.IsForkPullRequest() && pr.State != "closed" {
			if pr.Head.OwnerName != "" {
				remote := pr.Head.OwnerName
				_, ok := g.prHeadCache[remote]
				if !ok {
					// git remote add
					err := g.gitRepo.AddRemote(remote, pr.Head.CloneURL, true)
					if err != nil {
						log.Error("AddRemote failed: %s", err)
					} else {
						g.prHeadCache[remote] = struct{}{}
						ok = true
					}
				}

				if ok {
					_, err = git.NewCommand("fetch", remote, pr.Head.Ref).RunInDir(g.repoPath())
					if err != nil {
						log.Error("Fetch branch from %s failed: %v", pr.Head.CloneURL, err)
					} else {
						headBranch := filepath.Join(g.repoPath(), "refs", "heads", pr.Head.OwnerName, pr.Head.Ref)
						if err := os.MkdirAll(filepath.Dir(headBranch), os.ModePerm); err != nil {
							return err
						}
						b, err := os.Create(headBranch)
						if err != nil {
							return err
						}
						_, err = b.WriteString(pr.Head.SHA)
						b.Close()
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	var err error
	if g.pullrequestFile == nil {
		if err := os.MkdirAll(g.pullrequestDir(), os.ModePerm); err != nil {
			return err
		}
		g.pullrequestFile, err = os.Create(filepath.Join(g.pullrequestDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(prs)
	if err != nil {
		return err
	}

	if _, err := g.pullrequestFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// CreateReviews create pull request reviews
func (g *RepositoryDumper) CreateReviews(reviews ...*base.Review) error {
	var err error
	if g.reviewFile == nil {
		if err := os.MkdirAll(g.reviewDir(), os.ModePerm); err != nil {
			return err
		}
		g.reviewFile, err = os.Create(filepath.Join(g.reviewDir(), "data.yml"))
		if err != nil {
			return err
		}
	}

	bs, err := yaml.Marshal(reviews)
	if err != nil {
		return err
	}

	if _, err := g.reviewFile.Write(bs); err != nil {
		return err
	}

	return nil
}

// Rollback when migrating failed, this will rollback all the changes.
func (g *RepositoryDumper) Rollback() error {
	g.Close()
	return os.RemoveAll(g.baseDir)
}

// DumpRepository dump repository according MigrateOptions to a local directory
func DumpRepository(ctx context.Context, baseDir, ownerName string, opts base.MigrateOptions) error {
	var uploader = NewRepositoryDumper(ctx, baseDir, ownerName, opts.RepoName)
	return MigrateRepositoryWithUploader(ctx, ownerName, opts, uploader)
}
