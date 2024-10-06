// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/timeutil"
	issue_service "code.gitea.io/gitea/services/issue"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &issue{}

type issue struct {
	common

	forgejoIssue *issues_model.Issue
}

func (o *issue) SetNative(issue any) {
	o.forgejoIssue = issue.(*issues_model.Issue)
}

func (o *issue) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoIssue.Index)
}

func (o *issue) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *issue) ToFormat() f3.Interface {
	if o.forgejoIssue == nil {
		return o.NewFormat()
	}

	var milestone *f3.Reference
	if o.forgejoIssue.Milestone != nil {
		milestone = f3_tree.NewIssueMilestoneReference(o.forgejoIssue.Milestone.ID)
	}

	assignees := make([]*f3.Reference, 0, len(o.forgejoIssue.Assignees))
	for _, assignee := range o.forgejoIssue.Assignees {
		assignees = append(assignees, f3_tree.NewUserReference(assignee.ID))
	}

	labels := make([]*f3.Reference, 0, len(o.forgejoIssue.Labels))
	for _, label := range o.forgejoIssue.Labels {
		labels = append(labels, f3_tree.NewIssueLabelReference(label.ID))
	}

	return &f3.Issue{
		Title:     o.forgejoIssue.Title,
		Common:    f3.NewCommon(o.GetNativeID()),
		PosterID:  f3_tree.NewUserReference(o.forgejoIssue.Poster.ID),
		Assignees: assignees,
		Labels:    labels,
		Content:   o.forgejoIssue.Content,
		Milestone: milestone,
		State:     string(o.forgejoIssue.State()),
		Created:   o.forgejoIssue.CreatedUnix.AsTime(),
		Updated:   o.forgejoIssue.UpdatedUnix.AsTime(),
		Closed:    o.forgejoIssue.ClosedUnix.AsTimePtr(),
		IsLocked:  o.forgejoIssue.IsLocked,
	}
}

func (o *issue) FromFormat(content f3.Interface) {
	issue := content.(*f3.Issue)
	var milestone *issues_model.Milestone
	if issue.Milestone != nil {
		milestone = &issues_model.Milestone{
			ID: issue.Milestone.GetIDAsInt(),
		}
	}
	o.forgejoIssue = &issues_model.Issue{
		Title:    issue.Title,
		Index:    f3_util.ParseInt(issue.GetID()),
		PosterID: issue.PosterID.GetIDAsInt(),
		Poster: &user_model.User{
			ID: issue.PosterID.GetIDAsInt(),
		},
		Content:     issue.Content,
		Milestone:   milestone,
		IsClosed:    issue.State == f3.IssueStateClosed,
		CreatedUnix: timeutil.TimeStamp(issue.Created.Unix()),
		UpdatedUnix: timeutil.TimeStamp(issue.Updated.Unix()),
		IsLocked:    issue.IsLocked,
	}

	assignees := make([]*user_model.User, 0, len(issue.Assignees))
	for _, assignee := range issue.Assignees {
		assignees = append(assignees, &user_model.User{ID: assignee.GetIDAsInt()})
	}
	o.forgejoIssue.Assignees = assignees

	labels := make([]*issues_model.Label, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labels = append(labels, &issues_model.Label{ID: label.GetIDAsInt()})
	}
	o.forgejoIssue.Labels = labels

	if issue.Closed != nil {
		o.forgejoIssue.ClosedUnix = timeutil.TimeStamp(issue.Closed.Unix())
	}
}

func (o *issue) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	id := node.GetID().Int64()

	issue, err := issues_model.GetIssueByIndex(ctx, project, id)
	if issues_model.IsErrIssueNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("issue %v %w", id, err))
	}
	if err := issue.LoadAttributes(ctx); err != nil {
		panic(err)
	}

	o.forgejoIssue = issue
	return true
}

func (o *issue) Patch(ctx context.Context) {
	node := o.GetNode()
	project := f3_tree.GetProjectID(o.GetNode())
	id := node.GetID().Int64()
	o.Trace("repo_id = %d, index = %d", project, id)
	if _, err := db.GetEngine(ctx).Where("`repo_id` = ? AND `index` = ?", project, id).Cols("name", "content", "is_closed").Update(o.forgejoIssue); err != nil {
		panic(fmt.Errorf("%v %v", o.forgejoIssue, err))
	}
}

func (o *issue) Put(ctx context.Context) generic.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	o.forgejoIssue.RepoID = f3_tree.GetProjectID(o.GetNode())
	makeLabels := func(issueID int64) []issues_model.IssueLabel {
		labels := make([]issues_model.IssueLabel, 0, len(o.forgejoIssue.Labels))
		for _, label := range o.forgejoIssue.Labels {
			o.Trace("%d with label %d", issueID, label.ID)
			labels = append(labels, issues_model.IssueLabel{
				IssueID: issueID,
				LabelID: label.ID,
			})
		}
		return labels
	}

	idx, err := db.GetNextResourceIndex(ctx, "issue_index", o.forgejoIssue.RepoID)
	if err != nil {
		panic(fmt.Errorf("generate issue index failed: %w", err))
	}
	o.forgejoIssue.Index = idx

	sess := db.GetEngine(ctx)

	if _, err = sess.NoAutoTime().Insert(o.forgejoIssue); err != nil {
		panic(err)
	}

	labels := makeLabels(o.forgejoIssue.ID)
	if len(labels) > 0 {
		if _, err := sess.Insert(labels); err != nil {
			panic(err)
		}
	}

	makeAssignees := func(issueID int64) []issues_model.IssueAssignees {
		assignees := make([]issues_model.IssueAssignees, 0, len(o.forgejoIssue.Assignees))
		for _, assignee := range o.forgejoIssue.Assignees {
			o.Trace("%d with assignee %d", issueID, assignee.ID)
			assignees = append(assignees, issues_model.IssueAssignees{
				IssueID:    issueID,
				AssigneeID: assignee.ID,
			})
		}
		return assignees
	}

	assignees := makeAssignees(o.forgejoIssue.ID)
	if len(assignees) > 0 {
		if _, err := sess.Insert(assignees); err != nil {
			panic(err)
		}
	}

	o.Trace("issue created %d/%d", o.forgejoIssue.ID, o.forgejoIssue.Index)
	return generic.NewNodeID(o.forgejoIssue.Index)
}

func (o *issue) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	owner := f3_tree.GetOwnerName(o.GetNode())
	project := f3_tree.GetProjectName(o.GetNode())
	repoPath := repo_model.RepoPath(owner, project)
	gitRepo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		panic(err)
	}
	defer gitRepo.Close()

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	if err := issue_service.DeleteIssue(ctx, doer, gitRepo, o.forgejoIssue); err != nil {
		panic(err)
	}
}

func newIssue() generic.NodeDriverInterface {
	return &issue{}
}
