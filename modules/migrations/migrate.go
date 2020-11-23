// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2018 Jonas Franz. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/matchlist"
	"code.gitea.io/gitea/modules/migrations/base"
	"code.gitea.io/gitea/modules/setting"
)

// MigrateOptions is equal to base.MigrateOptions
type MigrateOptions = base.MigrateOptions

var (
	factories []base.DownloaderFactory

	allowList          *matchlist.Matchlist
	blockList          *matchlist.Matchlist
	privateNetworkList *matchlist.Matchlist
)

// RegisterDownloaderFactory registers a downloader factory
func RegisterDownloaderFactory(factory base.DownloaderFactory) {
	factories = append(factories, factory)
}

func isMigrateURLAllowed(remoteURL string) (bool, error) {
	u, err := url.Parse(strings.ToLower(remoteURL))
	if err != nil {
		return false, err
	}

	if strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https") {
		if len(setting.Migrations.AllowlistedDomains) > 0 {
			if !allowList.Match(u.Host) {
				return false, fmt.Errorf("migrate from '%v' is not allowed", u.Host)
			}
		} else {
			if blockList.Match(u.Host) {
				return false, fmt.Errorf("migrate from '%v' is not allowed", u.Host)
			}
		}
	}

	if !setting.Migrations.AllowLocalNetworks {
		addrList, err := net.LookupIP(u.Host)
		if err != nil {
			return false, fmt.Errorf("migrate from '%v' failed: unknown hostname", u.Host)
		} else {
			for _, addr := range addrList {
				// workaround with **privateNetworkList** for RFC 1918, as long as "net" do not support it
				// https://github.com/golang/go/issues/29146
				if !addr.IsGlobalUnicast() || privateNetworkList.Match(addr.String()) {
					return false, fmt.Errorf("migrate from '%v' not allowed, has private network address '%s'", u.Host, addr.String())
				}
			}
		}
	}

	return true, nil
}

// MigrateRepository migrate repository according MigrateOptions
func MigrateRepository(ctx context.Context, doer *models.User, ownerName string, opts base.MigrateOptions) (*models.Repository, error) {
	allowed, err := isMigrateURLAllowed(opts.CloneAddr)
	if !allowed {
		return nil, err
	}

	var (
		downloader base.Downloader
		uploader   = NewGiteaLocalUploader(ctx, doer, ownerName, opts.RepoName)
	)

	for _, factory := range factories {
		if factory.GitServiceType() == opts.GitServiceType {
			downloader, err = factory.New(ctx, opts)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if downloader == nil {
		opts.Wiki = true
		opts.Milestones = false
		opts.Labels = false
		opts.Releases = false
		opts.Comments = false
		opts.Issues = false
		opts.PullRequests = false
		downloader = NewPlainGitDownloader(ownerName, opts.RepoName, opts.CloneAddr)
		log.Trace("Will migrate from git: %s", opts.OriginalURL)
	}

	uploader.gitServiceType = opts.GitServiceType

	if setting.Migrations.MaxAttempts > 1 {
		downloader = base.NewRetryDownloader(ctx, downloader, setting.Migrations.MaxAttempts, setting.Migrations.RetryBackoff)
	}

	if err := migrateRepository(downloader, uploader, opts); err != nil {
		if err1 := uploader.Rollback(); err1 != nil {
			log.Error("rollback failed: %v", err1)
		}

		if err2 := models.CreateRepositoryNotice(fmt.Sprintf("Migrate repository from %s failed: %v", opts.OriginalURL, err)); err2 != nil {
			log.Error("create repository notice failed: ", err2)
		}
		return nil, err
	}

	return uploader.repo, nil
}

// migrateRepository will download information and then upload it to Uploader, this is a simple
// process for small repository. For a big repository, save all the data to disk
// before upload is better
func migrateRepository(downloader base.Downloader, uploader base.Uploader, opts base.MigrateOptions) error {
	repo, err := downloader.GetRepoInfo()
	if err != nil {
		return err
	}
	repo.IsPrivate = opts.Private
	repo.IsMirror = opts.Mirror
	if opts.Description != "" {
		repo.Description = opts.Description
	}
	log.Trace("migrating git data")
	if err := uploader.CreateRepo(repo, opts); err != nil {
		return err
	}
	defer uploader.Close()

	log.Trace("migrating topics")
	topics, err := downloader.GetTopics()
	if err != nil {
		return err
	}
	if len(topics) > 0 {
		if err := uploader.CreateTopics(topics...); err != nil {
			return err
		}
	}

	if opts.Milestones {
		log.Trace("migrating milestones")
		milestones, err := downloader.GetMilestones()
		if err != nil {
			return err
		}

		msBatchSize := uploader.MaxBatchInsertSize("milestone")
		for len(milestones) > 0 {
			if len(milestones) < msBatchSize {
				msBatchSize = len(milestones)
			}

			if err := uploader.CreateMilestones(milestones...); err != nil {
				return err
			}
			milestones = milestones[msBatchSize:]
		}
	}

	if opts.Labels {
		log.Trace("migrating labels")
		labels, err := downloader.GetLabels()
		if err != nil {
			return err
		}

		lbBatchSize := uploader.MaxBatchInsertSize("label")
		for len(labels) > 0 {
			if len(labels) < lbBatchSize {
				lbBatchSize = len(labels)
			}

			if err := uploader.CreateLabels(labels...); err != nil {
				return err
			}
			labels = labels[lbBatchSize:]
		}
	}

	if opts.Releases {
		log.Trace("migrating releases")
		releases, err := downloader.GetReleases()
		if err != nil {
			return err
		}

		relBatchSize := uploader.MaxBatchInsertSize("release")
		for len(releases) > 0 {
			if len(releases) < relBatchSize {
				relBatchSize = len(releases)
			}

			if err := uploader.CreateReleases(downloader, releases[:relBatchSize]...); err != nil {
				return err
			}
			releases = releases[relBatchSize:]
		}

		// Once all releases (if any) are inserted, sync any remaining non-release tags
		if err := uploader.SyncTags(); err != nil {
			return err
		}
	}

	var (
		commentBatchSize = uploader.MaxBatchInsertSize("comment")
		reviewBatchSize  = uploader.MaxBatchInsertSize("review")
	)

	if opts.Issues {
		log.Trace("migrating issues and comments")
		var issueBatchSize = uploader.MaxBatchInsertSize("issue")

		for i := 1; ; i++ {
			issues, isEnd, err := downloader.GetIssues(i, issueBatchSize)
			if err != nil {
				return err
			}

			if err := uploader.CreateIssues(issues...); err != nil {
				return err
			}

			if !opts.Comments {
				continue
			}

			var allComments = make([]*base.Comment, 0, commentBatchSize)
			for _, issue := range issues {
				comments, err := downloader.GetComments(issue.Number)
				if err != nil {
					return err
				}

				allComments = append(allComments, comments...)

				if len(allComments) >= commentBatchSize {
					if err := uploader.CreateComments(allComments[:commentBatchSize]...); err != nil {
						return err
					}

					allComments = allComments[commentBatchSize:]
				}
			}

			if len(allComments) > 0 {
				if err := uploader.CreateComments(allComments...); err != nil {
					return err
				}
			}

			if isEnd {
				break
			}
		}
	}

	if opts.PullRequests {
		log.Trace("migrating pull requests and comments")
		var prBatchSize = uploader.MaxBatchInsertSize("pullrequest")
		for i := 1; ; i++ {
			prs, isEnd, err := downloader.GetPullRequests(i, prBatchSize)
			if err != nil {
				return err
			}

			if err := uploader.CreatePullRequests(prs...); err != nil {
				return err
			}

			if !opts.Comments {
				continue
			}

			// plain comments
			var allComments = make([]*base.Comment, 0, commentBatchSize)
			for _, pr := range prs {
				comments, err := downloader.GetComments(pr.Number)
				if err != nil {
					return err
				}

				allComments = append(allComments, comments...)

				if len(allComments) >= commentBatchSize {
					if err := uploader.CreateComments(allComments[:commentBatchSize]...); err != nil {
						return err
					}
					allComments = allComments[commentBatchSize:]
				}
			}
			if len(allComments) > 0 {
				if err := uploader.CreateComments(allComments...); err != nil {
					return err
				}
			}

			// migrate reviews
			var allReviews = make([]*base.Review, 0, reviewBatchSize)
			for _, pr := range prs {
				number := pr.Number

				// on gitlab migrations pull number change
				if pr.OriginalNumber > 0 {
					number = pr.OriginalNumber
				}

				reviews, err := downloader.GetReviews(number)
				if pr.OriginalNumber > 0 {
					for i := range reviews {
						reviews[i].IssueIndex = pr.Number
					}
				}
				if err != nil {
					return err
				}

				allReviews = append(allReviews, reviews...)

				if len(allReviews) >= reviewBatchSize {
					if err := uploader.CreateReviews(allReviews[:reviewBatchSize]...); err != nil {
						return err
					}
					allReviews = allReviews[reviewBatchSize:]
				}
			}
			if len(allReviews) > 0 {
				if err := uploader.CreateReviews(allReviews...); err != nil {
					return err
				}
			}

			if isEnd {
				break
			}
		}
	}

	return nil
}

// Init migrations service
func Init() error {
	var err error
	allowList, err = matchlist.NewMatchlist(setting.Migrations.AllowlistedDomains...)
	if err != nil {
		return fmt.Errorf("init migration allowList domains failed: %v", err)
	}

	blockList, err = matchlist.NewMatchlist(setting.Migrations.BlocklistedDomains...)
	if err != nil {
		return fmt.Errorf("init migration blockList domains failed: %v", err)
	}

	// TODO: remove if https://github.com/golang/go/issues/29146 got resolved
	privateNetworkList, _ = matchlist.NewMatchlist(
		"localhost",                                     // localhost
		"{10,127}\\.[0-9]*\\.[0-9]*\\.[0-9]*",           // 127.0.0.0/8 & 10.0.0.0/8
		"172\\.{1[6-9],2[0-9],3[01]}\\.[0-9]*\\.[0-9]*", // 172.16.0.0/12
		"192\\.168\\.[0-9]*\\.[0-9]*",                   // 192.168.0.0/16
	)

	return nil
}
