// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"code.gitea.io/gitea/modules/git/foreachref"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

// TagPrefix tags prefix path on the repository
const TagPrefix = "refs/tags/"

// CreateTag create one tag in the repository
func (repo *Repository) CreateTag(name, revision string) error {
	_, _, err := NewCommand(repo.Ctx, "tag").AddDashesAndList(name, revision).RunStdString(&RunOpts{Dir: repo.Path})
	return err
}

// CreateAnnotatedTag create one annotated tag in the repository
func (repo *Repository) CreateAnnotatedTag(name, message, revision string) error {
	_, _, err := NewCommand(repo.Ctx, "tag", "-a", "-m").AddDynamicArguments(message).AddDashesAndList(name, revision).RunStdString(&RunOpts{Dir: repo.Path})
	return err
}

// GetTagNameBySHA returns the name of a tag from its tag object SHA or commit SHA
func (repo *Repository) GetTagNameBySHA(sha string) (string, error) {
	if len(sha) < 5 {
		return "", fmt.Errorf("SHA is too short: %s", sha)
	}

	stdout, _, err := NewCommand(repo.Ctx, "show-ref", "--tags", "-d").RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return "", err
	}

	tagRefs := strings.Split(stdout, "\n")
	for _, tagRef := range tagRefs {
		if len(strings.TrimSpace(tagRef)) > 0 {
			fields := strings.Fields(tagRef)
			if strings.HasPrefix(fields[0], sha) && strings.HasPrefix(fields[1], TagPrefix) {
				name := fields[1][len(TagPrefix):]
				// annotated tags show up twice, we should only return if is not the ^{} ref
				if !strings.HasSuffix(name, "^{}") {
					return name, nil
				}
			}
		}
	}
	return "", ErrNotExist{ID: sha}
}

// GetTagID returns the object ID for a tag (annotated tags have both an object SHA AND a commit SHA)
func (repo *Repository) GetTagID(name string) (string, error) {
	stdout, _, err := NewCommand(repo.Ctx, "show-ref", "--tags").AddDashesAndList(name).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return "", err
	}
	// Make sure exact match is used: "v1" != "release/v1"
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "refs/tags/"+name {
			return fields[0], nil
		}
	}
	return "", ErrNotExist{ID: name}
}

// GetTag returns a Git tag by given name.
func (repo *Repository) GetTag(name string) (*Tag, error) {
	idStr, err := repo.GetTagID(name)
	if err != nil {
		return nil, err
	}

	id, err := NewIDFromString(idStr)
	if err != nil {
		return nil, err
	}

	tag, err := repo.getTag(id, name)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// GetTagWithID returns a Git tag by given name and ID
func (repo *Repository) GetTagWithID(idStr, name string) (*Tag, error) {
	id, err := NewIDFromString(idStr)
	if err != nil {
		return nil, err
	}

	tag, err := repo.getTag(id, name)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// GetTagInfos returns all tag infos of the repository.
func (repo *Repository) GetTagInfos(page, pageSize int) ([]*Tag, int, error) {
	// Generally, refname:short should be equal to refname:lstrip=2 except core.warnAmbiguousRefs is used to select the strict abbreviation mode.
	// https://git-scm.com/docs/git-for-each-ref#Documentation/git-for-each-ref.txt-refname
	forEachRefFmt := foreachref.NewFormat("objecttype", "refname:lstrip=2", "object", "objectname", "creator", "contents", "contents:signature")

	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	defer stdoutWriter.Close()
	stderr := strings.Builder{}
	rc := &RunOpts{Dir: repo.Path, Stdout: stdoutWriter, Stderr: &stderr}

	go func() {
		err := NewCommand(repo.Ctx, "for-each-ref").
			AddOptionFormat("--format=%s", forEachRefFmt.Flag()).
			AddArguments("--sort", "-*creatordate", "refs/tags").Run(rc)
		if err != nil {
			_ = stdoutWriter.CloseWithError(ConcatenateError(err, stderr.String()))
		} else {
			_ = stdoutWriter.Close()
		}
	}()

	var tags []*Tag
	parser := forEachRefFmt.Parser(stdoutReader)
	for {
		ref := parser.Next()
		if ref == nil {
			break
		}

		tag, err := parseTagRef(ref)
		if err != nil {
			return nil, 0, fmt.Errorf("GetTagInfos: parse tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := parser.Err(); err != nil {
		return nil, 0, fmt.Errorf("GetTagInfos: parse output: %w", err)
	}

	slices.SortFunc(tags, func(b, a *Tag) int {
		return a.Tagger.When.Compare(b.Tagger.When)
	})
	tagsTotal := len(tags)
	if page != 0 {
		tags = util.PaginateSlice(tags, page, pageSize).([]*Tag)
	}

	return tags, tagsTotal, nil
}

// parseTagRef parses a tag from a 'git for-each-ref'-produced reference.
func parseTagRef(ref map[string]string) (tag *Tag, err error) {
	tag = &Tag{
		Type: ref["objecttype"],
		Name: ref["refname:lstrip=2"],
	}

	tag.ID, err = NewIDFromString(ref["objectname"])
	if err != nil {
		return nil, fmt.Errorf("parse objectname '%s': %w", ref["objectname"], err)
	}

	if tag.Type == "commit" {
		// lightweight tag
		tag.Object = tag.ID
	} else {
		// annotated tag
		tag.Object, err = NewIDFromString(ref["object"])
		if err != nil {
			return nil, fmt.Errorf("parse object '%s': %w", ref["object"], err)
		}
	}

	tag.Tagger = parseSignatureFromCommitLine(ref["creator"])
	tag.Message = ref["contents"]
	// strip the signature if present in contents field
	pgpStart := strings.Index(tag.Message, beginpgp)
	if pgpStart >= 0 {
		tag.Message = tag.Message[0:pgpStart]
	} else {
		sshStart := strings.Index(tag.Message, beginssh)
		if sshStart >= 0 {
			tag.Message = tag.Message[0:sshStart]
		}
	}

	// annotated tag with signature
	if tag.Type == "tag" && ref["contents:signature"] != "" {
		payload := fmt.Sprintf("object %s\ntype commit\ntag %s\ntagger %s\n\n%s\n",
			tag.Object, tag.Name, ref["creator"], strings.TrimSpace(tag.Message))
		tag.Signature = &ObjectSignature{
			Signature: ref["contents:signature"],
			Payload:   payload,
		}
	}

	return tag, nil
}

// GetAnnotatedTag returns a Git tag by its SHA, must be an annotated tag
func (repo *Repository) GetAnnotatedTag(sha string) (*Tag, error) {
	id, err := NewIDFromString(sha)
	if err != nil {
		return nil, err
	}

	// Tag type must be "tag" (annotated) and not a "commit" (lightweight) tag
	if tagType, err := repo.GetTagType(id); err != nil {
		return nil, err
	} else if ObjectType(tagType) != ObjectTag {
		// not an annotated tag
		return nil, ErrNotExist{ID: id.String()}
	}

	// Get tag name
	name, err := repo.GetTagNameBySHA(id.String())
	if err != nil {
		return nil, err
	}

	tag, err := repo.getTag(id, name)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// IsTagExist returns true if given tag exists in the repository.
func (repo *Repository) IsTagExist(name string) bool {
	if repo == nil || name == "" {
		return false
	}

	return repo.IsReferenceExist(TagPrefix + name)
}

// GetTags returns all tags of the repository.
// returning at most limit tags, or all if limit is 0.
func (repo *Repository) GetTags(skip, limit int) (tags []string, err error) {
	tags, _, err = callShowRef(repo.Ctx, repo.Path, TagPrefix, TrustedCmdArgs{TagPrefix, "--sort=-taggerdate"}, skip, limit)
	return tags, err
}

// GetTagType gets the type of the tag, either commit (simple) or tag (annotated)
func (repo *Repository) GetTagType(id ObjectID) (string, error) {
	wr, rd, cancel, err := repo.CatFileBatchCheck(repo.Ctx)
	if err != nil {
		return "", err
	}
	defer cancel()
	_, err = wr.Write([]byte(id.String() + "\n"))
	if err != nil {
		return "", err
	}
	_, typ, _, err := ReadBatchLine(rd)
	if IsErrNotExist(err) {
		return "", ErrNotExist{ID: id.String()}
	}
	return typ, nil
}

func (repo *Repository) getTag(tagID ObjectID, name string) (*Tag, error) {
	t, ok := repo.tagCache.Get(tagID.String())
	if ok {
		log.Debug("Hit cache: %s", tagID)
		tagClone := *t.(*Tag)
		tagClone.Name = name // This is necessary because lightweight tags may have same id
		return &tagClone, nil
	}

	tp, err := repo.GetTagType(tagID)
	if err != nil {
		return nil, err
	}

	// Get the commit ID and tag ID (may be different for annotated tag) for the returned tag object
	commitIDStr, err := repo.GetTagCommitID(name)
	if err != nil {
		// every tag should have a commit ID so return all errors
		return nil, err
	}
	commitID, err := NewIDFromString(commitIDStr)
	if err != nil {
		return nil, err
	}

	// If type is "commit, the tag is a lightweight tag
	if ObjectType(tp) == ObjectCommit {
		commit, err := repo.GetCommit(commitIDStr)
		if err != nil {
			return nil, err
		}
		tag := &Tag{
			Name:    name,
			ID:      tagID,
			Object:  commitID,
			Type:    tp,
			Tagger:  commit.Committer,
			Message: commit.Message(),
		}

		repo.tagCache.Set(tagID.String(), tag)
		return tag, nil
	}

	// The tag is an annotated tag with a message.
	wr, rd, cancel, err := repo.CatFileBatch(repo.Ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	if _, err := wr.Write([]byte(tagID.String() + "\n")); err != nil {
		return nil, err
	}
	_, typ, size, err := ReadBatchLine(rd)
	if err != nil {
		if errors.Is(err, io.EOF) || IsErrNotExist(err) {
			return nil, ErrNotExist{ID: tagID.String()}
		}
		return nil, err
	}
	if typ != "tag" {
		if err := DiscardFull(rd, size+1); err != nil {
			return nil, err
		}
		return nil, ErrNotExist{ID: tagID.String()}
	}

	// then we need to parse the tag
	// and load the commit
	data, err := io.ReadAll(io.LimitReader(rd, size))
	if err != nil {
		return nil, err
	}
	_, err = rd.Discard(1)
	if err != nil {
		return nil, err
	}

	tag, err := parseTagData(tagID.Type(), data)
	if err != nil {
		return nil, err
	}

	tag.Name = name
	tag.ID = tagID
	tag.Type = tp

	repo.tagCache.Set(tagID.String(), tag)
	return tag, nil
}
