// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"context"
	"html/template"
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/translation"

	"github.com/stretchr/testify/assert"
)

const testInput = `  space @mention-user  
/just/a/path.bin
https://example.com/file.bin
[local link](file.bin)
[remote link](https://example.com)
[[local link|file.bin]]
[[remote link|https://example.com]]
![local image](image.jpg)
![remote image](https://example.com/image.jpg)
[[local image|image.jpg]]
[[remote link|https://example.com/image.jpg]]
https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
:+1:
mail@domain.com
@mention-user test
#123
  space
` + "`code :+1: #123 code`\n"

var testMetas = map[string]string{
	"user":     "user13",
	"repo":     "repo11",
	"repoPath": "../../tests/gitea-repositories-meta/user13/repo11.git/",
	"mode":     "comment",
}

func TestApostrophesInMentions(t *testing.T) {
	rendered := RenderMarkdownToHtml(context.Background(), "@mention-user's comment")
	assert.EqualValues(t, template.HTML("<p><a href=\"/mention-user\" class=\"mention\" rel=\"nofollow\">@mention-user</a>&#39;s comment</p>\n"), rendered)
}

func TestNonExistantUserMention(t *testing.T) {
	rendered := RenderMarkdownToHtml(context.Background(), "@ThisUserDoesNotExist @mention-user")
	assert.EqualValues(t, template.HTML("<p>@ThisUserDoesNotExist <a href=\"/mention-user\" class=\"mention\" rel=\"nofollow\">@mention-user</a></p>\n"), rendered)
}

func TestRenderCommitBody(t *testing.T) {
	type args struct {
		ctx   context.Context
		msg   string
		metas map[string]string
	}
	tests := []struct {
		name string
		args args
		want template.HTML
	}{
		{
			name: "multiple lines",
			args: args{
				ctx: context.Background(),
				msg: "first line\nsecond line",
			},
			want: "second line",
		},
		{
			name: "multiple lines with leading newlines",
			args: args{
				ctx: context.Background(),
				msg: "\n\n\n\nfirst line\nsecond line",
			},
			want: "second line",
		},
		{
			name: "multiple lines with trailing newlines",
			args: args{
				ctx: context.Background(),
				msg: "first line\nsecond line\n\n\n",
			},
			want: "second line",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, RenderCommitBody(tt.args.ctx, tt.args.msg, tt.args.metas), "RenderCommitBody(%v, %v, %v)", tt.args.ctx, tt.args.msg, tt.args.metas)
		})
	}

	expected := `/just/a/path.bin
<a href="https://example.com/file.bin" class="link">https://example.com/file.bin</a>
[local link](file.bin)
[remote link](<a href="https://example.com" class="link">https://example.com</a>)
[[local link|file.bin]]
[[remote link|<a href="https://example.com" class="link">https://example.com</a>]]
![local image](image.jpg)
![remote image](<a href="https://example.com/image.jpg" class="link">https://example.com/image.jpg</a>)
[[local image|image.jpg]]
[[remote link|<a href="https://example.com/image.jpg" class="link">https://example.com/image.jpg</a>]]
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" class="compare"><code class="nohighlight">88fc37a3c0...12fc37a3c0 (hash)</code></a>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" class="commit"><code class="nohighlight">88fc37a3c0</code></a>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span>
<a href="mailto:mail@domain.com" class="mailto">mail@domain.com</a>
<a href="/mention-user" class="mention">@mention-user</a> test
<a href="/user13/repo11/issues/123" class="ref-issue">#123</a>
  space
` + "`code <span class=\"emoji\" aria-label=\"thumbs up\" data-alias=\"+1\">👍</span> <a href=\"/user13/repo11/issues/123\" class=\"ref-issue\">#123</a> code`"
	assert.EqualValues(t, expected, RenderCommitBody(context.Background(), testInput, testMetas))
}

func TestRenderCommitMessage(t *testing.T) {
	expected := `space <a href="/mention-user" class="mention">@mention-user</a>  `

	assert.EqualValues(t, expected, RenderCommitMessage(context.Background(), testInput, testMetas))
}

func TestRenderCommitMessageLinkSubject(t *testing.T) {
	expected := `<a href="https://example.com/link" class="default-link muted">space </a><a href="/mention-user" class="mention">@mention-user</a>`

	assert.EqualValues(t, expected, RenderCommitMessageLinkSubject(context.Background(), testInput, "https://example.com/link", testMetas))
}

func TestRenderIssueTitle(t *testing.T) {
	expected := `  space @mention-user  
/just/a/path.bin
https://example.com/file.bin
[local link](file.bin)
[remote link](https://example.com)
[[local link|file.bin]]
[[remote link|https://example.com]]
![local image](image.jpg)
![remote image](https://example.com/image.jpg)
[[local image|image.jpg]]
[[remote link|https://example.com/image.jpg]]
https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span>
mail@domain.com
@mention-user test
<a href="/user13/repo11/issues/123" class="ref-issue">#123</a>
  space
<code class="inline-code-block">code :+1: #123 code</code>
`
	assert.EqualValues(t, expected, RenderIssueTitle(context.Background(), testInput, testMetas))
}

func TestRenderRefIssueTitle(t *testing.T) {
	expected := `  space @mention-user  
/just/a/path.bin
https://example.com/file.bin
[local link](file.bin)
[remote link](https://example.com)
[[local link|file.bin]]
[[remote link|https://example.com]]
![local image](image.jpg)
![remote image](https://example.com/image.jpg)
[[local image|image.jpg]]
[[remote link|https://example.com/image.jpg]]
https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span>
mail@domain.com
@mention-user test
#123
  space
<code class="inline-code-block">code :+1: #123 code</code>
`
	assert.EqualValues(t, expected, RenderRefIssueTitle(context.Background(), testInput))
}

func TestRenderMarkdownToHtml(t *testing.T) {
	expected := `<p>space <a href="/mention-user" class="mention" rel="nofollow">@mention-user</a><br/>
/just/a/path.bin
<a href="https://example.com/file.bin" rel="nofollow">https://example.com/file.bin</a>
<a href="/file.bin" rel="nofollow">local link</a>
<a href="https://example.com" rel="nofollow">remote link</a>
<a href="/src/file.bin" rel="nofollow">local link</a>
<a href="https://example.com" rel="nofollow">remote link</a>
<a href="/image.jpg" target="_blank" rel="nofollow noopener"><img src="/image.jpg" alt="local image"/></a>
<a href="https://example.com/image.jpg" target="_blank" rel="nofollow noopener"><img src="https://example.com/image.jpg" alt="remote image"/></a>
<a href="/image.jpg" rel="nofollow"><img src="/image.jpg" title="local image" alt=""/></a>
<a href="https://example.com/image.jpg" rel="nofollow"><img src="https://example.com/image.jpg" title="remote link" alt=""/></a>
<a href="https://example.com/user/repo/compare/88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb#hash" rel="nofollow"><code>88fc37a3c0...12fc37a3c0 (hash)</code></a>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb...12fc37a3c0a4dda553bdcfc80c178a58247f42fb pare
<a href="https://example.com/user/repo/commit/88fc37a3c0a4dda553bdcfc80c178a58247f42fb" rel="nofollow"><code>88fc37a3c0</code></a>
com 88fc37a3c0a4dda553bdcfc80c178a58247f42fb mit
<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span>
<a href="mailto:mail@domain.com" rel="nofollow">mail@domain.com</a>
<a href="/mention-user" class="mention" rel="nofollow">@mention-user</a> test
#123
space
<code>code :+1: #123 code</code></p>
`
	assert.EqualValues(t, expected, RenderMarkdownToHtml(context.Background(), testInput))
}

func TestRenderLabels(t *testing.T) {
	unittest.PrepareTestEnv(t)

	tr := &translation.MockLocale{}
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})

	assert.Contains(t, RenderLabels(db.DefaultContext, tr, []*issues_model.Label{label}, "user2/repo1", false),
		"user2/repo1/issues?labels=1")
	assert.Contains(t, RenderLabels(db.DefaultContext, tr, []*issues_model.Label{label}, "user2/repo1", true),
		"user2/repo1/pulls?labels=1")
}
