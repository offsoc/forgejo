// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import "fmt"

type (
	LimitSubject  int
	LimitSubjects []LimitSubject
)

const (
	LimitSubjectNone LimitSubject = iota
	LimitSubjectSizeAll
	LimitSubjectSizeReposAll
	LimitSubjectSizeReposPublic
	LimitSubjectSizeReposPrivate
	LimitSubjectSizeGitAll
	LimitSubjectSizeGitLFS
	LimitSubjectSizeAssetsAll
	LimitSubjectSizeAssetsAttachmentsAll
	LimitSubjectSizeAssetsAttachmentsIssues
	LimitSubjectSizeAssetsAttachmentsReleases
	LimitSubjectSizeAssetsArtifacts
	LimitSubjectSizeAssetsPackagesAll
	LimitSubjectSizeWiki

	LimitSubjectFirst = LimitSubjectSizeAll
	LimitSubjectLast  = LimitSubjectSizeWiki
)

var limitSubjectRepr = map[string]LimitSubject{
	"none":                             LimitSubjectNone,
	"size:all":                         LimitSubjectSizeAll,
	"size:repos:all":                   LimitSubjectSizeReposAll,
	"size:repos:public":                LimitSubjectSizeReposPublic,
	"size:repos:private":               LimitSubjectSizeReposPrivate,
	"size:git:all":                     LimitSubjectSizeGitAll,
	"size:git:lfs":                     LimitSubjectSizeGitLFS,
	"size:assets:all":                  LimitSubjectSizeAssetsAll,
	"size:assets:attachments:all":      LimitSubjectSizeAssetsAttachmentsAll,
	"size:assets:attachments:issues":   LimitSubjectSizeAssetsAttachmentsIssues,
	"size:assets:attachments:releases": LimitSubjectSizeAssetsAttachmentsReleases,
	"size:assets:artifacts":            LimitSubjectSizeAssetsArtifacts,
	"size:assets:packages:all":         LimitSubjectSizeAssetsPackagesAll,
	"size:assets:wiki":                 LimitSubjectSizeWiki,
}

func (subject LimitSubject) String() string {
	for repr, limit := range limitSubjectRepr {
		if limit == subject {
			return repr
		}
	}
	return "<unknown>"
}

func (subjects LimitSubjects) GoString() string {
	return fmt.Sprintf("%T{%+v}", subjects, subjects)
}

func ParseLimitSubject(repr string) (LimitSubject, error) {
	result, has := limitSubjectRepr[repr]
	if !has {
		return LimitSubjectNone, ErrParseLimitSubjectUnrecognized{Subject: repr}
	}
	return result, nil
}
