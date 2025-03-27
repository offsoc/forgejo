// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package callout

import (
	"strings"

	"forgejo.org/modules/svg"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type GitHubCalloutTransformer struct{}

// Transform transforms the given AST tree.
func (g *GitHubCalloutTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	supportedAttentionTypes := map[string]bool{
		"note":      true,
		"tip":       true,
		"important": true,
		"warning":   true,
		"caution":   true,
	}

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if v, ok := n.(*ast.Blockquote); ok {
			if v.ChildCount() == 0 {
				return ast.WalkContinue, nil
			}

			// We only want attention blockquotes when the AST looks like:
			// Text: "["
			// Text: "!TYPE"
			// Text(SoftLineBreak): "]"

			// grab these nodes and make sure we adhere to the attention blockquote structure
			firstParagraph := v.FirstChild()
			if firstParagraph.ChildCount() < 3 {
				return ast.WalkContinue, nil
			}
			firstTextNode, ok := firstParagraph.FirstChild().(*ast.Text)
			if !ok || string(firstTextNode.Value(reader.Source())) != "[" {
				return ast.WalkContinue, nil
			}
			secondTextNode, ok := firstTextNode.NextSibling().(*ast.Text)
			if !ok {
				return ast.WalkContinue, nil
			}
			// If the second node's text isn't one of the supported attention
			// types, continue walking.
			secondTextNodeText := secondTextNode.Value(reader.Source())
			attentionType := strings.ToLower(strings.TrimPrefix(string(secondTextNodeText), "!"))
			if _, has := supportedAttentionTypes[attentionType]; !has {
				return ast.WalkContinue, nil
			}

			thirdTextNode, ok := secondTextNode.NextSibling().(*ast.Text)
			if !ok || string(thirdTextNode.Value(reader.Source())) != "]" {
				return ast.WalkContinue, nil
			}

			// color the blockquote
			v.SetAttributeString("class", []byte("attention-header attention-"+attentionType))

			// create an emphasis to make it bold
			attentionParagraph := ast.NewParagraph()
			attentionParagraph.SetAttributeString("class", []byte("attention-title"))
			emphasis := ast.NewEmphasis(2)
			emphasis.SetAttributeString("class", []byte("attention-"+attentionType))
			firstParagraph.InsertBefore(firstParagraph, firstTextNode, emphasis)

			// capitalize first letter
			attentionText := ast.NewString([]byte(strings.ToUpper(string(attentionType[0])) + attentionType[1:]))

			// replace the ![TYPE] with a dedicated paragraph of icon+Type
			emphasis.AppendChild(emphasis, attentionText)
			attentionParagraph.AppendChild(attentionParagraph, NewAttention(attentionType))
			attentionParagraph.AppendChild(attentionParagraph, emphasis)
			firstParagraph.Parent().InsertBefore(firstParagraph.Parent(), firstParagraph, attentionParagraph)
			firstParagraph.RemoveChild(firstParagraph, firstTextNode)
			firstParagraph.RemoveChild(firstParagraph, secondTextNode)
			firstParagraph.RemoveChild(firstParagraph, thirdTextNode)
		}
		return ast.WalkContinue, nil
	})
}

type GitHubCalloutHTMLRenderer struct {
	html.Config
}

// RegisterFuncs implements renderer.NodeRenderer.RegisterFuncs.
func (r *GitHubCalloutHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAttention, r.renderAttention)
}

// renderAttention renders a quote marked with i.e. "> **Note**" or "> **Warning**" with a corresponding svg
func (r *GitHubCalloutHTMLRenderer) renderAttention(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*Attention)

		var octiconName string
		switch n.AttentionType {
		case "note":
			octiconName = "info"
		case "tip":
			octiconName = "light-bulb"
		case "important":
			octiconName = "report"
		case "warning":
			octiconName = "alert"
		case "caution":
			octiconName = "stop"
		default:
			octiconName = "info"
		}
		_, _ = w.WriteString(string(svg.RenderHTML("octicon-"+octiconName, 16, "attention-icon attention-"+n.AttentionType)))
	}
	return ast.WalkContinue, nil
}

func NewGitHubCalloutHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &GitHubCalloutHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}
