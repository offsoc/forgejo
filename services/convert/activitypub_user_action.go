// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"time"

	activities_model "code.gitea.io/gitea/models/activities"
	issues_model "code.gitea.io/gitea/models/issues"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
)

func ActionToForgeUserActivity(ctx context.Context, action *activities_model.Action) (fm.ForgeUserActivity, error) {
	render := func(format string, args ...any) string {
		return fmt.Sprintf(`<a href="%s" rel="nofollow">%s</a> %s`, action.ActUser.HTMLURL(), action.GetActDisplayName(ctx), fmt.Sprintf(format, args...))
	}
	renderIssue := func(issue *issues_model.Issue) string {
		return fmt.Sprintf(`<a href="%s" rel="nofollow">%s#%d</a>`,
			issue.HTMLURL(),
			action.GetRepoPath(ctx),
			issue.Index,
		)
	}
	renderRepo := func() string {
		return fmt.Sprintf(`<a href="%s" rel="nofollow">%s</a>`, action.Repo.HTMLURL(), action.GetRepoPath(ctx))
	}
	renderBranch := func() string {
		return fmt.Sprintf(`<a href="%s" rel="nofollow">%s</a>`, action.GetRefLink(ctx), action.GetBranch())
	}
	renderTag := func() string {
		return fmt.Sprintf(`<a href="%s" rel="nofollow">%s</a>`, action.GetRefLink(ctx), action.GetTag())
	}

	makeUserActivity := func(format string, args ...any) (fm.ForgeUserActivity, error) {
		return fm.NewForgeUserActivity(ctx, action.ActUser, action.ID, render(format, args...))
	}

	switch action.OpType {
	case activities_model.ActionCreateRepo:
		return makeUserActivity("created a new repository: %s", renderRepo())
	case activities_model.ActionRenameRepo:
		return makeUserActivity("renamed a repository: %s", renderRepo())
	case activities_model.ActionStarRepo:
		return makeUserActivity("starred a repository: %s", renderRepo())
	case activities_model.ActionWatchRepo:
		return makeUserActivity("started watching a repository: %s", renderRepo())
	case activities_model.ActionCommitRepo:
		type PushCommit struct {
			Sha1           string
			Message        string
			AuthorEmail    string
			AuthorName     string
			CommitterEmail string
			CommitterName  string
			Timestamp      time.Time
		}
		type PushCommits struct {
			Commits    []*PushCommit
			HeadCommit *PushCommit
			CompareURL string
			Len        int
		}

		commits := &PushCommits{}
		if err := json.Unmarshal([]byte(action.GetContent()), commits); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		commitsHTML := ""
		renderCommit := func(commit *PushCommit) string {
			return fmt.Sprintf(`<li><a href="%s" rel="nofollow">%s</a> <pre>%s</pre></li>`,
				fmt.Sprintf("%s/commit/%s", action.GetRepoAbsoluteLink(ctx), url.PathEscape(commit.Sha1)),
				commit.Sha1,
				html.EscapeString(commit.Message),
			)
		}
		for _, commit := range commits.Commits {
			commitsHTML += renderCommit(commit)
		}
		return makeUserActivity("pushed to %s at %s: <ul>%s</ul>", renderBranch(), renderRepo(), commitsHTML)
	case activities_model.ActionCreateIssue:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("opened issue %s", renderIssue(action.Issue))
	case activities_model.ActionCreatePullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("opened pull request %s", renderIssue(action.Issue))
	case activities_model.ActionTransferRepo:
		return makeUserActivity("transferred %s", renderRepo())
	case activities_model.ActionPushTag:
		return makeUserActivity("pushed %s at %s", renderTag(), renderRepo())
	case activities_model.ActionCommentIssue:
		renderedComment, err := markdown.RenderString(&markup.RenderContext{
			Ctx: ctx,
		}, action.Comment.Content)
		if err != nil {
			return fm.ForgeUserActivity{}, err
		}

		return makeUserActivity(`<a href="%s" rel="nofollow">commented</a> on %s: <blockquote>%s</blockquote>`,
			action.GetCommentHTMLURL(ctx),
			renderIssue(action.Comment.Issue),
			renderedComment,
		)
	case activities_model.ActionMergePullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("merged pull request %s", renderIssue(action.Issue))
	case activities_model.ActionCloseIssue:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("closed issue %s", renderIssue(action.Issue))
	case activities_model.ActionReopenIssue:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("reopened issue %s", renderIssue(action.Issue))
	case activities_model.ActionClosePullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("closed pull request %s", renderIssue(action.Issue))
	case activities_model.ActionReopenPullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("reopened pull request %s", renderIssue(action.Issue))
	case activities_model.ActionDeleteTag:
		return makeUserActivity("deleted tag %s at %s", action.GetTag(), renderRepo())
	case activities_model.ActionDeleteBranch:
		return makeUserActivity("deleted branch %s at %s", action.GetBranch(), renderRepo())
	case activities_model.ActionApprovePullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("approved pull request %s", renderIssue(action.Issue))
	case activities_model.ActionRejectPullRequest:
		if err := action.LoadIssue(ctx); err != nil {
			return fm.ForgeUserActivity{}, err
		}
		return makeUserActivity("rejected pull request %s", renderIssue(action.Issue))
	case activities_model.ActionCommentPull:
		renderedComment, err := markdown.RenderString(&markup.RenderContext{
			Ctx: ctx,
		}, action.Comment.Content)
		if err != nil {
			return fm.ForgeUserActivity{}, err
		}

		return makeUserActivity(`<a href="%s" rel="nofollow">commented</a> on %s: <blockquote>%s</blockquote>`,
			action.GetCommentHTMLURL(ctx),
			renderIssue(action.Comment.Issue),
			renderedComment,
		)
	case activities_model.ActionMirrorSyncPush:
	case activities_model.ActionMirrorSyncCreate:
	case activities_model.ActionMirrorSyncDelete:
	case activities_model.ActionPublishRelease:
	case activities_model.ActionPullReviewDismissed:
	case activities_model.ActionPullRequestReadyForReview:
	case activities_model.ActionAutoMergePullRequest:
	}

	return makeUserActivity("performed an unrecognised action: %s", action.OpType.String())
}
