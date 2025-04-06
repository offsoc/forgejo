// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"forgejo.org/modules/util"
)

// GetRefs returns all references of the repository.
func (repo *Repository) GetRefs() ([]*Reference, error) {
	return repo.GetRefsFiltered("")
}

// ListOccurrences lists all refs of the given refType the given commit appears in sorted by creation date DESC
// refType should only be a literal "branch" or "tag" and nothing else
func (repo *Repository) ListOccurrences(ctx context.Context, refType, commitSHA string) ([]string, error) {
	cmd := NewCommand(ctx)
	switch refType {
	case "branch":
		cmd.AddArguments("branch")
	case "tag":
		cmd.AddArguments("tag")
	default:
		return nil, util.NewInvalidArgumentErrorf(`can only use "branch" or "tag" for refType, but got %q`, refType)
	}
	stdout, _, err := cmd.AddArguments("--no-color", "--sort=-creatordate", "--contains").AddDynamicArguments(commitSHA).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return nil, err
	}

	refs := strings.Split(strings.TrimSpace(stdout), "\n")
	if refType == "branch" {
		return parseBranches(refs), nil
	}
	return parseTags(refs), nil
}

func parseBranches(refs []string) []string {
	results := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.HasPrefix(ref, "* ") { // current branch (main branch)
			results = append(results, ref[len("* "):])
		} else if strings.HasPrefix(ref, "  ") { // all other branches
			results = append(results, ref[len("  "):])
		} else if ref != "" {
			results = append(results, ref)
		}
	}
	return results
}

func parseTags(refs []string) []string {
	results := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref != "" {
			results = append(results, ref)
		}
	}
	return results
}

// ExpandRef expands any partial reference to its full form
func (repo *Repository) ExpandRef(ref string) (string, error) {
	if strings.HasPrefix(ref, "refs/") {
		return ref, nil
	} else if strings.HasPrefix(ref, "tags/") || strings.HasPrefix(ref, "heads/") {
		return "refs/" + ref, nil
	} else if repo.IsTagExist(ref) {
		return TagPrefix + ref, nil
	} else if repo.IsBranchExist(ref) {
		return BranchPrefix + ref, nil
	} else if repo.IsCommitExist(ref) {
		return ref, nil
	}
	return "", fmt.Errorf("could not expand reference '%s'", ref)
}

// GetRefsFiltered returns all references of the repository that matches patterm exactly or starting with.
func (repo *Repository) GetRefsFiltered(pattern string) ([]*Reference, error) {
	stdoutReader, stdoutWriter := io.Pipe()
	defer func() {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
	}()

	go func() {
		stderrBuilder := &strings.Builder{}
		err := NewCommand(repo.Ctx, "for-each-ref").Run(&RunOpts{
			Dir:    repo.Path,
			Stdout: stdoutWriter,
			Stderr: stderrBuilder,
		})
		if err != nil {
			_ = stdoutWriter.CloseWithError(ConcatenateError(err, stderrBuilder.String()))
		} else {
			_ = stdoutWriter.Close()
		}
	}()

	refs := make([]*Reference, 0)
	bufReader := bufio.NewReader(stdoutReader)
	for {
		// The output of for-each-ref is simply a list:
		// <sha> SP <type> TAB <ref> LF
		sha, err := bufReader.ReadString(' ')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		sha = sha[:len(sha)-1]

		typ, err := bufReader.ReadString('\t')
		if err == io.EOF {
			// This should not happen, but we'll tolerate it
			break
		}
		if err != nil {
			return nil, err
		}
		typ = typ[:len(typ)-1]

		refName, err := bufReader.ReadString('\n')
		if err == io.EOF {
			// This should not happen, but we'll tolerate it
			break
		}
		if err != nil {
			return nil, err
		}
		refName = refName[:len(refName)-1]

		// refName cannot be HEAD but can be remotes or stash
		if strings.HasPrefix(refName, RemotePrefix) || refName == "/refs/stash" {
			continue
		}

		if pattern == "" || strings.HasPrefix(refName, pattern) {
			r := &Reference{
				Name:   refName,
				Object: MustIDFromString(sha),
				Type:   typ,
				repo:   repo,
			}
			refs = append(refs, r)
		}
	}

	return refs, nil
}
