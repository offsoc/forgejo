// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package code

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	"forgejo.org/modules/highlight"
	"forgejo.org/modules/indexer/code/internal"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/gitdiff"
)

// Result a search result to display
type Result struct {
	RepoID      int64
	Filename    string
	CommitID    string
	UpdatedUnix timeutil.TimeStamp
	Language    string
	Color       string
	Lines       []ResultLine
}

type ResultLine struct {
	Num              int
	FormattedContent template.HTML
}

type SearchResultLanguages = internal.SearchResultLanguages

type SearchOptions = internal.SearchOptions

var CodeSearchOptions = [2]string{"exact", "union"}

type SearchMode = internal.CodeSearchMode

const (
	SearchModeExact = internal.CodeSearchModeExact
	SearchModeUnion = internal.CodeSearchModeUnion
)

func indices(content string, selectionStartIndex, selectionEndIndex int) (int, int) {
	startIndex := selectionStartIndex
	numLinesBefore := 0
	for ; startIndex > 0; startIndex-- {
		if content[startIndex-1] == '\n' {
			if numLinesBefore == 1 {
				break
			}
			numLinesBefore++
		}
	}

	endIndex := selectionEndIndex
	numLinesAfter := 0
	for ; endIndex < len(content); endIndex++ {
		if content[endIndex] == '\n' {
			if numLinesAfter == 1 {
				break
			}
			numLinesAfter++
		}
	}

	return startIndex, endIndex
}

func writeStrings(buf *bytes.Buffer, strs ...string) error {
	for _, s := range strs {
		_, err := buf.WriteString(s)
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	highlightTagStart = "<span class=\"search-highlight\">"
	highlightTagEnd   = "</span>"
)

func HighlightSearchResultCode(filename string, lineNums []int, highlightRanges [][3]int, code string) []ResultLine {
	hcd := gitdiff.NewHighlightCodeDiff()
	hcd.CollectUsedRunes(code)
	startTag, endTag := hcd.NextPlaceholder(), hcd.NextPlaceholder()
	hcd.PlaceholderTokenMap[startTag] = highlightTagStart
	hcd.PlaceholderTokenMap[endTag] = highlightTagEnd

	// we should highlight the whole code block first, otherwise it doesn't work well with multiple line highlighting
	hl, _ := highlight.Code(filename, "", code)
	conv := hcd.ConvertToPlaceholders(string(hl))
	convLines := strings.Split(conv, "\n")

	// each highlightRange is of the form [line number, start pos, end pos]
	for _, highlightRange := range highlightRanges {
		ln, start, end := highlightRange[0], highlightRange[1], highlightRange[2]
		line := convLines[ln]
		if line == "" || len(line) <= start || len(line) < end {
			continue
		}

		sb := strings.Builder{}
		count := -1
		isOpen := false
		for _, r := range line {
			if token, ok := hcd.PlaceholderTokenMap[r];
			// token was not found
			!ok ||
				// token was marked as used
				token == "" ||
				// the token is not an valid html tag emitted by chroma
				!(len(token) > 6 && (token[0:5] == "<span" || token[0:6] == "</span")) {
				count++
			} else if !isOpen {
				// open the tag only after all other placeholders
				sb.WriteRune(r)
				continue
			} else if isOpen && count < end {
				// if the tag is open, but a placeholder exists in between
				// close the tag
				sb.WriteRune(endTag)
				// write the placeholder
				sb.WriteRune(r)
				// reopen the tag
				sb.WriteRune(startTag)
				continue
			}

			switch count {
			case end:
				// if tag is not open, no need to close
				if !isOpen {
					break
				}
				sb.WriteRune(endTag)
				isOpen = false
			case start:
				// if tag is open, do not open again
				if isOpen {
					break
				}
				isOpen = true
				sb.WriteRune(startTag)
			}

			sb.WriteRune(r)
		}
		if isOpen {
			sb.WriteRune(endTag)
		}
		convLines[ln] = sb.String()
	}
	conv = strings.Join(convLines, "\n")

	highlightedLines := strings.Split(hcd.Recover(conv), "\n")
	// The lineNums outputted by highlight.Code might not match the original lineNums, because "highlight" removes the last `\n`
	lines := make([]ResultLine, min(len(highlightedLines), len(lineNums)))
	for i := 0; i < len(lines); i++ {
		lines[i].Num = lineNums[i]
		lines[i].FormattedContent = template.HTML(highlightedLines[i])
	}
	return lines
}

func searchResult(result *internal.SearchResult, startIndex, endIndex int) (*Result, error) {
	startLineNum := 1 + strings.Count(result.Content[:startIndex], "\n")

	var formattedLinesBuffer bytes.Buffer

	contentLines := strings.SplitAfter(result.Content[startIndex:endIndex], "\n")
	lineNums := make([]int, 0, len(contentLines))
	index := startIndex
	var highlightRanges [][3]int
	for i, line := range contentLines {
		var err error
		if index < result.EndIndex &&
			result.StartIndex < index+len(line) &&
			result.StartIndex < result.EndIndex {
			openActiveIndex := max(result.StartIndex-index, 0)
			closeActiveIndex := min(result.EndIndex-index, len(line))
			highlightRanges = append(highlightRanges, [3]int{i, openActiveIndex, closeActiveIndex})
			err = writeStrings(&formattedLinesBuffer,
				line[:openActiveIndex],
				line[openActiveIndex:closeActiveIndex],
				line[closeActiveIndex:],
			)
		} else {
			err = writeStrings(&formattedLinesBuffer, line)
		}
		if err != nil {
			return nil, err
		}

		lineNums = append(lineNums, startLineNum+i)
		index += len(line)
	}

	return &Result{
		RepoID:      result.RepoID,
		Filename:    result.Filename,
		CommitID:    result.CommitID,
		UpdatedUnix: result.UpdatedUnix,
		Language:    result.Language,
		Color:       result.Color,
		Lines:       HighlightSearchResultCode(result.Filename, lineNums, highlightRanges, formattedLinesBuffer.String()),
	}, nil
}

// PerformSearch perform a search on a repository
func PerformSearch(ctx context.Context, opts *SearchOptions) (int, []*Result, []*SearchResultLanguages, error) {
	if opts == nil || len(opts.Keyword) == 0 {
		return 0, nil, nil, nil
	}

	total, results, resultLanguages, err := (*globalIndexer.Load()).Search(ctx, opts)
	if err != nil {
		return 0, nil, nil, err
	}

	displayResults := make([]*Result, len(results))

	for i, result := range results {
		startIndex, endIndex := indices(result.Content, result.StartIndex, result.EndIndex)
		displayResults[i], err = searchResult(result, startIndex, endIndex)
		if err != nil {
			return 0, nil, nil, err
		}
	}
	return int(total), displayResults, resultLanguages, nil
}
