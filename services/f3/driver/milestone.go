// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &milestone{}

type milestone struct {
	common

	forgejoMilestone *issues_model.Milestone
}

func (o *milestone) SetNative(milestone any) {
	o.forgejoMilestone = milestone.(*issues_model.Milestone)
}

func (o *milestone) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoMilestone.ID)
}

func (o *milestone) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *milestone) ToFormat() f3.Interface {
	if o.forgejoMilestone == nil {
		return o.NewFormat()
	}
	return &f3.Milestone{
		Common:      f3.NewCommon(fmt.Sprintf("%d", o.forgejoMilestone.ID)),
		Title:       o.forgejoMilestone.Name,
		Description: o.forgejoMilestone.Content,
		Created:     o.forgejoMilestone.CreatedUnix.AsTime(),
		Updated:     o.forgejoMilestone.UpdatedUnix.AsTimePtr(),
		Deadline:    o.forgejoMilestone.DeadlineUnix.AsTimePtr(),
		State:       string(o.forgejoMilestone.State()),
	}
}

func (o *milestone) FromFormat(content f3.Interface) {
	milestone := content.(*f3.Milestone)

	var deadline timeutil.TimeStamp
	if milestone.Deadline != nil {
		deadline = timeutil.TimeStamp(milestone.Deadline.Unix())
	}
	if deadline == 0 {
		deadline = timeutil.TimeStamp(time.Date(9999, 1, 1, 0, 0, 0, 0, setting.DefaultUILocation).Unix())
	}

	var closed timeutil.TimeStamp
	if milestone.Closed != nil {
		closed = timeutil.TimeStamp(milestone.Closed.Unix())
	}

	if milestone.Created.IsZero() {
		if milestone.Updated != nil {
			milestone.Created = *milestone.Updated
		} else if milestone.Deadline != nil {
			milestone.Created = *milestone.Deadline
		} else {
			milestone.Created = time.Now()
		}
	}
	if milestone.Updated == nil || milestone.Updated.IsZero() {
		milestone.Updated = &milestone.Created
	}

	o.forgejoMilestone = &issues_model.Milestone{
		ID:             f3_util.ParseInt(milestone.GetID()),
		Name:           milestone.Title,
		Content:        milestone.Description,
		IsClosed:       milestone.State == f3.MilestoneStateClosed,
		CreatedUnix:    timeutil.TimeStamp(milestone.Created.Unix()),
		UpdatedUnix:    timeutil.TimeStamp(milestone.Updated.Unix()),
		ClosedDateUnix: closed,
		DeadlineUnix:   deadline,
	}
}

func (o *milestone) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	id := node.GetID().Int64()

	milestone, err := issues_model.GetMilestoneByRepoID(ctx, project, id)
	if issues_model.IsErrMilestoneNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("milestone %v %w", id, err))
	}
	o.forgejoMilestone = milestone
	return true
}

func (o *milestone) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoMilestone.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoMilestone.ID).Cols("name", "description", "is_closed", "deadline_unix").Update(o.forgejoMilestone); err != nil {
		panic(fmt.Errorf("UpdateMilestoneCols: %v %v", o.forgejoMilestone, err))
	}
}

func (o *milestone) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	o.forgejoMilestone.RepoID = f3_tree.GetProjectID(o.GetNode())
	if err := issues_model.NewMilestone(ctx, o.forgejoMilestone); err != nil {
		panic(err)
	}
	o.Trace("milestone created %d", o.forgejoMilestone.ID)
	return f3_id.NewNodeID(o.forgejoMilestone.ID)
}

func (o *milestone) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())

	if err := issues_model.DeleteMilestoneByRepoID(ctx, project, o.forgejoMilestone.ID); err != nil {
		panic(err)
	}
}

func newMilestone() generic.NodeDriverInterface {
	return &milestone{}
}
