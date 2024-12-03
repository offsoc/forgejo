// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	issues_model "code.gitea.io/gitea/models/issues"
	org_model "code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestCombineLabelComments(t *testing.T) {
	kases := []struct {
		name           string
		beforeCombined []*issues_model.Comment
		afterCombined  []*issues_model.Comment
	}{
		{
			name: "kase 1",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    1,
					Content:     "1",
					CreatedUnix: 0,
					AddedLabels: []*issues_model.Label{},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
		},
		{
			name: "kase 2",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 70,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    1,
					Content:     "1",
					CreatedUnix: 0,
					AddedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    1,
					Content:     "",
					CreatedUnix: 70,
					RemovedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
		},
		{
			name: "kase 3",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 2,
					Content:  "",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    1,
					Content:     "1",
					CreatedUnix: 0,
					AddedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    2,
					Content:     "",
					CreatedUnix: 0,
					RemovedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
		},
		{
			name: "kase 4",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/backport",
					},
					CreatedUnix: 10,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeLabel,
					PosterID:    1,
					Content:     "1",
					CreatedUnix: 10,
					AddedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
						{
							Name: "kind/backport",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
				},
			},
		},
		{
			name: "kase 5",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    2,
					Content:     "testtest",
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					AddedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    2,
					Content:     "testtest",
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "",
					RemovedLabels: []*issues_model.Label{
						{
							Name: "kind/bug",
						},
					},
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
			},
		},
		{
			name: "kase 6",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "reviewed/confirmed",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/feature",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeLabel,
					PosterID: 1,
					Content:  "1",
					Label: &issues_model.Label{
						Name: "kind/bug",
					},
					AddedLabels: []*issues_model.Label{
						{
							Name: "reviewed/confirmed",
						},
						{
							Name: "kind/feature",
						},
					},
					CreatedUnix: 0,
				},
			},
		},
	}

	for _, kase := range kases {
		t.Run(kase.name, func(t *testing.T) {
			issue := issues_model.Issue{
				Comments: kase.beforeCombined,
			}
			combineLabelComments(&issue)
			assert.EqualValues(t, kase.afterCombined, issue.Comments)
		})
	}
}

func TestCombineReviewRequests(t *testing.T) {
	testCases := []struct {
		name           string
		beforeCombined []*issues_model.Comment
		afterCombined  []*issues_model.Comment
	}{
		{
			name: "case 1",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:               issues_model.CommentTypeReviewRequest,
					PosterID:           1,
					CreatedUnix:        0,
					AddedRequestReview: []issues_model.RequestReviewTarget{},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
			},
		},
		{
			name: "case 2",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   2,
						Name: "Ghost 2",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					AddedRequestReview: []issues_model.RequestReviewTarget{
						&RequestReviewTarget{
							user: &user_model.User{
								ID:   1,
								Name: "Ghost",
							},
						},
						&RequestReviewTarget{
							user: &user_model.User{
								ID:   2,
								Name: "Ghost 2",
							},
						},
					},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
			},
		},
		{
			name: "case 3",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					AddedRequestReview: []issues_model.RequestReviewTarget{
						&RequestReviewTarget{
							user: &user_model.User{
								ID:   1,
								Name: "Ghost",
							},
						},
					},
					RemovedRequestReview: []issues_model.RequestReviewTarget{
						&RequestReviewTarget{
							team: &org_model.Team{
								ID:   1,
								Name: "Team 1",
							},
						},
					},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
			},
		},
		{
			name: "case 4",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					AddedRequestReview: []issues_model.RequestReviewTarget{
						&RequestReviewTarget{
							user: &user_model.User{
								ID:   1,
								Name: "Ghost",
							},
						},
					},
					RemovedRequestReview: []issues_model.RequestReviewTarget{},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
			},
		},
		{
			name: "case 5",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:                 issues_model.CommentTypeReviewRequest,
					PosterID:             1,
					CreatedUnix:          0,
					AddedRequestReview:   []issues_model.RequestReviewTarget{},
					RemovedRequestReview: []issues_model.RequestReviewTarget{},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
			},
		},
		{
			name: "case 6",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 0,
				},
				{
					Type:            issues_model.CommentTypeReviewRequest,
					PosterID:        1,
					RemovedAssignee: true,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					RemovedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						team: &org_model.Team{
							ID:   1,
							Name: "Team 1",
						},
					}},
					AddedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						user: &user_model.User{
							ID:   1,
							Name: "Ghost",
						},
					}},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
				{
					Type:        issues_model.CommentTypeComment,
					PosterID:    1,
					Content:     "test",
					CreatedUnix: 0,
				},
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					AddedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						team: &org_model.Team{
							ID:   1,
							Name: "Team 1",
						},
					}},
					RemovedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						user: &user_model.User{
							ID:   1,
							Name: "Ghost",
						},
					}},
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
				},
			},
		},
		{
			name: "case 7",
			beforeCombined: []*issues_model.Comment{
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
					CreatedUnix: 0,
				},
				{
					Type:     issues_model.CommentTypeReviewRequest,
					PosterID: 1,
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
					CreatedUnix: 61,
				},
			},
			afterCombined: []*issues_model.Comment{
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					AddedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						user: &user_model.User{
							ID:   1,
							Name: "Ghost",
						},
					}},
					Assignee: &user_model.User{
						ID:   1,
						Name: "Ghost",
					},
				},
				{
					Type:        issues_model.CommentTypeReviewRequest,
					PosterID:    1,
					CreatedUnix: 0,
					RemovedRequestReview: []issues_model.RequestReviewTarget{&RequestReviewTarget{
						team: &org_model.Team{
							ID:   1,
							Name: "Team 1",
						},
					}},
					AssigneeTeam: &org_model.Team{
						ID:   1,
						Name: "Team 1",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			issue := issues_model.Issue{
				Comments: testCase.beforeCombined,
			}
			combineRequestReviewComments(&issue)
			assert.EqualValues(t, testCase.afterCombined[0], issue.Comments[0])
		})
	}
}
