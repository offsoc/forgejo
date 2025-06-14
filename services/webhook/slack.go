// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"
	gitea_context "forgejo.org/services/context"
	"forgejo.org/services/forms"
	"forgejo.org/services/webhook/shared"

	"code.forgejo.org/go-chi/binding"
)

type slackHandler struct{}

func (slackHandler) Type() webhook_module.HookType { return webhook_module.SLACK }
func (slackHandler) Icon(size int) template.HTML   { return shared.ImgIcon("slack.png", size) }

type slackForm struct {
	forms.WebhookCoreForm
	PayloadURL string `binding:"Required;ValidUrl"`
	Channel    string `binding:"Required"`
	Username   string
	IconURL    string
	Color      string
}

var _ binding.Validator = &slackForm{}

// Validate implements binding.Validator.
func (s *slackForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := gitea_context.GetWebContext(req)
	if !IsValidSlackChannel(strings.TrimSpace(s.Channel)) {
		errs = append(errs, binding.Error{
			FieldNames:     []string{"Channel"},
			Classification: "",
			Message:        ctx.Locale.TrString("repo.settings.add_webhook.invalid_channel_name"),
		})
	}
	return errs
}

func (slackHandler) UnmarshalForm(bind func(any)) forms.WebhookForm {
	var form slackForm
	bind(&form)

	return forms.WebhookForm{
		WebhookCoreForm: form.WebhookCoreForm,
		URL:             form.PayloadURL,
		ContentType:     webhook_model.ContentTypeJSON,
		Secret:          "",
		HTTPMethod:      http.MethodPost,
		Metadata: &SlackMeta{
			Channel:  strings.TrimSpace(form.Channel),
			Username: form.Username,
			IconURL:  form.IconURL,
			Color:    form.Color,
		},
	}
}

// SlackMeta contains the slack metadata
type SlackMeta struct {
	Channel  string `json:"channel"`
	Username string `json:"username"`
	IconURL  string `json:"icon_url"`
	Color    string `json:"color"`
}

// Metadata returns slack metadata
func (slackHandler) Metadata(w *webhook_model.Webhook) any {
	s := &SlackMeta{}
	if err := json.Unmarshal([]byte(w.Meta), s); err != nil {
		log.Error("slackHandler.Metadata(%d): %v", w.ID, err)
	}
	return s
}

// SlackPayload contains the information about the slack channel
type SlackPayload struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text"`
	Username    string            `json:"username"`
	IconURL     string            `json:"icon_url"`
	UnfurlLinks int               `json:"unfurl_links"`
	LinkNames   int               `json:"link_names"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment contains the slack message
type SlackAttachment struct {
	Fallback  string `json:"fallback"`
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link"`
	Text      string `json:"text"`
}

// SlackTextFormatter replaces &, <, > with HTML characters
// see: https://api.slack.com/docs/formatting
func SlackTextFormatter(s string) string {
	// replace & < >
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// SlackShortTextFormatter replaces &, <, > with HTML characters
func SlackShortTextFormatter(s string) string {
	s = strings.Split(s, "\n")[0]
	// replace & < >
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// SlackLinkFormatter creates a link compatible with slack
func SlackLinkFormatter(url, text string) string {
	return fmt.Sprintf("<%s|%s>", url, SlackTextFormatter(text))
}

// SlackLinkToRef slack-formatter link to a repo ref
func SlackLinkToRef(repoURL, ref string) string {
	// FIXME: SHA1 hardcoded here
	url := git.RefURL(repoURL, ref)
	refName := git.RefName(ref).ShortName()
	return SlackLinkFormatter(url, refName)
}

// TODO: fix spelling to Converter
// Create implements payloadConvertor Create method
func (s slackConvertor) Create(p *api.CreatePayload) (SlackPayload, error) {
	refLink := SlackLinkToRef(p.Repo.HTMLURL, p.Ref)
	text := fmt.Sprintf("[%s:%s] %s created by %s", p.Repo.FullName, refLink, p.RefType, p.Sender.UserName)

	return s.createPayload(text, nil), nil
}

// Delete composes Slack payload for delete a branch or tag.
func (s slackConvertor) Delete(p *api.DeletePayload) (SlackPayload, error) {
	refName := git.RefName(p.Ref).ShortName()
	repoLink := SlackLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	text := fmt.Sprintf("[%s:%s] %s deleted by %s", repoLink, refName, p.RefType, p.Sender.UserName)

	return s.createPayload(text, nil), nil
}

// Fork composes Slack payload for forked by a repository.
func (s slackConvertor) Fork(p *api.ForkPayload) (SlackPayload, error) {
	baseLink := SlackLinkFormatter(p.Forkee.HTMLURL, p.Forkee.FullName)
	forkLink := SlackLinkFormatter(p.Repo.HTMLURL, p.Repo.FullName)
	text := fmt.Sprintf("%s is forked to %s", baseLink, forkLink)

	return s.createPayload(text, nil), nil
}

// Issue implements payloadConvertor Issue method
func (s slackConvertor) Issue(p *api.IssuePayload) (SlackPayload, error) {
	text, issueTitle, attachmentText, color := getIssuesPayloadInfo(p, SlackLinkFormatter, true)

	var attachments []SlackAttachment
	if attachmentText != "" {
		attachmentText = SlackTextFormatter(attachmentText)
		issueTitle = SlackTextFormatter(issueTitle)
		attachments = append(attachments, SlackAttachment{
			Color:     fmt.Sprintf("%x", color),
			Title:     issueTitle,
			TitleLink: p.Issue.HTMLURL,
			Text:      attachmentText,
		})
	}

	return s.createPayload(text, attachments), nil
}

// IssueComment implements payloadConvertor IssueComment method
func (s slackConvertor) IssueComment(p *api.IssueCommentPayload) (SlackPayload, error) {
	text, issueTitle, color := getIssueCommentPayloadInfo(p, SlackLinkFormatter, true)

	return s.createPayload(text, []SlackAttachment{{
		Color:     fmt.Sprintf("%x", color),
		Title:     issueTitle,
		TitleLink: p.Comment.HTMLURL,
		Text:      SlackTextFormatter(p.Comment.Body),
	}}), nil
}

// Wiki implements payloadConvertor Wiki method
func (s slackConvertor) Wiki(p *api.WikiPayload) (SlackPayload, error) {
	text, _, _ := getWikiPayloadInfo(p, SlackLinkFormatter, true)

	return s.createPayload(text, nil), nil
}

// Release implements payloadConvertor Release method
func (s slackConvertor) Release(p *api.ReleasePayload) (SlackPayload, error) {
	text, _ := getReleasePayloadInfo(p, SlackLinkFormatter, true)

	return s.createPayload(text, nil), nil
}

func (s slackConvertor) Package(p *api.PackagePayload) (SlackPayload, error) {
	text, _ := getPackagePayloadInfo(p, SlackLinkFormatter, true)

	return s.createPayload(text, nil), nil
}

// Push implements payloadConvertor Push method
func (s slackConvertor) Push(p *api.PushPayload) (SlackPayload, error) {
	// n new commits
	var (
		commitDesc   string
		commitString string
	)

	if p.TotalCommits == 1 {
		commitDesc = "1 new commit"
	} else {
		commitDesc = fmt.Sprintf("%d new commits", p.TotalCommits)
	}
	if len(p.CompareURL) > 0 {
		commitString = SlackLinkFormatter(p.CompareURL, commitDesc)
	} else {
		commitString = commitDesc
	}

	branchLink := SlackLinkToRef(p.Repo.HTMLURL, p.Ref)
	text := fmt.Sprintf("[%s:%s] %s pushed by %s", p.Repo.FullName, branchLink, commitString, p.Pusher.UserName)

	var attachmentText string
	// for each commit, generate attachment text
	for i, commit := range p.Commits {
		attachmentText += fmt.Sprintf("%s: %s - %s", SlackLinkFormatter(commit.URL, commit.ID[:7]), SlackShortTextFormatter(commit.Message), SlackTextFormatter(commit.Author.Name))
		// add linebreak to each commit but the last
		if i < len(p.Commits)-1 {
			attachmentText += "\n"
		}
	}

	return s.createPayload(text, []SlackAttachment{{
		Color:     s.Color,
		Title:     p.Repo.HTMLURL,
		TitleLink: p.Repo.HTMLURL,
		Text:      attachmentText,
	}}), nil
}

// PullRequest implements payloadConvertor PullRequest method
func (s slackConvertor) PullRequest(p *api.PullRequestPayload) (SlackPayload, error) {
	text, issueTitle, attachmentText, color := getPullRequestPayloadInfo(p, SlackLinkFormatter, true)

	var attachments []SlackAttachment
	if attachmentText != "" {
		attachmentText = SlackTextFormatter(p.PullRequest.Body)
		issueTitle = SlackTextFormatter(issueTitle)
		attachments = append(attachments, SlackAttachment{
			Color:     fmt.Sprintf("%x", color),
			Title:     issueTitle,
			TitleLink: p.PullRequest.HTMLURL,
			Text:      attachmentText,
		})
	}

	return s.createPayload(text, attachments), nil
}

// Review implements payloadConvertor Review method
func (s slackConvertor) Review(p *api.PullRequestPayload, event webhook_module.HookEventType) (SlackPayload, error) {
	title := fmt.Sprintf("#%d %s", p.Index, p.PullRequest.Title)
	titleLink := fmt.Sprintf("%s/pulls/%d", p.Repository.HTMLURL, p.Index)
	var text string

	if p.Action == api.HookIssueReviewed {
		action, err := parseHookPullRequestEventType(event)
		if err != nil {
			return SlackPayload{}, err
		}

		text = fmt.Sprintf("[%s] Pull request review %s: [%s](%s) by %s", p.Repository.FullName, action, title, titleLink, p.Sender.UserName)
	}

	return s.createPayload(text, nil), nil
}

// Repository implements payloadConvertor Repository method
func (s slackConvertor) Repository(p *api.RepositoryPayload) (SlackPayload, error) {
	repoLink := SlackLinkFormatter(p.Repository.HTMLURL, p.Repository.FullName)
	var text string

	switch p.Action {
	case api.HookRepoCreated:
		text = fmt.Sprintf("[%s] Repository created by %s", repoLink, p.Sender.UserName)
	case api.HookRepoDeleted:
		text = fmt.Sprintf("[%s] Repository deleted by %s", repoLink, p.Sender.UserName)
	}

	return s.createPayload(text, nil), nil
}

func (s slackConvertor) Action(p *api.ActionPayload) (SlackPayload, error) {
	text, _ := getActionPayloadInfo(p, SlackLinkFormatter)

	return s.createPayload(text, nil), nil
}

func (s slackConvertor) createPayload(text string, attachments []SlackAttachment) SlackPayload {
	return SlackPayload{
		Channel:     s.Channel,
		Text:        text,
		Username:    s.Username,
		IconURL:     s.IconURL,
		Attachments: attachments,
	}
}

type slackConvertor struct {
	Channel  string
	Username string
	IconURL  string
	Color    string
}

var _ shared.PayloadConvertor[SlackPayload] = slackConvertor{}

func (slackHandler) NewRequest(ctx context.Context, w *webhook_model.Webhook, t *webhook_model.HookTask) (*http.Request, []byte, error) {
	meta := &SlackMeta{}
	if err := json.Unmarshal([]byte(w.Meta), meta); err != nil {
		return nil, nil, fmt.Errorf("slackHandler.NewRequest meta json: %w", err)
	}
	sc := slackConvertor{
		Channel:  meta.Channel,
		Username: meta.Username,
		IconURL:  meta.IconURL,
		Color:    meta.Color,
	}
	return shared.NewJSONRequest(sc, w, t, true)
}

var slackChannel = regexp.MustCompile(`^#?[a-z0-9_-]{1,80}$`)

// IsValidSlackChannel validates a channel name conforms to what slack expects:
// https://api.slack.com/methods/conversations.rename#naming
// Conversation names can only contain lowercase letters, numbers, hyphens, and underscores, and must be 80 characters or less.
// Forgejo accepts if it starts with a #.
func IsValidSlackChannel(name string) bool {
	return slackChannel.MatchString(name)
}
