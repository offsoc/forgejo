// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markup

import (
	"strings"
	"testing"

	"forgejo.org/modules/git"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	AppURL    = "http://localhost:3000/"
	Repo      = "gogits/gogs"
	AppSubURL = AppURL + Repo + "/"
)

func TestRender_StandardLinks(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = AppSubURL

	test := func(input, expected string) {
		buffer, err := RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: setting.AppSubURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	// No BranchPath or TreePath set.
	test("[[file:comfy][comfy]]",
		`<p><a href="http://localhost:3000/gogits/gogs/comfy">comfy</a></p>`)

	test("[[https://google.com/]]",
		`<p><a href="https://google.com/">https://google.com/</a></p>`)

	lnk := util.URLJoin(AppSubURL, "WikiPage")
	test("[[WikiPage][WikiPage]]",
		`<p><a href="`+lnk+`">WikiPage</a></p>`)
}

func TestRender_BaseLinks(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = AppSubURL

	testBranch := func(input, expected string) {
		buffer, err := RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       setting.AppSubURL,
				BranchPath: "branch/main",
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	testBranchTree := func(input, expected string) {
		buffer, err := RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base:       setting.AppSubURL,
				BranchPath: "branch/main",
				TreePath:   "deep/nested/folder",
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	testBranch("[[file:comfy][comfy]]",
		`<p><a href="http://localhost:3000/gogits/gogs/src/branch/main/comfy">comfy</a></p>`)
	testBranchTree("[[file:comfy][comfy]]",
		`<p><a href="http://localhost:3000/gogits/gogs/src/branch/main/deep/nested/folder/comfy">comfy</a></p>`)

	testBranch("[[file:./src][./src/]]",
		`<p><a href="http://localhost:3000/gogits/gogs/src/branch/main/src">./src/</a></p>`)
	testBranchTree("[[file:./src][./src/]]",
		`<p><a href="http://localhost:3000/gogits/gogs/src/branch/main/deep/nested/folder/src">./src/</a></p>`)
}

func TestRender_Media(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = AppSubURL

	test := func(input, expected string) {
		buffer, err := RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
			Links: markup.Links{
				Base: setting.AppSubURL,
			},
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	url := "../../.images/src/02/train.jpg"
	result := util.URLJoin(AppSubURL, url)

	test("[[file:"+url+"]]",
		`<p><img src="`+result+`" alt="`+result+`" /></p>`)

	// With description.
	test("[[https://example.com][https://example.com/example.svg]]",
		`<p><a href="https://example.com"><img src="https://example.com/example.svg" alt="https://example.com/example.svg" /></a></p>`)
	test("[[https://example.com][pre https://example.com/example.svg post]]",
		`<p><a href="https://example.com">pre <img src="https://example.com/example.svg" alt="https://example.com/example.svg" /> post</a></p>`)
	test("[[https://example.com][https://example.com/example.mp4]]",
		`<p><a href="https://example.com"><video src="https://example.com/example.mp4">https://example.com/example.mp4</video></a></p>`)
	test("[[https://example.com][pre https://example.com/example.mp4 post]]",
		`<p><a href="https://example.com">pre <video src="https://example.com/example.mp4">https://example.com/example.mp4</video> post</a></p>`)

	// Without description.
	test("[[https://example.com/example.svg]]",
		`<p><img src="https://example.com/example.svg" alt="https://example.com/example.svg" /></p>`)
	test("[[https://example.com/example.mp4]]",
		`<p><video src="https://example.com/example.mp4">https://example.com/example.mp4</video></p>`)

	// Text description.
	test("[[file:./lem-post.png][file:./lem-post.png]]",
		`<p><a href="http://localhost:3000/gogits/gogs/lem-post.png"><img src="http://localhost:3000/gogits/gogs/lem-post.png" alt="http://localhost:3000/gogits/gogs/lem-post.png" /></a></p>`)
	test("[[file:./lem-post.mp4][file:./lem-post.mp4]]",
		`<p><a href="http://localhost:3000/gogits/gogs/lem-post.mp4"><video src="http://localhost:3000/gogits/gogs/lem-post.mp4">http://localhost:3000/gogits/gogs/lem-post.mp4</video></a></p>`)
}

func TestRender_Source(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = AppSubURL

	test := func(input, expected string) {
		buffer, err := RenderString(&markup.RenderContext{
			Ctx: git.DefaultContext,
		}, input)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(buffer))
	}

	test(`#+begin_src go
// HelloWorld prints "Hello World"
func HelloWorld() {
	fmt.Println("Hello World")
}
#+end_src
`, `<div class="src src-go">
<pre><code class="chroma language-go"><span class="c1">// HelloWorld prints &#34;Hello World&#34;</span>
<span class="kd">func</span> <span class="nf">HelloWorld</span><span class="p">()</span> <span class="p">{</span>
	<span class="nx">fmt</span><span class="p">.</span><span class="nf">Println</span><span class="p">(</span><span class="s">&#34;Hello World&#34;</span><span class="p">)</span>
<span class="p">}</span></code></pre>
</div>`)
}
