// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"mime/quotedprintable"
	"regexp"
	"strings"
	"testing"
	texttmpl "text/template"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const subjectTpl = `
{{.SubjectPrefix}}[{{.Repo}}] @{{.Doer.Name}} #{{.Issue.Index}} - {{.Issue.Title}}
`

const bodyTpl = `
<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
</head>

<body>
	<p>{{.Body}}</p>
	<p>
		---
		<br>
		<a href="{{.Link}}">View it on Gitea</a>.
	</p>
</body>
</html>
`

func prepareMailerTest(t *testing.T) (doer *user_model.User, repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) {
	require.NoError(t, unittest.PrepareTestDatabase())

	doer = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, Owner: doer})
	issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1, Repo: repo, Poster: doer})
	require.NoError(t, issue.LoadRepo(db.DefaultContext))
	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2, Issue: issue})
	return doer, repo, issue, comment
}

func TestComposeIssueCommentMessage(t *testing.T) {
	defer MockMailSettings(nil)()
	doer, _, issue, comment := prepareMailerTest(t)

	markup.Init(&markup.ProcessorHelper{
		IsUsernameMentionable: func(ctx context.Context, username string) bool {
			return username == doer.Name
		},
	})

	defer test.MockVariableValue(&setting.IncomingEmail.Enabled, true)()

	subjectTemplates = texttmpl.Must(texttmpl.New("issue/comment").Parse(subjectTpl))
	bodyTemplates = template.Must(template.New("issue/comment").Parse(bodyTpl))

	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}, {Name: "Test2", Email: "test2@gitea.com"}}
	msgs, err := composeIssueCommentMessages(&mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: activities_model.ActionCommentIssue,
		Content: fmt.Sprintf("test @%s %s#%d body", doer.Name, issue.Repo.FullName(), issue.Index),
		Comment: comment,
	}, "en-US", recipients, false, "issue comment")
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	gomailMsg := msgs[0].ToMessage()
	replyTo := gomailMsg.GetHeader("Reply-To")[0]
	subject := gomailMsg.GetHeader("Subject")[0]

	assert.Len(t, gomailMsg.GetHeader("To"), 1, "exactly one recipient is expected in the To field")
	tokenRegex := regexp.MustCompile(`\Aincoming\+(.+)@localhost\z`)
	assert.Regexp(t, tokenRegex, replyTo)
	token := tokenRegex.FindAllStringSubmatch(replyTo, 1)[0][1]
	assert.Equal(t, "Re: ", subject[:4], "Comment reply subject should contain Re:")
	assert.Equal(t, "Re: [user2/repo1] @user2 #1 - issue1", subject)
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", gomailMsg.GetHeader("In-Reply-To")[0], "In-Reply-To header doesn't match")
	assert.ElementsMatch(t, []string{"<user2/repo1/issues/1@localhost>", "<reply-" + token + "@localhost>"}, gomailMsg.GetHeader("References"), "References header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1/comment/2@localhost>", gomailMsg.GetHeader("Message-ID")[0], "Message-ID header doesn't match")
	assert.Equal(t, "<mailto:"+replyTo+">", gomailMsg.GetHeader("List-Post")[0])
	assert.Len(t, gomailMsg.GetHeader("List-Unsubscribe"), 2) // url + mailto

	var buf bytes.Buffer
	gomailMsg.WriteTo(&buf)

	b, err := io.ReadAll(quotedprintable.NewReader(&buf))
	require.NoError(t, err)

	// text/plain
	assert.Contains(t, string(b), fmt.Sprintf(`( %s )`, doer.HTMLURL()))
	assert.Contains(t, string(b), fmt.Sprintf(`( %s )`, issue.HTMLURL()))

	// text/html
	assert.Contains(t, string(b), fmt.Sprintf(`href="%s"`, doer.HTMLURL()))
	assert.Contains(t, string(b), fmt.Sprintf(`href="%s"`, issue.HTMLURL()))
}

func TestComposeIssueMessage(t *testing.T) {
	defer MockMailSettings(nil)()
	doer, _, issue, _ := prepareMailerTest(t)

	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}, {Name: "Test2", Email: "test2@gitea.com"}}
	msgs, err := composeIssueCommentMessages(&mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: activities_model.ActionCreateIssue,
		Content: "test body",
	}, "en-US", recipients, false, "issue create")
	require.NoError(t, err)
	assert.Len(t, msgs, 2)

	gomailMsg := msgs[0].ToMessage()
	mailto := gomailMsg.GetHeader("To")
	subject := gomailMsg.GetHeader("Subject")
	messageID := gomailMsg.GetHeader("Message-ID")
	inReplyTo := gomailMsg.GetHeader("In-Reply-To")
	references := gomailMsg.GetHeader("References")

	assert.Len(t, mailto, 1, "exactly one recipient is expected in the To field")
	assert.Equal(t, "[user2/repo1] issue1 (Issue #1)", subject[0])
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", inReplyTo[0], "In-Reply-To header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", references[0], "References header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", messageID[0], "Message-ID header doesn't match")
	assert.Empty(t, gomailMsg.GetHeader("List-Post"))         // incoming mail feature disabled
	assert.Len(t, gomailMsg.GetHeader("List-Unsubscribe"), 1) // url without mailto
}

func TestMailerIssueTemplate(t *testing.T) {
	defer MockMailSettings(nil)()
	require.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	expect := func(t *testing.T, msg *Message, issue *issues_model.Issue, expected ...string) {
		subject := msg.ToMessage().GetHeader("Subject")
		msgbuf := new(bytes.Buffer)
		_, _ = msg.ToMessage().WriteTo(msgbuf)
		wholemsg := msgbuf.String()
		assert.Contains(t, subject[0], fallbackMailSubject(issue))
		for _, s := range expected {
			assert.Contains(t, wholemsg, s)
		}
		AssertTranslatedLocale(t, wholemsg, "mail.issue")
	}

	testCompose := func(t *testing.T, ctx *mailCommentContext) *Message {
		t.Helper()
		recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}}

		ctx.Context = t.Context()
		fromMention := false
		msgs, err := composeIssueCommentMessages(ctx, "en-US", recipients, fromMention, "TestMailerIssueTemplate")
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		return msgs[0]
	}

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	require.NoError(t, issue.LoadRepo(db.DefaultContext))

	msg := testCompose(t, &mailCommentContext{
		Issue: issue, Doer: doer, ActionType: activities_model.ActionCreateIssue,
		Content: issue.Content,
	})
	expect(t, msg, issue, issue.Content)

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2, Issue: issue})

	msg = testCompose(t, &mailCommentContext{
		Issue: issue, Doer: doer, ActionType: activities_model.ActionCommentIssue,
		Content: comment.Content, Comment: comment,
	})
	expect(t, msg, issue, comment.Content)

	msg = testCompose(t, &mailCommentContext{
		Issue: issue, Doer: doer, ActionType: activities_model.ActionCloseIssue,
		Content: comment.Content, Comment: comment,
	})
	expect(t, msg, issue, comment.Content)

	msg = testCompose(t, &mailCommentContext{
		Issue: issue, Doer: doer, ActionType: activities_model.ActionReopenIssue,
		Content: comment.Content, Comment: comment,
	})
	expect(t, msg, issue, comment.Content)

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	require.NoError(t, pull.LoadAttributes(db.DefaultContext))
	pullComment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4, Issue: pull})

	msg = testCompose(t, &mailCommentContext{
		Issue: pull, Doer: doer, ActionType: activities_model.ActionCommentPull,
		Content: pullComment.Content, Comment: pullComment,
	})
	expect(t, msg, pull, pullComment.Content)

	msg = testCompose(t, &mailCommentContext{
		Issue: pull, Doer: doer, ActionType: activities_model.ActionMergePullRequest,
		Content: pullComment.Content, Comment: pullComment,
	})
	expect(t, msg, pull, pullComment.Content, pull.PullRequest.BaseBranch)

	reviewComment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 9})
	require.NoError(t, reviewComment.LoadReview(db.DefaultContext))

	approveComment := reviewComment
	approveComment.Review.Type = issues_model.ReviewTypeApprove
	msg = testCompose(t, &mailCommentContext{
		Issue: pull, Doer: doer, ActionType: activities_model.ActionApprovePullRequest,
		Content: approveComment.Content, Comment: approveComment,
	})
	expect(t, msg, pull, approveComment.Content)

	rejectComment := reviewComment
	rejectComment.Review.Type = issues_model.ReviewTypeReject
	msg = testCompose(t, &mailCommentContext{
		Issue: pull, Doer: doer, ActionType: activities_model.ActionRejectPullRequest,
		Content: rejectComment.Content, Comment: rejectComment,
	})
	expect(t, msg, pull, rejectComment.Content)
}

func TestTemplateSelection(t *testing.T) {
	defer MockMailSettings(nil)()
	doer, repo, issue, comment := prepareMailerTest(t)
	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}}

	subjectTemplates = texttmpl.Must(texttmpl.New("issue/default").Parse("issue/default/subject"))
	texttmpl.Must(subjectTemplates.New("issue/new").Parse("issue/new/subject"))
	texttmpl.Must(subjectTemplates.New("pull/comment").Parse("pull/comment/subject"))
	texttmpl.Must(subjectTemplates.New("issue/close").Parse("")) // Must default to fallback subject

	bodyTemplates = template.Must(template.New("issue/default").Parse("issue/default/body"))
	template.Must(bodyTemplates.New("issue/new").Parse("issue/new/body"))
	template.Must(bodyTemplates.New("pull/comment").Parse("pull/comment/body"))
	template.Must(bodyTemplates.New("issue/close").Parse("issue/close/body"))

	expect := func(t *testing.T, msg *Message, expSubject, expBody string) {
		subject := msg.ToMessage().GetHeader("Subject")
		msgbuf := new(bytes.Buffer)
		_, _ = msg.ToMessage().WriteTo(msgbuf)
		wholemsg := msgbuf.String()
		assert.Equal(t, []string{expSubject}, subject)
		assert.Contains(t, wholemsg, expBody)
	}

	msg := testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: activities_model.ActionCreateIssue,
		Content: "test body",
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "issue/new/subject", "issue/new/body")

	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: activities_model.ActionCommentIssue,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "issue/default/subject", "issue/default/body")

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2, Repo: repo, Poster: doer})
	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4, Issue: pull})
	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   pull, Doer: doer, ActionType: activities_model.ActionCommentPull,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "pull/comment/subject", "pull/comment/body")

	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: t.Context(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: activities_model.ActionCloseIssue,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "Re: [user2/repo1] issue1 (Issue #1)", "issue/close/body")
}

func TestTemplateServices(t *testing.T) {
	defer MockMailSettings(nil)()
	doer, _, issue, comment := prepareMailerTest(t)
	require.NoError(t, issue.LoadRepo(db.DefaultContext))

	expect := func(t *testing.T, issue *issues_model.Issue, comment *issues_model.Comment, doer *user_model.User,
		actionType activities_model.ActionType, fromMention bool, tplSubject, tplBody, expSubject, expBody string,
	) {
		subjectTemplates = texttmpl.Must(texttmpl.New("issue/default").Parse(tplSubject))
		bodyTemplates = template.Must(template.New("issue/default").Parse(tplBody))

		recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}}
		msg := testComposeIssueCommentMessage(t, &mailCommentContext{
			Context: t.Context(), // TODO: use a correct context
			Issue:   issue, Doer: doer, ActionType: actionType,
			Content: "test body", Comment: comment,
		}, recipients, fromMention, "TestTemplateServices")

		subject := msg.ToMessage().GetHeader("Subject")
		msgbuf := new(bytes.Buffer)
		_, _ = msg.ToMessage().WriteTo(msgbuf)
		wholemsg := msgbuf.String()

		assert.Equal(t, []string{expSubject}, subject)
		assert.Contains(t, wholemsg, "\r\n"+expBody+"\r\n")
	}

	expect(t, issue, comment, doer, activities_model.ActionCommentIssue, false,
		"{{.SubjectPrefix}}[{{.Repo}}]: @{{.Doer.Name}} commented on #{{.Issue.Index}} - {{.Issue.Title}}",
		"//{{.ActionType}},{{.ActionName}},{{if .IsMention}}norender{{end}}//",
		"Re: [user2/repo1]: @user2 commented on #1 - issue1",
		"//issue,comment,//")

	expect(t, issue, comment, doer, activities_model.ActionCommentIssue, true,
		"{{if .IsMention}}must render{{end}}",
		"//subject is: {{.Subject}}//",
		"must render",
		"//subject is: must render//")

	expect(t, issue, comment, doer, activities_model.ActionCommentIssue, true,
		"{{.FallbackSubject}}",
		"//{{.SubjectPrefix}}//",
		"Re: [user2/repo1] issue1 (Issue #1)",
		"//Re: //")
}

func testComposeIssueCommentMessage(t *testing.T, ctx *mailCommentContext, recipients []*user_model.User, fromMention bool, info string) *Message {
	msgs, err := composeIssueCommentMessages(ctx, "en-US", recipients, fromMention, info)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	return msgs[0]
}

func TestGenerateAdditionalHeaders(t *testing.T) {
	defer MockMailSettings(nil)()
	doer, _, issue, _ := prepareMailerTest(t)

	ctx := &mailCommentContext{Context: t.Context() /* TODO: use a correct context */, Issue: issue, Doer: doer}
	recipient := &user_model.User{Name: "test", Email: "test@gitea.com"}

	headers := generateAdditionalHeaders(ctx, "dummy-reason", recipient)

	expected := map[string]string{
		"List-ID":                   "user2/repo1 <repo1.user2.localhost>",
		"List-Archive":              "<https://try.gitea.io/user2/repo1>",
		"X-Mailer":                  "Forgejo",
		"X-Gitea-Reason":            "dummy-reason",
		"X-Gitea-Sender":            "user2",
		"X-Gitea-Recipient":         "test",
		"X-Gitea-Recipient-Address": "test@gitea.com",
		"X-Gitea-Repository":        "repo1",
		"X-Gitea-Repository-Path":   "user2/repo1",
		"X-Gitea-Repository-Link":   "https://try.gitea.io/user2/repo1",
		"X-Gitea-Issue-ID":          "1",
		"X-Gitea-Issue-Link":        "https://try.gitea.io/user2/repo1/issues/1",
		"X-Forgejo-Sender":          "user2",
		"X-Forgejo-Recipient":       "test",
	}

	for key, value := range expected {
		if assert.Contains(t, headers, key) {
			assert.Equal(t, value, headers[key])
		}
	}
}

func Test_createReference(t *testing.T) {
	defer MockMailSettings(nil)()
	_, _, issue, comment := prepareMailerTest(t)
	_, _, pullIssue, _ := prepareMailerTest(t)
	pullIssue.IsPull = true

	type args struct {
		issue      *issues_model.Issue
		comment    *issues_model.Comment
		actionType activities_model.ActionType
	}
	tests := []struct {
		name   string
		args   args
		prefix string
	}{
		{
			name: "Open Issue",
			args: args{
				issue:      issue,
				actionType: activities_model.ActionCreateIssue,
			},
			prefix: fmt.Sprintf("<%s/issues/%d@%s>", issue.Repo.FullName(), issue.Index, setting.Domain),
		},
		{
			name: "Open Pull",
			args: args{
				issue:      pullIssue,
				actionType: activities_model.ActionCreatePullRequest,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d@%s>", issue.Repo.FullName(), issue.Index, setting.Domain),
		},
		{
			name: "Comment Issue",
			args: args{
				issue:      issue,
				comment:    comment,
				actionType: activities_model.ActionCommentIssue,
			},
			prefix: fmt.Sprintf("<%s/issues/%d/comment/%d@%s>", issue.Repo.FullName(), issue.Index, comment.ID, setting.Domain),
		},
		{
			name: "Comment Pull",
			args: args{
				issue:      pullIssue,
				comment:    comment,
				actionType: activities_model.ActionCommentPull,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d/comment/%d@%s>", issue.Repo.FullName(), issue.Index, comment.ID, setting.Domain),
		},
		{
			name: "Close Issue",
			args: args{
				issue:      issue,
				actionType: activities_model.ActionCloseIssue,
			},
			prefix: fmt.Sprintf("<%s/issues/%d/close/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Close Pull",
			args: args{
				issue:      pullIssue,
				actionType: activities_model.ActionClosePullRequest,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d/close/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Reopen Issue",
			args: args{
				issue:      issue,
				actionType: activities_model.ActionReopenIssue,
			},
			prefix: fmt.Sprintf("<%s/issues/%d/reopen/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Reopen Pull",
			args: args{
				issue:      pullIssue,
				actionType: activities_model.ActionReopenPullRequest,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d/reopen/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Merge Pull",
			args: args{
				issue:      pullIssue,
				actionType: activities_model.ActionMergePullRequest,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d/merge/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Ready Pull",
			args: args{
				issue:      pullIssue,
				actionType: activities_model.ActionPullRequestReadyForReview,
			},
			prefix: fmt.Sprintf("<%s/pulls/%d/ready/", issue.Repo.FullName(), issue.Index),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createReference(tt.args.issue, tt.args.comment, tt.args.actionType)
			if !strings.HasPrefix(got, tt.prefix) {
				t.Errorf("createReference() = %v, want %v", got, tt.prefix)
			}
		})
	}
}

func TestFromDisplayName(t *testing.T) {
	template, err := texttmpl.New("mailFrom").Parse("{{ .DisplayName }}")
	require.NoError(t, err)
	defer test.MockVariableValue(&setting.MailService, &setting.Mailer{FromDisplayNameFormatTemplate: template})()

	tests := []struct {
		userDisplayName string
		fromDisplayName string
	}{{
		userDisplayName: "test",
		fromDisplayName: "test",
	}, {
		userDisplayName: "Hi Its <Mee>",
		fromDisplayName: "Hi Its <Mee>",
	}, {
		userDisplayName: "Æsir",
		fromDisplayName: "=?utf-8?q?=C3=86sir?=",
	}, {
		userDisplayName: "new😀user",
		fromDisplayName: "=?utf-8?q?new=F0=9F=98=80user?=",
	}}

	for _, tc := range tests {
		t.Run(tc.userDisplayName, func(t *testing.T) {
			user := &user_model.User{FullName: tc.userDisplayName, Name: "tmp"}
			got := fromDisplayName(user)
			assert.Equal(t, tc.fromDisplayName, got)
		})
	}

	t.Run("template with all available vars", func(t *testing.T) {
		template, err = texttmpl.New("mailFrom").Parse("{{ .DisplayName }} (by {{ .AppName }} on [{{ .Domain }}])")
		require.NoError(t, err)
		defer test.MockVariableValue(&setting.MailService, &setting.Mailer{FromDisplayNameFormatTemplate: template})()
		defer test.MockVariableValue(&setting.AppName, "Code IT")()
		defer test.MockVariableValue(&setting.Domain, "code.it")()

		assert.Equal(t, "Mister X (by Code IT on [code.it])", fromDisplayName(&user_model.User{FullName: "Mister X", Name: "tmp"}))
	})
}

func TestFallbackSubjectType(t *testing.T) {
	_, _, issue, _ := prepareMailerTest(t)
	assert.Contains(t, fallbackMailSubject(issue), "Issue")
	_, _, pr, _ := prepareMailerTest(t)
	pr.IsPull = true
	assert.Contains(t, fallbackMailSubject(pr), "PR")
}
