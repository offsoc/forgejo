// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forgejo.org/models/db"
	issue_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/card"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/services/context"
)

// drawUser draws a user avatar in a summary card
func drawUser(ctx *context.Context, card *card.Card, user *user_model.User) error {
	if user.UseCustomAvatar {
		posterAvatarPath := user.CustomAvatarRelativePath()
		if posterAvatarPath != "" {
			userAvatarFile, err := storage.Avatars.Open(user.CustomAvatarRelativePath())
			if err != nil {
				return err
			}
			userAvatarImage, _, err := image.Decode(userAvatarFile)
			if err != nil {
				return err
			}
			card.DrawImage(userAvatarImage)
		}
	} else {
		posterAvatarLink := user.AvatarLinkWithSize(ctx, 256)
		card.DrawExternalImage(posterAvatarLink)
	}
	return nil
}

// drawRepoIcon draws the repo icon in a summary card
func drawRepoIcon(ctx *context.Context, card *card.Card, repo *repo_model.Repository) error {
	repoAvatarPath := repo.CustomAvatarRelativePath()

	if repoAvatarPath != "" {
		repoAvatarFile, err := storage.RepoAvatars.Open(repoAvatarPath)
		if err != nil {
			return err
		}
		repoAvatarImage, _, err := image.Decode(repoAvatarFile)
		if err != nil {
			return err
		}
		card.DrawImage(repoAvatarImage)
		return nil
	}

	// If the repo didn't have an avatar, fallback to the repo owner's avatar for the right-hand-side icon
	err := repo.LoadOwner(ctx)
	if err != nil {
		return err
	}
	if repo.Owner != nil {
		err = drawUser(ctx, card, repo.Owner)
		if err != nil {
			return err
		}
	}

	return nil
}

// hexToColor converts a hex color to a go color
func hexToColor(colorStr string) (*color.RGBA, error) {
	colorStr = strings.TrimLeft(colorStr, "#")

	b, err := hex.DecodeString(colorStr)
	if err != nil {
		return nil, err
	}

	if len(b) < 3 {
		return nil, fmt.Errorf("expected at least 3 bytes from DecodeString, got %d", len(b))
	}

	color := color.RGBA{b[0], b[1], b[2], 255}

	return &color, nil
}

func drawLanguagesCard(ctx *context.Context, card *card.Card) error {
	languageList, err := repo_model.GetTopLanguageStats(ctx, ctx.Repo.Repository, 5)
	if err != nil {
		return err
	}
	if len(languageList) == 0 {
		card.DrawRect(0, 0, card.Width, card.Height, color.White)
		return nil
	}

	currentX := 0
	var langColor *color.RGBA

	for _, lang := range languageList {
		langColor, err = hexToColor(lang.Color)
		if err != nil {
			return err
		}

		langWidth := float32(card.Width) * (lang.Percentage / 100)
		card.DrawRect(currentX, 0, currentX+int(langWidth), card.Width, langColor)
		currentX += int(langWidth)
	}

	if currentX < card.Width {
		card.DrawRect(currentX, 0, card.Width, card.Height, langColor)
	}

	return nil
}

func drawRepoSummaryCard(ctx *context.Context, repo *repo_model.Repository) (*card.Card, error) {
	width, height := card.DefaultSize()
	mainCard, err := card.NewCard(width, height)
	if err != nil {
		return nil, err
	}

	contentCard, languageBarCard := mainCard.Split(false, 90)

	contentCard.SetMargin(60)
	topSection, bottomSection := contentCard.Split(false, 75)
	issueSummary, issueIcon := topSection.Split(true, 80)
	repoInfo, issueDescription := issueSummary.Split(false, 30)

	repoInfo.SetMargin(10)
	_, err = repoInfo.DrawText(repo.FullName(), color.Black, 56, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	issueDescription.SetMargin(10)
	_, err = issueDescription.DrawText(repo.Description, color.Gray{128}, 36, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	issueIcon.SetMargin(10)
	err = drawRepoIcon(ctx, issueIcon, repo)
	if err != nil {
		return nil, err
	}

	topCountCard, bottomCountCard := bottomSection.Split(false, 50)

	releaseCount, err := db.Count[repo_model.Release](ctx, repo_model.FindReleasesOptions{
		// only show draft releases for users who can write, read-only users shouldn't see draft releases.
		IncludeDrafts: ctx.Repo.CanWrite(unit_model.TypeReleases),
		RepoID:        ctx.Repo.Repository.ID,
	})
	if err != nil {
		return nil, err
	}

	starsText := ctx.Locale.TrN(
		repo.NumStars,
		"explore.stars_one",
		"explore.stars_few",
		repo.NumStars,
	)
	forksText := ctx.Locale.TrN(
		repo.NumForks,
		"explore.forks_one",
		"explore.forks_few",
		repo.NumForks,
	)
	releasesText := ctx.Locale.TrN(
		releaseCount,
		"repo.activity.title.releases_1",
		"repo.activity.title.releases_n",
		releaseCount,
	)

	topCountText := fmt.Sprintf("%s • %s • %s", starsText, forksText, releasesText)

	topCountCard.SetMargin(10)
	_, err = topCountCard.DrawText(topCountText, color.Gray{128}, 36, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	issuesText := ctx.Locale.TrN(
		repo.NumOpenIssues,
		"repo.activity.title.issues_1",
		"repo.activity.title.issues_n",
		repo.NumOpenIssues,
	)
	pullRequestsText := ctx.Locale.TrN(
		repo.NumOpenPulls,
		"repo.activity.title.prs_1",
		"repo.activity.title.prs_n",
		repo.NumOpenPulls,
	)

	bottomCountText := fmt.Sprintf("%s • %s", issuesText, pullRequestsText)

	bottomCountCard.SetMargin(10)
	_, err = bottomCountCard.DrawText(bottomCountText, color.Gray{128}, 36, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	err = drawLanguagesCard(ctx, languageBarCard)
	if err != nil {
		return nil, err
	}

	return mainCard, nil
}

func drawIssueSummaryCard(ctx *context.Context, issue *issue_model.Issue) (*card.Card, error) {
	width, height := card.DefaultSize()
	mainCard, err := card.NewCard(width, height)
	if err != nil {
		return nil, err
	}

	mainCard.SetMargin(60)
	topSection, bottomSection := mainCard.Split(false, 75)
	issueSummary, issueIcon := topSection.Split(true, 80)
	repoInfo, issueDescription := issueSummary.Split(false, 15)

	repoInfo.SetMargin(10)
	_, err = repoInfo.DrawText(fmt.Sprintf("%s - #%d", issue.Repo.FullName(), issue.Index), color.Gray{128}, 36, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	issueDescription.SetMargin(10)
	_, err = issueDescription.DrawText(issue.Title, color.Black, 56, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	issueIcon.SetMargin(10)
	err = drawRepoIcon(ctx, issueIcon, issue.Repo)
	if err != nil {
		return nil, err
	}

	issueStats, issueAttribution := bottomSection.Split(false, 50)

	var state string
	if issue.IsPull && issue.PullRequest.HasMerged {
		if issue.PullRequest.Status == 3 {
			state = ctx.Locale.TrString("repo.pulls.manually_merged")
		} else {
			state = ctx.Locale.TrString("repo.pulls.merged")
		}
	} else if issue.IsClosed {
		state = ctx.Locale.TrString("repo.issues.closed_title")
	} else if issue.IsPull {
		if issue.PullRequest.IsWorkInProgress(ctx) {
			state = ctx.Locale.TrString("repo.issues.draft_title")
		} else {
			state = ctx.Locale.TrString("repo.issues.open_title")
		}
	} else {
		state = ctx.Locale.TrString("repo.issues.open_title")
	}
	state = strings.ToLower(state)

	issueStats.SetMargin(10)
	if issue.IsPull {
		reviews := map[int64]bool{}
		for _, comment := range issue.Comments {
			if comment.Review != nil {
				reviews[comment.Review.ID] = true
			}
		}
		_, err = issueStats.DrawText(
			fmt.Sprintf("%s, %s, %s",
				ctx.Locale.TrN(
					issue.NumComments,
					"repo.issues.num_comments_1",
					"repo.issues.num_comments",
					issue.NumComments,
				),
				ctx.Locale.TrN(
					len(reviews),
					"repo.issues.num_reviews_one",
					"repo.issues.num_reviews_few",
					len(reviews),
				),
				state,
			),
			color.Gray{128}, 36, card.Top, card.Left)
	} else {
		_, err = issueStats.DrawText(
			fmt.Sprintf("%s, %s",
				ctx.Locale.TrN(
					issue.NumComments,
					"repo.issues.num_comments_1",
					"repo.issues.num_comments",
					issue.NumComments,
				),
				state,
			),
			color.Gray{128}, 36, card.Top, card.Left)
	}
	if err != nil {
		return nil, err
	}

	issueAttributionIcon, issueAttributionText := issueAttribution.Split(true, 8)
	issueAttributionText.SetMargin(5)
	_, err = issueAttributionText.DrawText(
		fmt.Sprintf(
			"%s - %s",
			issue.Poster.Name,
			issue.Created.AsTime().Format(time.DateOnly),
		),
		color.Gray{128}, 36, card.Middle, card.Left)
	if err != nil {
		return nil, err
	}
	err = drawUser(ctx, issueAttributionIcon, issue.Poster)
	if err != nil {
		return nil, err
	}

	return mainCard, nil
}

func drawReleaseSummaryCard(ctx *context.Context, release *repo_model.Release) (*card.Card, error) {
	width, height := card.DefaultSize()
	mainCard, err := card.NewCard(width, height)
	if err != nil {
		return nil, err
	}

	mainCard.SetMargin(60)
	topSection, bottomSection := mainCard.Split(false, 75)
	releaseSummary, repoIcon := topSection.Split(true, 80)
	repoInfo, releaseDescription := releaseSummary.Split(false, 15)

	repoInfo.SetMargin(10)
	_, err = repoInfo.DrawText(release.Repo.FullName(), color.Gray{128}, 36, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	releaseDescription.SetMargin(10)
	_, err = releaseDescription.DrawText(release.DisplayName(), color.Black, 56, card.Top, card.Left)
	if err != nil {
		return nil, err
	}

	repoIcon.SetMargin(10)
	err = drawRepoIcon(ctx, repoIcon, release.Repo)
	if err != nil {
		return nil, err
	}

	downloadCountCard, releaseDateCard := bottomSection.Split(true, 75)

	downloadCount, err := release.GetTotalDownloadCount(ctx)
	if err != nil {
		return nil, err
	}

	downloadCountText := ctx.Locale.TrN(
		strconv.FormatInt(downloadCount, 10),
		"repo.release.download_count_one",
		"repo.release.download_count_few",
		strconv.FormatInt(downloadCount, 10),
	)

	_, err = downloadCountCard.DrawText(string(downloadCountText), color.Gray{128}, 36, card.Bottom, card.Left)
	if err != nil {
		return nil, err
	}

	_, err = releaseDateCard.DrawText(release.CreatedUnix.AsTime().Format(time.DateOnly), color.Gray{128}, 36, card.Bottom, card.Left)
	if err != nil {
		return nil, err
	}

	return mainCard, nil
}

// checkCardCache checks if a card in cache and serves it
func checkCardCache(ctx *context.Context, cacheKey string) bool {
	cache := cache.GetCache()
	pngData, ok := cache.Get(cacheKey).([]byte)
	if ok && pngData != nil && len(pngData) > 0 {
		ctx.Resp.Header().Set("Content-Type", "image/png")
		ctx.Resp.WriteHeader(http.StatusOK)
		_, err := ctx.Resp.Write(pngData)
		if err != nil {
			ctx.ServerError("GetSummaryCard", err)
		}
		return true
	}

	return false
}

// serveCard server a Card to the user adds it to the cache
func serveCard(ctx *context.Context, card *card.Card, cacheKey string) {
	cache := cache.GetCache()

	// Encode image, store in cache
	var imageBuffer bytes.Buffer
	err := png.Encode(&imageBuffer, card.Img)
	if err != nil {
		ctx.ServerError("GetSummaryCard", err)
		return
	}
	imageBytes := imageBuffer.Bytes()
	err = cache.Put(cacheKey, imageBytes, setting.CacheService.TTLSeconds())
	if err != nil {
		// don't abort serving the image if we just had a cache storage failure
		log.Warn("failed to cache issue summary card: %v", err)
	}

	// Finish the uncached image response
	ctx.Resp.Header().Set("Content-Type", "image/png")
	ctx.Resp.WriteHeader(http.StatusOK)
	_, err = ctx.Resp.Write(imageBytes)
	if err != nil {
		ctx.ServerError("GetSummaryCard", err)
		return
	}
}

func DrawRepoSummaryCard(ctx *context.Context) {
	cacheKey := fmt.Sprintf("summary_card:repo:%s:%d", ctx.Locale.Language(), ctx.Repo.Repository.ID)

	if checkCardCache(ctx, cacheKey) {
		return
	}

	card, err := drawRepoSummaryCard(ctx, ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("drawRepoSummaryCar", err)
		return
	}

	serveCard(ctx, card, cacheKey)
}

func DrawIssueSummaryCard(ctx *context.Context) {
	issue, err := issue_model.GetIssueWithAttrsByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issue_model.IsErrIssueNotExist(err) {
			ctx.Error(http.StatusNotFound)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err.Error())
		}
		return
	}

	if !ctx.Repo.CanReadIssuesOrPulls(issue.IsPull) {
		ctx.Error(http.StatusNotFound)
		return
	}

	cacheKey := fmt.Sprintf("summary_card:issue:%s:%d", ctx.Locale.Language(), issue.ID)

	if checkCardCache(ctx, cacheKey) {
		return
	}

	card, err := drawIssueSummaryCard(ctx, issue)
	if err != nil {
		ctx.ServerError("drawIssueSummaryCar", err)
		return
	}

	serveCard(ctx, card, cacheKey)
}

func DrawReleaseSummaryCard(ctx *context.Context) {
	release, err := repo_model.GetRelease(ctx, ctx.Repo.Repository.ID, ctx.Params("*"))
	if err != nil {
		if repo_model.IsErrReleaseNotExist(err) {
			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetReleaseForRepoByID", err)
		}
		return
	}

	err = release.LoadRepo(ctx)
	if err != nil {
		ctx.ServerError("LoadRepo", err)
		return
	}

	cacheKey := fmt.Sprintf("summary_card:release:%s:%d", ctx.Locale.Language(), release.ID)

	if checkCardCache(ctx, cacheKey) {
		return
	}

	card, err := drawReleaseSummaryCard(ctx, release)
	if err != nil {
		ctx.ServerError("drawRepoSummaryCar", err)
		return
	}

	serveCard(ctx, card, cacheKey)
}
