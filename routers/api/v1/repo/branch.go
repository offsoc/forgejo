// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
)

// GetBranch get a branch of a repository
func GetBranch(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/branches/{branch} repository repoGetBranch
	// ---
	// summary: Retrieve a specific branch from a repository, including its effective branch protection
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: branch
	//   in: path
	//   description: branch to get
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Branch"
	if ctx.Repo.TreePath != "" {
		// if TreePath != "", then URL contained extra slashes
		// (i.e. "master/subbranch" instead of "master"), so branch does
		// not exist
		ctx.NotFound()
		return
	}
	branch, err := ctx.Repo.Repository.GetBranch(ctx.Repo.BranchName)
	if err != nil {
		if git.IsErrBranchNotExist(err) {
			ctx.NotFound(err)
		} else {
			ctx.Error(500, "GetBranch", err)
		}
		return
	}

	c, err := branch.GetCommit()
	if err != nil {
		ctx.Error(500, "GetCommit", err)
		return
	}

	branchProtection, err := ctx.Repo.Repository.GetBranchProtection(ctx.Repo.BranchName)
	if err != nil {
		ctx.Error(500, "GetBranchProtection", err)
		return
	}

	ctx.JSON(200, convert.ToBranch(ctx.Repo.Repository, branch, c, branchProtection, ctx.User, ctx.Repo.IsAdmin()))
}

// ListBranches list all the branches of a repository
func ListBranches(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/branches repository repoListBranches
	// ---
	// summary: List a repository's branches
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/BranchList"
	branches, err := ctx.Repo.Repository.GetBranches()
	if err != nil {
		ctx.Error(500, "GetBranches", err)
		return
	}

	apiBranches := make([]*api.Branch, len(branches))
	for i := range branches {
		c, err := branches[i].GetCommit()
		if err != nil {
			ctx.Error(500, "GetCommit", err)
			return
		}
		branchProtection, err := ctx.Repo.Repository.GetBranchProtection(branches[i].Name)
		if err != nil {
			ctx.Error(500, "GetBranchProtection", err)
			return
		}
		apiBranches[i] = convert.ToBranch(ctx.Repo.Repository, branches[i], c, branchProtection, ctx.User, ctx.Repo.IsAdmin())
	}

	ctx.JSON(200, &apiBranches)
}

// GetBranchProtection gets a branch protection
func GetBranchProtection(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/branch_protections/{id} repository repoGetBranchProtection
	// ---
	// summary: Get a specific branch protection for the repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: ID of the branch protection
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/BranchProtection"

	repo := ctx.Repo.Repository
	bpID := ctx.ParamsInt64(":id")
	bp, err := models.GetProtectedBranchByID(bpID)
	if err != nil {
		ctx.Error(500, "GetProtectedBranchByID", err)
		return
	}
	if bp == nil || bp.RepoID != repo.ID {
		ctx.NotFound()
		return
	}

	ctx.JSON(200, convert.ToBranchProtection(bp))
}

// ListBranchProtections list branch protections for a repo
func ListBranchProtections(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/branch_protections repository repoListBranchProtection
	// ---
	// summary: List branch protections for a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/BranchProtectionList"

	repo := ctx.Repo.Repository
	bps, err := repo.GetProtectedBranches()
	if err != nil {
		ctx.Error(500, "GetProtectedBranches", err)
		return
	}
	apiBps := make([]*api.BranchProtection, len(bps))
	for i := range bps {
		apiBps[i] = convert.ToBranchProtection(bps[i])
	}

	ctx.JSON(200, apiBps)
}

// CreateBranchProtection creates a branch protection for a repo
func CreateBranchProtection(ctx *context.APIContext, form api.CreateBranchProtectionOption) {
	// swagger:operation POST /repos/{owner}/{repo}/branch_protections repository repoCreateBranchProtection
	// ---
	// summary: Create a branch protections for a repository
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateBranchProtectionOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/BranchProtection"

	// Currently protection must match an actual branch
	if !git.IsBranchExist(ctx.Repo.Repository.RepoPath(), form.BranchName) {
		ctx.NotFound()
		return
	}

	protectBranch, err := models.GetProtectedBranchBy(ctx.Repo.Repository.ID, form.BranchName)
	if err != nil {
		ctx.ServerError("GetProtectBranchOfRepoByName", err)
		return
	} else if protectBranch != nil {
		ctx.Error(403, "Branch protection already exist", err)
		return
	}

	var requiredApprovals int64
	if form.RequiredApprovals > 0 {
		requiredApprovals = form.RequiredApprovals
	}

	whitelistUsers, err := models.GetUserIDsByNames(form.PushWhitelistUsernames, false)
	if err != nil {
		ctx.ServerError("GetUserIDsByNames", err)
	}
	mergeWhitelistUsers, err := models.GetUserIDsByNames(form.MergeWhitelistUsernames, false)
	if err != nil {
		ctx.ServerError("GetUserIDsByNames", err)
	}
	approvalsWhitelistUsers, err := models.GetUserIDsByNames(form.ApprovalsWhitelistUsernames, false)
	if err != nil {
		ctx.ServerError("GetUserIDsByNames", err)
	}

	protectBranch = &models.ProtectedBranch{
		ID:                       0,
		RepoID:                   ctx.Repo.Repository.ID,
		BranchName:               form.BranchName,
		CanPush:                  form.EnablePush,
		EnableWhitelist:          form.EnablePush && form.EnablePushWhitelist,
		EnableMergeWhitelist:     form.EnableMergeWhitelist,
		WhitelistDeployKeys:      form.EnablePush && form.EnablePushWhitelist && form.PushWhitelistDeployKeys,
		EnableStatusCheck:        form.EnableStatusCheck,
		StatusCheckContexts:      form.StatusCheckContexts,
		EnableApprovalsWhitelist: form.EnableApprovalsWhitelist,
		RequiredApprovals:        requiredApprovals,
	}

	err = models.UpdateProtectBranch(ctx.Repo.Repository, protectBranch, models.WhitelistOptions{
		UserIDs:          whitelistUsers,
		TeamIDs:          form.PushWhitelistTeamIDs,
		MergeUserIDs:     mergeWhitelistUsers,
		MergeTeamIDs:     form.MergeWhitelistTeamIDs,
		ApprovalsUserIDs: approvalsWhitelistUsers,
		ApprovalsTeamIDs: form.ApprovalsWhitelistTeamIDs,
	})
	if err != nil {
		ctx.ServerError("UpdateProtectBranch", err)
		return
	}

	// Reload from db to get all whitelists
	bp, err := models.GetProtectedBranchBy(ctx.Repo.Repository.ID, form.BranchName)
	if err != nil {
		ctx.Error(500, "GetProtectedBranchByID", err)
		return
	}
	if bp == nil || bp.RepoID != ctx.Repo.Repository.ID {
		ctx.Error(500, "New branch protection not found", err)
		return
	}

	ctx.JSON(200, convert.ToBranchProtection(bp))

}

// EditBranchProtection edits a branch protection for a repo
func EditBranchProtection(ctx *context.APIContext, form api.EditBranchProtectionOption) {
	// swagger:operation PUT /repos/{owner}/{repo}/branch_protections/{id} repository repoEditBranchProtection
	// ---
	// summary: Edit a branch protections for a repository. Only fields that are set will be changed
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: ID of the branch protection
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditBranchProtectionOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/BranchProtection"

	protectBranch, err := models.GetProtectedBranchByID(ctx.ParamsInt64(":id"))
	if err != nil || protectBranch == nil {
		ctx.ServerError("GetProtectBranchOfRepoByName", err)
		return
	} else if protectBranch.RepoID != ctx.Repo.Repository.ID {
		ctx.Error(403, "Branch protection for repo does not exist", err)
		return
	}

	if form.EnablePush != nil {
		if !*form.EnablePush {
			protectBranch.CanPush = false
			protectBranch.EnableWhitelist = false
			protectBranch.WhitelistDeployKeys = false
		} else {
			protectBranch.CanPush = true
			if form.EnablePushWhitelist != nil {
				if !*form.EnablePushWhitelist {
					protectBranch.EnableWhitelist = false
					protectBranch.WhitelistDeployKeys = false
				} else {
					protectBranch.EnableWhitelist = true
					if form.PushWhitelistDeployKeys != nil {
						protectBranch.WhitelistDeployKeys = *form.PushWhitelistDeployKeys
					}
				}
			}
		}
	}

	if form.EnableMergeWhitelist != nil {
		protectBranch.EnableMergeWhitelist = *form.EnableMergeWhitelist
	}

	if form.EnableStatusCheck != nil {
		protectBranch.StatusCheckContexts = form.StatusCheckContexts
	}

	if form.RequiredApprovals != nil && *form.RequiredApprovals >= 0 {
		protectBranch.RequiredApprovals = *form.RequiredApprovals
	}

	if form.EnableApprovalsWhitelist != nil {
		protectBranch.EnableApprovalsWhitelist = *form.EnableApprovalsWhitelist
	}

	var whitelistUsers []int64
	if form.PushWhitelistUsernames != nil {
		whitelistUsers, err = models.GetUserIDsByNames(form.PushWhitelistUsernames, false)
		if err != nil {
			ctx.ServerError("GetUserIDsByNames", err)
		}
	} else {
		whitelistUsers = protectBranch.WhitelistUserIDs
	}
	var whitelistTeams []int64
	if form.PushWhitelistTeamIDs != nil {
		whitelistTeams = form.PushWhitelistTeamIDs
	} else {
		whitelistTeams = protectBranch.WhitelistTeamIDs
	}

	var mergeWhitelistUsers []int64
	if form.MergeWhitelistUsernames != nil {
		mergeWhitelistUsers, err = models.GetUserIDsByNames(form.MergeWhitelistUsernames, false)
		if err != nil {
			ctx.ServerError("GetUserIDsByNames", err)
		}
	} else {
		mergeWhitelistUsers = protectBranch.MergeWhitelistUserIDs
	}
	var mergeWhitelistTeams []int64
	if form.MergeWhitelistTeamIDs != nil {
		mergeWhitelistTeams = form.MergeWhitelistTeamIDs
	} else {
		mergeWhitelistTeams = protectBranch.MergeWhitelistTeamIDs
	}

	var approvalsWhitelistUsers []int64
	if form.ApprovalsWhitelistUsernames != nil {
		approvalsWhitelistUsers, err = models.GetUserIDsByNames(form.ApprovalsWhitelistUsernames, false)
		if err != nil {
			ctx.ServerError("GetUserIDsByNames", err)
		}
	} else {
		approvalsWhitelistUsers = protectBranch.ApprovalsWhitelistUserIDs
	}
	var approvalsWhitelistTeams []int64
	if form.ApprovalsWhitelistTeamIDs != nil {
		approvalsWhitelistTeams = form.ApprovalsWhitelistTeamIDs
	} else {
		approvalsWhitelistTeams = protectBranch.ApprovalsWhitelistTeamIDs
	}

	err = models.UpdateProtectBranch(ctx.Repo.Repository, protectBranch, models.WhitelistOptions{
		UserIDs:          whitelistUsers,
		TeamIDs:          whitelistTeams,
		MergeUserIDs:     mergeWhitelistUsers,
		MergeTeamIDs:     mergeWhitelistTeams,
		ApprovalsUserIDs: approvalsWhitelistUsers,
		ApprovalsTeamIDs: approvalsWhitelistTeams,
	})
	if err != nil {
		ctx.ServerError("UpdateProtectBranch", err)
		return
	}

	// Reload from db to ensure get all whitelists
	bp, err := models.GetProtectedBranchByID(ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.Error(500, "GetProtectedBranchByID", err)
		return
	}
	if bp == nil || bp.RepoID != ctx.Repo.Repository.ID {
		ctx.Error(500, "New branch protection not found", err)
		return
	}

	ctx.JSON(200, convert.ToBranchProtection(bp))
}

// DeleteBranchProtection deletes a branch protection for a repo
func DeleteBranchProtection(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/branch_protections/{id} repository repoDeleteBranchProtection
	// ---
	// summary: Delete a specific branch protection for the repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: ID of the branch protection
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	if err := ctx.Repo.Repository.DeleteProtectedBranch(ctx.ParamsInt64(":id")); err != nil {
		ctx.ServerError("DeleteProtectedBranch", err)
		return
	}

	ctx.Status(204)
}
