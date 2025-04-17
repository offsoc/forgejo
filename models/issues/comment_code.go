// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"

	"xorm.io/builder"
)

// CodeConversation contains the comment of a given review
type CodeConversation []*Comment

// CodeConversationsAtLine contains the conversations for a given line
type CodeConversationsAtLine map[int64][]CodeConversation

// CodeConversationsAtLineAndTreePath contains the conversations for a given TreePath and line
type CodeConversationsAtLineAndTreePath map[string]CodeConversationsAtLine

func newCodeConversationsAtLineAndTreePath(comments []*Comment) CodeConversationsAtLineAndTreePath {
	tree := make(CodeConversationsAtLineAndTreePath)
	for _, comment := range comments {
		tree.insertComment(comment)
	}
	return tree
}

func (tree CodeConversationsAtLineAndTreePath) insertComment(comment *Comment) {
	// attempt to append comment to existing conversations (i.e. list of comments belonging to the same review)
	for i, conversation := range tree[comment.TreePath][comment.Line] {
		if conversation[0].ReviewID == comment.ReviewID {
			tree[comment.TreePath][comment.Line][i] = append(conversation, comment)
			return
		}
	}

	// no previous conversation was found at this line, create it
	if tree[comment.TreePath] == nil {
		tree[comment.TreePath] = make(map[int64][]CodeConversation)
	}

	tree[comment.TreePath][comment.Line] = append(tree[comment.TreePath][comment.Line], CodeConversation{comment})
}

// FetchCodeConversations will return a 2d-map: ["Path"]["Line"] = List of CodeConversation (one per review) for this line
func FetchCodeConversations(ctx context.Context, issue *Issue, doer *user_model.User, showOutdatedComments bool) (CodeConversationsAtLineAndTreePath, error) {
	opts := FindCommentsOptions{
		Type:    CommentTypeCode,
		IssueID: issue.ID,
	}
	comments, err := findCodeComments(ctx, opts, issue, doer, nil, showOutdatedComments)
	if err != nil {
		return nil, err
	}

	return newCodeConversationsAtLineAndTreePath(comments), nil
}

// CodeComments represents comments on code by using this structure: FILENAME -> LINE (+ == proposed; - == previous) -> COMMENTS
type CodeComments map[string]map[int64][]*Comment

func fetchCodeCommentsByReview(ctx context.Context, issue *Issue, doer *user_model.User, review *Review, showOutdatedComments bool) (CodeComments, error) {
	pathToLineToComment := make(CodeComments)
	if review == nil {
		review = &Review{ID: 0}
	}
	opts := FindCommentsOptions{
		Type:     CommentTypeCode,
		IssueID:  issue.ID,
		ReviewID: review.ID,
	}

	comments, err := findCodeComments(ctx, opts, issue, doer, review, showOutdatedComments)
	if err != nil {
		return nil, err
	}

	for _, comment := range comments {
		if pathToLineToComment[comment.TreePath] == nil {
			pathToLineToComment[comment.TreePath] = make(map[int64][]*Comment)
		}
		pathToLineToComment[comment.TreePath][comment.Line] = append(pathToLineToComment[comment.TreePath][comment.Line], comment)
	}
	return pathToLineToComment, nil
}

func findCodeComments(ctx context.Context, opts FindCommentsOptions, issue *Issue, doer *user_model.User, review *Review, showOutdatedComments bool) (CommentList, error) {
	var comments CommentList
	if review == nil {
		review = &Review{ID: 0}
	}
	conds := opts.ToConds()

	if !showOutdatedComments && review.ID == 0 {
		conds = conds.And(builder.Eq{"invalidated": false})
	}

	e := db.GetEngine(ctx)
	if err := e.Where(conds).
		Asc("comment.created_unix").
		Asc("comment.id").
		Find(&comments); err != nil {
		return nil, err
	}

	if err := issue.LoadRepo(ctx); err != nil {
		return nil, err
	}

	if err := comments.LoadPosters(ctx); err != nil {
		return nil, err
	}

	if err := comments.LoadAttachments(ctx); err != nil {
		return nil, err
	}

	// Find all reviews by ReviewID
	reviews := make(map[int64]*Review)
	ids := make([]int64, 0, len(comments))
	for _, comment := range comments {
		if comment.ReviewID != 0 {
			ids = append(ids, comment.ReviewID)
		}
	}
	if err := e.In("id", ids).Find(&reviews); err != nil {
		return nil, err
	}

	n := 0
	for _, comment := range comments {
		if re, ok := reviews[comment.ReviewID]; ok && re != nil {
			// If the review is pending only the author can see the comments (except if the review is set)
			if review.ID == 0 && re.Type == ReviewTypePending &&
				(doer == nil || doer.ID != re.ReviewerID) {
				continue
			}
			comment.Review = re
		}
		comments[n] = comment
		n++

		if err := comment.LoadResolveDoer(ctx); err != nil {
			return nil, err
		}

		if err := comment.LoadReactions(ctx, issue.Repo); err != nil {
			return nil, err
		}

		var err error
		if comment.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
			Ctx: ctx,
			Links: markup.Links{
				Base: issue.Repo.Link(),
			},
			Metas: issue.Repo.ComposeMetas(ctx),
		}, comment.Content); err != nil {
			return nil, err
		}
	}
	return comments[:n], nil
}

// FetchCodeConversation fetches the code conversation of a given comment (same review, treePath and line number)
func FetchCodeConversation(ctx context.Context, comment *Comment, doer *user_model.User) (CommentList, error) {
	opts := FindCommentsOptions{
		Type:     CommentTypeCode,
		IssueID:  comment.IssueID,
		ReviewID: comment.ReviewID,
		TreePath: comment.TreePath,
		Line:     comment.Line,
	}
	return findCodeComments(ctx, opts, comment.Issue, doer, nil, true)
}
