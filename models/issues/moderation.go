// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues

import (
	"context"

	"forgejo.org/models/moderation"
	"forgejo.org/modules/json"
	"forgejo.org/modules/timeutil"
)

// CommentData represents a trimmed down comment that is used for preserving
// only the fields needed for abusive content reports (mainly string fields).
type CommentData struct {
	PosterID       int64
	IssueID        int64
	Content        string
	ContentVersion int
	CreatedUnix    timeutil.TimeStamp
	UpdatedUnix    timeutil.TimeStamp
}

// newCommentData creates a trimmed down comment to be used just to create a JSON structure
// (keeping only the fields relevant for moderation purposes)
func newCommentData(comment *Comment) CommentData {
	return CommentData{
		PosterID:       comment.PosterID,
		IssueID:        comment.IssueID,
		Content:        comment.Content,
		ContentVersion: comment.ContentVersion,
		CreatedUnix:    comment.CreatedUnix,
		UpdatedUnix:    comment.UpdatedUnix,
	}
}

// IfNeededCreateShadowCopyForComment checks if for the given comment there are any reports of abusive content submitted
// and if found a shadow copy of relevant comment fields will be stored into DB and linked to the above report(s).
// This function should be called before a comment is deleted or updated.
func IfNeededCreateShadowCopyForComment(ctx context.Context, comment *Comment) error {
	if moderation.IsReported(ctx, moderation.ReportedContentTypeComment, comment.ID) {
		commentData := newCommentData(comment)
		content, err := json.Marshal(commentData)
		if err != nil {
			return err
		}
		return moderation.CreateShadowCopyForComment(ctx, comment.ID, string(content))
	}

	return nil
}
