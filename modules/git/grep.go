// Copyright 2024 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

type GrepResult struct {
	Filename          string
	LineNumbers       []int
	LineCodes         []string
	HighlightedRanges [][3]int
}

type GrepMode int

const (
	FixedGrepMode GrepMode = iota
	FixedAnyGrepMode
	RegExpGrepMode
)

var GrepSearchOptions = [3]string{"exact", "union", "regexp"}

type GrepOptions struct {
	RefName           string
	MaxResultLimit    int
	MatchesPerFile    int // >= git 2.38
	ContextLineNumber int
	Mode              GrepMode
	Filename          string
}

func (opts *GrepOptions) ensureDefaults() {
	opts.RefName = cmp.Or(opts.RefName, "HEAD")
	opts.MaxResultLimit = cmp.Or(opts.MaxResultLimit, 50)
	opts.MatchesPerFile = cmp.Or(opts.MatchesPerFile, 20)
}

func hasPrefixFold(s, t string) bool {
	if len(s) < len(t) {
		return false
	}
	return strings.EqualFold(s[:len(t)], t)
}

func GrepSearch(ctx context.Context, repo *Repository, search string, opts GrepOptions) ([]*GrepResult, error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("unable to create os pipe to grep: %w", err)
	}
	defer func() {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
	}()

	opts.ensureDefaults()

	/*
	 The output is like this ("^@" means \x00; the first number denotes the line,
	 the second number denotes the column of the first match in line):

	 HEAD:.air.toml
	 6^@8^@bin = "gitea"

	 HEAD:.changelog.yml
	 2^@10^@repo: go-gitea/gitea
	*/
	var results []*GrepResult
	// -I skips binary files
	cmd := NewCommand(ctx, "grep",
		"-I", "--null", "--break", "--heading",
		"--line-number", "--ignore-case", "--full-name")
	if opts.Mode == RegExpGrepMode {
		// No `--column` -- regexp mode does not support highlighting in the
		// current implementation as the length of the match is unknown from
		// `grep` but required for highlighting.
		cmd.AddArguments("--perl-regexp")
	} else {
		cmd.AddArguments("--fixed-strings", "--column")
	}

	cmd.AddOptionValues("--context", fmt.Sprint(opts.ContextLineNumber))

	// --max-count requires at least git 2.38
	if CheckGitVersionAtLeast("2.38.0") == nil {
		cmd.AddOptionValues("--max-count", fmt.Sprint(opts.MatchesPerFile))
	} else {
		log.Warn("git-grep: --max-count requires at least git 2.38")
	}

	words := []string{search}
	if opts.Mode == FixedAnyGrepMode {
		words = strings.Fields(search)
	}
	for _, word := range words {
		cmd.AddGitGrepExpression(word)
	}

	// pathspec
	includeLen := len(setting.Indexer.IncludePatterns)
	if len(opts.Filename) > 0 {
		includeLen = 1
	}
	files := make([]string, 0, len(setting.Indexer.ExcludePatterns)+includeLen)
	if len(opts.Filename) > 0 && len(setting.Indexer.IncludePatterns) > 0 {
		// if the both a global include pattern and the per search path is defined
		// we only include results where the path matches the globally set pattern
		// (eg, global pattern = "src/**" and path = "node_modules/")

		// FIXME: this is a bit too restrictive, and fails to consider cases where the
		// globally set include pattern refers to a file than a directory
		// (eg, global pattern = "**.go" and path = "modules/git")
		exprMatched := false
		for _, expr := range setting.Indexer.IncludePatterns {
			if expr.Match(opts.Filename) {
				files = append(files, ":(literal)"+opts.Filename)
				exprMatched = true
				break
			}
		}
		if !exprMatched {
			log.Warn("git-grep: filepath %s does not match any include pattern", opts.Filename)
		}
	} else if len(opts.Filename) > 0 {
		// if the path is only set we just include results that matches it
		files = append(files, ":(literal)"+opts.Filename)
	} else {
		// otherwise if global include patterns are set include results that strictly match them
		for _, expr := range setting.Indexer.IncludePatterns {
			files = append(files, ":"+expr.Pattern())
		}
	}
	for _, expr := range setting.Indexer.ExcludePatterns {
		files = append(files, ":^"+expr.Pattern())
	}
	cmd.AddDynamicArguments(opts.RefName).AddDashesAndList(files...)

	stderr := bytes.Buffer{}
	err = cmd.Run(&RunOpts{
		Timeout: time.Duration(setting.Git.Timeout.Grep) * time.Second,

		Dir:    repo.Path,
		Stdout: stdoutWriter,
		Stderr: &stderr,
		PipelineFunc: func(ctx context.Context, cancel context.CancelFunc) error {
			_ = stdoutWriter.Close()
			defer stdoutReader.Close()

			isInBlock := false
			scanner := bufio.NewReader(stdoutReader)
			var res *GrepResult
			for {
				line, err := scanner.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				// Remove delimiter.
				if len(line) > 0 {
					line = line[:len(line)-1]
				}

				if !isInBlock {
					if _ /* ref */, filename, ok := strings.Cut(line, ":"); ok {
						isInBlock = true
						res = &GrepResult{Filename: filename}
						results = append(results, res)
					}
					continue
				}
				if line == "" {
					if len(results) >= opts.MaxResultLimit {
						cancel()
						break
					}
					isInBlock = false
					continue
				}
				if line == "--" {
					continue
				}
				if lineNum, lineCode, ok := strings.Cut(line, "\x00"); ok {
					lineNumInt, _ := strconv.Atoi(lineNum)
					res.LineNumbers = append(res.LineNumbers, lineNumInt)
					// We support highlighting only when `--column` parameter is used.
					if lineCol, lineCode2, ok := strings.Cut(lineCode, "\x00"); ok {
						lineColInt, _ := strconv.Atoi(lineCol)
						start := lineColInt - 1
						matchLen := len(lineCode2)
						for _, word := range words {
							if hasPrefixFold(lineCode2[start:], word) {
								matchLen = len(word)
								break
							}
						}
						res.HighlightedRanges = append(res.HighlightedRanges, [3]int{
							len(res.LineCodes),
							start,
							start + matchLen,
						})
						res.LineCodes = append(res.LineCodes, lineCode2)
						continue
					}
					res.LineCodes = append(res.LineCodes, lineCode)
				}
			}
			return nil
		},
	})
	// git grep exits by cancel (killed), usually it is caused by the limit of results
	if IsErrorExitCode(err, -1) && stderr.Len() == 0 {
		return results, nil
	}
	// git grep exits with 1 if no results are found
	if IsErrorExitCode(err, 1) && stderr.Len() == 0 {
		return nil, nil
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("unable to run git grep: %w, stderr: %s", err, stderr.String())
	}
	return results, nil
}
