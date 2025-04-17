// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/timeutil"
	release_service "forgejo.org/services/release"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &release{}

type release struct {
	common

	forgejoRelease *repo_model.Release
}

func (o *release) SetNative(release any) {
	o.forgejoRelease = release.(*repo_model.Release)
}

func (o *release) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoRelease.ID)
}

func (o *release) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *release) ToFormat() f3.Interface {
	if o.forgejoRelease == nil {
		return o.NewFormat()
	}
	return &f3.Release{
		Common:          f3.NewCommon(fmt.Sprintf("%d", o.forgejoRelease.ID)),
		TagName:         o.forgejoRelease.TagName,
		TargetCommitish: o.forgejoRelease.Target,
		Name:            o.forgejoRelease.Title,
		Body:            o.forgejoRelease.Note,
		Draft:           o.forgejoRelease.IsDraft,
		Prerelease:      o.forgejoRelease.IsPrerelease,
		PublisherID:     f3_tree.NewUserReference(o.forgejoRelease.Publisher.ID),
		Created:         o.forgejoRelease.CreatedUnix.AsTime(),
	}
}

func (o *release) FromFormat(content f3.Interface) {
	release := content.(*f3.Release)

	o.forgejoRelease = &repo_model.Release{
		ID:          f3_util.ParseInt(release.GetID()),
		PublisherID: release.PublisherID.GetIDAsInt(),
		Publisher: &user_model.User{
			ID: release.PublisherID.GetIDAsInt(),
		},
		TagName:      release.TagName,
		LowerTagName: strings.ToLower(release.TagName),
		Target:       release.TargetCommitish,
		Title:        release.Name,
		Note:         release.Body,
		IsDraft:      release.Draft,
		IsPrerelease: release.Prerelease,
		IsTag:        false,
		CreatedUnix:  timeutil.TimeStamp(release.Created.Unix()),
	}
}

func (o *release) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	release, err := repo_model.GetReleaseByID(ctx, id)
	if repo_model.IsErrReleaseNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("release %v %w", id, err))
	}

	release.Publisher, err = user_model.GetUserByID(ctx, release.PublisherID)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			release.Publisher = user_model.NewGhostUser()
		} else {
			panic(err)
		}
	}

	o.forgejoRelease = release
	return true
}

func (o *release) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoRelease.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoRelease.ID).Cols("title", "note").Update(o.forgejoRelease); err != nil {
		panic(fmt.Errorf("UpdateReleaseCols: %v %v", o.forgejoRelease, err))
	}
}

func (o *release) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	o.forgejoRelease.RepoID = f3_tree.GetProjectID(o.GetNode())

	owner := f3_tree.GetOwnerName(o.GetNode())
	project := f3_tree.GetProjectName(o.GetNode())
	repoPath := repo_model.RepoPath(owner, project)
	gitRepo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		panic(err)
	}
	defer gitRepo.Close()
	if err := release_service.CreateRelease(gitRepo, o.forgejoRelease, "", nil); err != nil {
		panic(err)
	}
	o.Trace("release created %d", o.forgejoRelease.ID)
	return f3_id.NewNodeID(o.forgejoRelease.ID)
}

func (o *release) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	repo, err := repo_model.GetRepositoryByID(ctx, project)
	if err != nil {
		panic(err)
	}

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	if err := release_service.DeleteReleaseByID(ctx, repo, o.forgejoRelease, doer, true); err != nil {
		panic(err)
	}
}

func newRelease() generic.NodeDriverInterface {
	return &release{}
}
