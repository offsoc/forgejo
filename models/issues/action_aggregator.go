// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues

import (
	"slices"

	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
)

type ActionAggregator struct {
	StartUnix int64
	AggAge    int64
	PosterID  int64
	StartInd  int
	EndInd    int

	PrevClosed bool
	IsClosed   bool

	AddedLabels   []*Label
	RemovedLabels []*Label

	AddedRequestReview   []RequestReviewTarget
	RemovedRequestReview []RequestReviewTarget
}

// Get the time threshold for aggregation of multiple actions together
func (agg *ActionAggregator) timeThreshold() int64 {
	if agg.AggAge > (60 * 60 * 24 * 30) { // Age > 1 month, aggregate by day
		return 60 * 60 * 24
	} else if agg.AggAge > (60 * 60 * 24) { // Age > 1 day, aggregate by hour
		return 60 * 60
	} else if agg.AggAge > (60 * 60) { // Age > 1 hour, aggregate by 10 mins
		return 60 * 10
	}
	// Else, aggregate by minute
	return 60
}

// TODO    Aggregate also
//   - Dependency added / removed
//   - Added / Removed due date
//   - Milestone Added / Removed
func (agg *ActionAggregator) aggregateAction(c *Comment, index int) {
	if agg.StartInd == -1 {
		agg.StartInd = index
	}
	agg.EndInd = index

	if c.Type == CommentTypeClose {
		agg.IsClosed = true
	} else if c.Type == CommentTypeReopen {
		agg.IsClosed = false
	} else if c.Type == CommentTypeReviewRequest {
		if c.AssigneeID > 0 {
			req := RequestReviewTarget{User: c.Assignee}
			if c.RemovedAssignee {
				agg.delReviewRequest(req)
			} else {
				agg.addReviewRequest(req)
			}
		} else if c.AssigneeTeamID > 0 {
			req := RequestReviewTarget{Team: c.AssigneeTeam}
			if c.RemovedAssignee {
				agg.delReviewRequest(req)
			} else {
				agg.addReviewRequest(req)
			}
		}

		for _, r := range c.RemovedRequestReview {
			agg.delReviewRequest(r)
		}

		for _, r := range c.AddedRequestReview {
			agg.addReviewRequest(r)
		}
	} else if c.Type == CommentTypeLabel {
		if c.Content == "1" {
			agg.addLabel(c.Label)
		} else {
			agg.delLabel(c.Label)
		}
	} else if c.Type == CommentTypeAggregator {
		agg.Merge(c.Aggregator)
	}
}

// Merge a past CommentAggregator with the next one in the issue comments list
func (agg *ActionAggregator) Merge(next *ActionAggregator) {
	agg.IsClosed = next.IsClosed

	for _, l := range next.AddedLabels {
		agg.addLabel(l)
	}

	for _, l := range next.RemovedLabels {
		agg.delLabel(l)
	}

	for _, r := range next.AddedRequestReview {
		agg.addReviewRequest(r)
	}

	for _, r := range next.RemovedRequestReview {
		agg.delReviewRequest(r)
	}
}

// Check if a comment can be aggregated or not depending on its type
func (agg *ActionAggregator) IsAggregated(t *CommentType) bool {
	switch *t {
	case CommentTypeAggregator, CommentTypeClose, CommentTypeReopen, CommentTypeLabel, CommentTypeReviewRequest:
		{
			return true
		}
	default:
		{
			return false
		}
	}
}

// Add a label to the aggregated list
func (agg *ActionAggregator) addLabel(lbl *Label) {
	for l, agglbl := range agg.RemovedLabels {
		if agglbl.ID == lbl.ID {
			agg.RemovedLabels = slices.Delete(agg.RemovedLabels, l, l+1)
			return
		}
	}

	if !slices.ContainsFunc(agg.AddedLabels, func(l *Label) bool { return l.ID == lbl.ID }) {
		agg.AddedLabels = append(agg.AddedLabels, lbl)
	}
}

// Remove a label from the aggregated list
func (agg *ActionAggregator) delLabel(lbl *Label) {
	for l, agglbl := range agg.AddedLabels {
		if agglbl.ID == lbl.ID {
			agg.AddedLabels = slices.Delete(agg.AddedLabels, l, l+1)
			return
		}
	}

	if !slices.ContainsFunc(agg.RemovedLabels, func(l *Label) bool { return l.ID == lbl.ID }) {
		agg.RemovedLabels = append(agg.RemovedLabels, lbl)
	}
}

// Add a review request to the aggregated list
func (agg *ActionAggregator) addReviewRequest(req RequestReviewTarget) {
	reqid := req.ID()
	reqty := req.Type()
	for r, aggreq := range agg.RemovedRequestReview {
		if (aggreq.ID() == reqid) && (aggreq.Type() == reqty) {
			agg.RemovedRequestReview = slices.Delete(agg.RemovedRequestReview, r, r+1)
			return
		}
	}

	if !slices.ContainsFunc(agg.AddedRequestReview, func(r RequestReviewTarget) bool { return (r.ID() == reqid) && (r.Type() == reqty) }) {
		agg.AddedRequestReview = append(agg.AddedRequestReview, req)
	}
}

// Delete a review request from the aggregated list
func (agg *ActionAggregator) delReviewRequest(req RequestReviewTarget) {
	reqid := req.ID()
	reqty := req.Type()
	for r, aggreq := range agg.AddedRequestReview {
		if (aggreq.ID() == reqid) && (aggreq.Type() == reqty) {
			agg.AddedRequestReview = slices.Delete(agg.AddedRequestReview, r, r+1)
			return
		}
	}

	if !slices.ContainsFunc(agg.RemovedRequestReview, func(r RequestReviewTarget) bool { return (r.ID() == reqid) && (r.Type() == reqty) }) {
		agg.RemovedRequestReview = append(agg.RemovedRequestReview, req)
	}
}

// Check if anything has changed with this aggregated list of comments
func (agg *ActionAggregator) Changed() bool {
	return (agg.IsClosed != agg.PrevClosed) ||
		(len(agg.AddedLabels) > 0) ||
		(len(agg.RemovedLabels) > 0) ||
		(len(agg.AddedRequestReview) > 0) ||
		(len(agg.RemovedRequestReview) > 0)
}

func (agg *ActionAggregator) OnlyLabelsChanged() bool {
	return ((len(agg.AddedLabels) > 0) || (len(agg.RemovedLabels) > 0)) &&
		(len(agg.AddedRequestReview) == 0) && (len(agg.RemovedRequestReview) == 0) &&
		(agg.PrevClosed == agg.IsClosed)
}

func (agg *ActionAggregator) OnlyRequestReview() bool {
	return ((len(agg.AddedRequestReview) > 0) || (len(agg.RemovedRequestReview) > 0)) &&
		(len(agg.AddedLabels) == 0) && (len(agg.RemovedLabels) == 0) &&
		(agg.PrevClosed == agg.IsClosed)
}

func (agg *ActionAggregator) OnlyClosedReopened() bool {
	return (agg.IsClosed != agg.PrevClosed) &&
		(len(agg.AddedLabels) == 0) && (len(agg.RemovedLabels) == 0) &&
		(len(agg.AddedRequestReview) == 0) && (len(agg.RemovedRequestReview) == 0)
}

// Reset the aggregator to start a new aggregating context
func (agg *ActionAggregator) Reset(cur *Comment, now int64) {
	agg.StartUnix = int64(cur.CreatedUnix)
	agg.AggAge = now - agg.StartUnix
	agg.PosterID = cur.PosterID

	agg.PrevClosed = agg.IsClosed

	agg.StartInd = -1
	agg.EndInd = -1
	agg.AddedLabels = []*Label{}
	agg.RemovedLabels = []*Label{}
	agg.AddedRequestReview = []RequestReviewTarget{}
	agg.RemovedRequestReview = []RequestReviewTarget{}
}

// Function that replaces all the comments aggregated with a single one
// Its CommentType depend on whether multiple type of comments are been aggregated or not
// If nothing has changed, we remove all the comments that get nullified
//
// The function returns how many comments has been removed, in order for the "for" loop
// of the main algorithm to change its index
func (agg *ActionAggregator) createAggregatedComment(issue *Issue, final bool) int {
	// If the aggregation of comments make the whole thing null, erase all the comments
	if !agg.Changed() {
		if final {
			issue.Comments = issue.Comments[:agg.StartInd]
		} else {
			issue.Comments = slices.Replace(issue.Comments, agg.StartInd, agg.EndInd+1)
		}
		return (agg.EndInd - agg.StartInd) + 1
	}

	newAgg := *agg // Trigger a memory allocation, get a COPY of the aggregator

	// Keep the same author, time, etc... But reset the parts we may want to use
	comment := issue.Comments[agg.StartInd]
	comment.Content = ""
	comment.Label = nil
	comment.Aggregator = nil
	comment.Assignee = nil
	comment.AssigneeID = 0
	comment.AssigneeTeam = nil
	comment.AssigneeTeamID = 0
	comment.RemovedAssignee = false
	comment.AddedLabels = nil
	comment.RemovedLabels = nil

	// In case there's only a single change, create a comment of this type
	// instead of an aggregator
	if agg.OnlyLabelsChanged() {
		comment.Type = CommentTypeLabel
	} else if agg.OnlyClosedReopened() {
		if agg.IsClosed {
			comment.Type = CommentTypeClose
		} else {
			comment.Type = CommentTypeReopen
		}
	} else if agg.OnlyRequestReview() {
		comment.Type = CommentTypeReviewRequest
	} else {
		comment.Type = CommentTypeAggregator
		comment.Aggregator = &newAgg
	}

	if len(newAgg.AddedLabels) > 0 {
		comment.AddedLabels = newAgg.AddedLabels
	}

	if len(newAgg.RemovedLabels) > 0 {
		comment.RemovedLabels = newAgg.RemovedLabels
	}

	if len(newAgg.AddedRequestReview) > 0 {
		comment.AddedRequestReview = newAgg.AddedRequestReview
	}

	if len(newAgg.RemovedRequestReview) > 0 {
		comment.RemovedRequestReview = newAgg.RemovedRequestReview
	}

	if final {
		issue.Comments = append(issue.Comments[:agg.StartInd], comment)
	} else {
		issue.Comments = slices.Replace(issue.Comments, agg.StartInd, agg.EndInd+1, comment)
	}
	return agg.EndInd - agg.StartInd
}

// combineCommentsHistory combines nearby elements in the history as one
func CombineCommentsHistory(issue *Issue, now int64) {
	if len(issue.Comments) < 1 {
		return
	}

	// Initialise a new empty aggregator, ready to combine comments
	var agg ActionAggregator
	agg.Reset(issue.Comments[0], now)

	for i := 0; i < len(issue.Comments); i++ {
		cur := issue.Comments[i]
		// If the comment we encounter is not accepted inside an aggregator
		if !agg.IsAggregated(&cur.Type) {
			// If we aggregated some data, create the resulting comment for it
			if agg.StartInd != -1 {
				i -= agg.createAggregatedComment(issue, false)
			}

			agg.StartInd = -1
			if i+1 < len(issue.Comments) {
				agg.Reset(issue.Comments[i+1], now)
			}

			// Do not need to continue the aggregation loop, skip to next comment
			continue
		}

		// If the comment we encounter cannot be aggregated with the current aggregator,
		// we create a new empty aggregator
		threshold := agg.timeThreshold()
		if ((int64(cur.CreatedUnix) - agg.StartUnix) > threshold) || (cur.PosterID != agg.PosterID) {
			// First, create the aggregated comment if there's data in it
			if agg.StartInd != -1 {
				i -= agg.createAggregatedComment(issue, false)
			}
			agg.Reset(cur, now)
		}

		agg.aggregateAction(cur, i)
	}

	// Create the aggregated comment if there's data in it
	if agg.StartInd != -1 {
		agg.createAggregatedComment(issue, true)
	}
}

type RequestReviewTarget struct {
	User *user_model.User
	Team *organization.Team
}

func (t *RequestReviewTarget) ID() int64 {
	if t.User != nil {
		return t.User.ID
	}
	return t.Team.ID
}

func (t *RequestReviewTarget) Name() string {
	if t.User != nil {
		return t.User.GetDisplayName()
	}
	return t.Team.Name
}

func (t *RequestReviewTarget) Type() string {
	if t.User != nil {
		return "user"
	}
	return "team"
}
