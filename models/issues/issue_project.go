// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	project_model "code.gitea.io/gitea/models/project"
	user_model "code.gitea.io/gitea/models/user"
)

// LoadProject load the project the issue was assigned to
func (issue *Issue) LoadProject() (err error) {
	return issue.loadProject(db.DefaultContext)
}

func (issue *Issue) loadProject(ctx context.Context) (err error) {
	if issue.Project == nil {
		var p project_model.Project
		if _, err = db.GetEngine(ctx).Table("project").
			Join("INNER", "project_issue", "project.id=project_issue.project_id").
			Where("project_issue.issue_id = ?", issue.ID).
			Get(&p); err != nil {
			return err
		}
		issue.Project = &p
	}
	return err
}

// ProjectID return project id if issue was assigned to one
func (issue *Issue) ProjectID() int64 {
	return issue.projectID(db.DefaultContext)
}

func (issue *Issue) projectID(ctx context.Context) int64 {
	var ip project_model.ProjectIssue
	has, err := db.GetEngine(ctx).Where("issue_id=?", issue.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectID
}

// ProjectColumnID returns project column id if issue was assigned to one
func (issue *Issue) ProjectColumnID() int64 {
	return issue.projectColumnID(db.DefaultContext)
}

func (issue *Issue) projectColumnID(ctx context.Context) int64 {
	var ip project_model.ProjectIssue
	has, err := db.GetEngine(ctx).Where("issue_id=?", issue.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectColumnID
}

// LoadIssuesFromColumn loads issues assigned to this column
func LoadIssuesFromColumn(b *project_model.Column) (IssueList, error) {
	issueList := make([]*Issue, 0, 10)

	if b.ID != 0 {
		issues, err := Issues(&IssuesOptions{
			ProjectColumnID: b.ID,
			ProjectID:       b.ProjectID,
		})
		if err != nil {
			return nil, err
		}
		issueList = issues
	}

	if b.Default {
		issues, err := Issues(&IssuesOptions{
			ProjectColumnID: -1, // Issues without ProjectColumnID
			ProjectID:       b.ProjectID,
		})
		if err != nil {
			return nil, err
		}
		issueList = append(issueList, issues...)
	}

	if err := IssueList(issueList).LoadComments(); err != nil {
		return nil, err
	}

	return issueList, nil
}

// LoadIssuesFromColumnList load issues assigned to the columns
func LoadIssuesFromColumnList(bs project_model.Columns) (map[int64]IssueList, error) {
	issuesMap := make(map[int64]IssueList, len(bs))
	for i := range bs {
		il, err := LoadIssuesFromColumn(bs[i])
		if err != nil {
			return nil, err
		}
		issuesMap[bs[i].ID] = il
	}
	return issuesMap, nil
}

// ChangeProjectAssign changes the project associated with an issue
func ChangeProjectAssign(issue *Issue, doer *user_model.User, newProjectID int64) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := addUpdateIssueProject(ctx, issue, doer, newProjectID); err != nil {
		return err
	}

	return committer.Commit()
}

func addUpdateIssueProject(ctx context.Context, issue *Issue, doer *user_model.User, newProjectID int64) error {
	oldProjectID := issue.projectID(ctx)

	// Only check if we add a new project and not remove it.
	if newProjectID > 0 {
		newProject, err := project_model.GetProjectByID(ctx, newProjectID)
		if err != nil {
			return err
		}
		if newProject.RepoID != issue.RepoID {
			return fmt.Errorf("issue's repository is not the same as project's repository")
		}
	}

	if _, err := db.GetEngine(ctx).Where("project_issue.issue_id=?", issue.ID).Delete(&project_model.ProjectIssue{}); err != nil {
		return err
	}

	if err := issue.LoadRepo(ctx); err != nil {
		return err
	}

	if oldProjectID > 0 || newProjectID > 0 {
		if _, err := CreateCommentCtx(ctx, &CreateCommentOptions{
			Type:         CommentTypeProject,
			Doer:         doer,
			Repo:         issue.Repo,
			Issue:        issue,
			OldProjectID: oldProjectID,
			ProjectID:    newProjectID,
		}); err != nil {
			return err
		}
	}

	return db.Insert(ctx, &project_model.ProjectIssue{
		IssueID:   issue.ID,
		ProjectID: newProjectID,
	})
}

// MoveIssueAcrossProjectColumn move a card from one column to another
func MoveIssueAcrossProjectColumns(issue *Issue, column *project_model.Column) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	var pis project_model.ProjectIssue
	has, err := sess.Where("issue_id=?", issue.ID).Get(&pis)
	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("issue has to be added to a project first")
	}

	pis.ProjectColumnID = column.ID
	if _, err := sess.ID(pis.ID).Cols("project_column_id").Update(&pis); err != nil {
		return err
	}

	return committer.Commit()
}
