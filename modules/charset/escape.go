// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:generate go run invisible/generate.go -v -o ./invisible_gen.go

//go:generate go run ambiguous/generate.go -v -o ./ambiguous_gen.go ambiguous/ambiguous.json

package charset

import (
	"html/template"
	"io"
	"slices"
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation"
)

// RuneNBSP is the codepoint for NBSP
const RuneNBSP = 0xa0

type escapeContext string

// Keep this consistent with the documentation of [ui].SKIP_ESCAPE_CONTEXTS
// Defines the different contexts that could be used to escape in.
const (
	// Wiki pages.
	WikiContext escapeContext = "wiki"
	// Rendered content (except markup), source code and blames.
	FileviewContext escapeContext = "file-view"
	// Commits or pull requet's diff.
	DiffContext escapeContext = "diff"
)

// EscapeControlHTML escapes the unicode control sequences in a provided html document
func EscapeControlHTML(html template.HTML, locale translation.Locale, context escapeContext, allowed ...rune) (escaped *EscapeStatus, output template.HTML) {
	sb := &strings.Builder{}
	escaped, _ = EscapeControlReader(strings.NewReader(string(html)), sb, locale, context, allowed...) // err has been handled in EscapeControlReader
	return escaped, template.HTML(sb.String())
}

// EscapeControlReader escapes the unicode control sequences in a provided reader of HTML content and writer in a locale and returns the findings as an EscapeStatus
func EscapeControlReader(reader io.Reader, writer io.Writer, locale translation.Locale, context escapeContext, allowed ...rune) (escaped *EscapeStatus, err error) {
	if !setting.UI.AmbiguousUnicodeDetection || slices.Contains(setting.UI.SkipEscapeContexts, string(context)) {
		_, err = io.Copy(writer, reader)
		return &EscapeStatus{}, err
	}
	outputStream := &HTMLStreamerWriter{Writer: writer}
	streamer := NewEscapeStreamer(locale, outputStream, allowed...).(*escapeStreamer)

	if err = StreamHTML(reader, streamer); err != nil {
		streamer.escaped.HasError = true
		log.Error("Error whilst escaping: %v", err)
	}
	return streamer.escaped, err
}
