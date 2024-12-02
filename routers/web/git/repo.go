// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"fmt"
	"net/http"
	"strings"

	actions_model "code.gitea.io/gitea/models/actions"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/context"
	repo_service "code.gitea.io/gitea/services/repository"
)

type serviceHandlerRepo struct {
	repo    *repo_model.Repository
	isWiki  bool
	environ []string
}

func (h *serviceHandlerRepo) Init(ctx *context.Context) bool {
	username := ctx.Params(":username")
	reponame := strings.TrimSuffix(ctx.Params(":reponame"), ".git")

	if ctx.FormString("go-get") == "1" {
		context.EarlyResponseForGoGetMeta(ctx)
		return false
	}

	var isPull, receivePack bool
	service := ctx.FormString("service")
	if service == "git-receive-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-receive-pack") {
		isPull = false
		receivePack = true
	} else if service == "git-upload-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-pack") {
		isPull = true
	} else if service == "git-upload-archive" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-archive") {
		isPull = true
	} else {
		isPull = ctx.Req.Method == "GET"
	}

	var accessMode perm.AccessMode
	if isPull {
		accessMode = perm.AccessModeRead
	} else {
		accessMode = perm.AccessModeWrite
	}

	isWiki := false
	unitType := unit.TypeCode

	if strings.HasSuffix(reponame, ".wiki") {
		isWiki = true
		unitType = unit.TypeWiki
		reponame = reponame[:len(reponame)-5]
	}

	owner := ctx.ContextUser
	if !owner.IsOrganization() && !owner.IsActive {
		ctx.PlainText(http.StatusForbidden, "Repository cannot be accessed. You cannot push or open issues/pull-requests.")
		return false
	}

	repoExist := true
	repo, err := repo_model.GetRepositoryByName(ctx, owner.ID, reponame)
	if err != nil {
		if !repo_model.IsErrRepoNotExist(err) {
			ctx.ServerError("GetRepositoryByName", err)
			return false
		}

		if redirectRepoID, err := repo_model.LookupRedirect(ctx, owner.ID, reponame); err == nil {
			context.RedirectToRepo(ctx.Base, redirectRepoID)
			return false
		}
		repoExist = false
	}

	// Don't allow pushing if the repo is archived
	if repoExist && repo.IsArchived && !isPull {
		ctx.PlainText(http.StatusForbidden, "This repo is archived. You can view files and clone it, but cannot push or open issues/pull-requests.")
		return false
	}

	// Only public pull don't need auth.
	isPublicPull := repoExist && !repo.IsPrivate && isPull
	var (
		askAuth = !isPublicPull || setting.Service.RequireSignInView
		environ []string
	)

	// don't allow anonymous pulls if organization is not public
	if isPublicPull {
		if err := repo.LoadOwner(ctx); err != nil {
			ctx.ServerError("LoadOwner", err)
			return false
		}

		askAuth = askAuth || (repo.Owner.Visibility != structs.VisibleTypePublic)
	}

	// check access
	if askAuth {
		// rely on the results of Contexter
		if !ctx.IsSigned {
			// TODO: support digit auth - which would be Authorization header with digit
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
			ctx.Error(http.StatusUnauthorized)
			return false
		}

		context.CheckRepoScopedToken(ctx, repo, auth_model.GetScopeLevelFromAccessMode(accessMode))
		if ctx.Written() {
			return false
		}

		if ctx.IsBasicAuth && ctx.Data["IsApiToken"] != true && ctx.Data["IsActionsToken"] != true {
			_, err = auth_model.GetTwoFactorByUID(ctx, ctx.Doer.ID)
			if err == nil {
				// TODO: This response should be changed to "invalid credentials" for security reasons once the expectation behind it (creating an app token to authenticate) is properly documented
				ctx.PlainText(http.StatusUnauthorized, "Users with two-factor authentication enabled cannot perform HTTP/HTTPS operations via plain username and password. Please create and use a personal access token on the user settings page")
				return false
			} else if !auth_model.IsErrTwoFactorNotEnrolled(err) {
				ctx.ServerError("IsErrTwoFactorNotEnrolled", err)
				return false
			}
		}

		if !ctx.Doer.IsActive || ctx.Doer.ProhibitLogin {
			ctx.PlainText(http.StatusForbidden, "Your account is disabled.")
			return false
		}

		environ = []string{
			repo_module.EnvRepoUsername + "=" + username,
			repo_module.EnvRepoName + "=" + reponame,
			repo_module.EnvPusherName + "=" + ctx.Doer.Name,
			repo_module.EnvPusherID + fmt.Sprintf("=%d", ctx.Doer.ID),
			repo_module.EnvAppURL + "=" + setting.AppURL,
		}

		if repoExist {
			// Because of special ref "refs/for" .. , need delay write permission check
			if git.SupportProcReceive {
				accessMode = perm.AccessModeRead
			}

			if ctx.Data["IsActionsToken"] == true {
				taskID := ctx.Data["ActionsTaskID"].(int64)
				task, err := actions_model.GetTaskByID(ctx, taskID)
				if err != nil {
					ctx.ServerError("GetTaskByID", err)
					return false
				}
				if task.RepoID != repo.ID {
					ctx.PlainText(http.StatusForbidden, "User permission denied")
					return false
				}

				if task.IsForkPullRequest {
					if accessMode > perm.AccessModeRead {
						ctx.PlainText(http.StatusForbidden, "User permission denied")
						return false
					}
					environ = append(environ, fmt.Sprintf("%s=%d", repo_module.EnvActionPerm, perm.AccessModeRead))
				} else {
					if accessMode > perm.AccessModeWrite {
						ctx.PlainText(http.StatusForbidden, "User permission denied")
						return false
					}
					environ = append(environ, fmt.Sprintf("%s=%d", repo_module.EnvActionPerm, perm.AccessModeWrite))
				}
			} else {
				p, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
				if err != nil {
					ctx.ServerError("GetUserRepoPermission", err)
					return false
				}

				if !p.CanAccess(accessMode, unitType) {
					ctx.PlainText(http.StatusNotFound, "Repository not found")
					return false
				}
			}

			if !isPull && repo.IsMirror {
				ctx.PlainText(http.StatusForbidden, "mirror repository is read-only")
				return false
			}
		}

		if !ctx.Doer.KeepEmailPrivate {
			environ = append(environ, repo_module.EnvPusherEmail+"="+ctx.Doer.Email)
		}

		if isWiki {
			environ = append(environ, repo_module.EnvRepoIsWiki+"=true")
		} else {
			environ = append(environ, repo_module.EnvRepoIsWiki+"=false")
		}
	}

	if !repoExist {
		if !receivePack {
			ctx.PlainText(http.StatusNotFound, "Repository not found")
			return false
		}

		if isWiki { // you cannot send wiki operation before create the repository
			ctx.PlainText(http.StatusNotFound, "Repository not found")
			return false
		}

		if owner.IsOrganization() && !setting.Repository.EnablePushCreateOrg {
			ctx.PlainText(http.StatusForbidden, "Push to create is not enabled for organizations.")
			return false
		}
		if !owner.IsOrganization() && !setting.Repository.EnablePushCreateUser {
			ctx.PlainText(http.StatusForbidden, "Push to create is not enabled for users.")
			return false
		}

		// Return dummy payload if GET receive-pack
		if ctx.Req.Method == http.MethodGet {
			dummyInfoRefs(ctx)
			return false
		}

		repo, err = repo_service.PushCreateRepo(ctx, ctx.Doer, owner, reponame)
		if err != nil {
			log.Error("pushCreateRepo: %v", err)
			ctx.Status(http.StatusNotFound)
			return false
		}
	}

	if isWiki {
		// Ensure the wiki is enabled before we allow access to it
		if _, err := repo.GetUnit(ctx, unit.TypeWiki); err != nil {
			if repo_model.IsErrUnitTypeNotExist(err) {
				ctx.PlainText(http.StatusForbidden, "repository wiki is disabled")
				return false
			}
			log.Error("Failed to get the wiki unit in %-v Error: %v", repo, err)
			ctx.ServerError("GetUnit(UnitTypeWiki) for "+repo.FullName(), err)
			return false
		}
	}

	environ = append(environ, repo_module.EnvRepoID+fmt.Sprintf("=%d", repo.ID))

	ctx.Req.URL.Path = strings.ToLower(ctx.Req.URL.Path) // blue: In case some repo name has upper case name

	h.repo = repo
	h.isWiki = isWiki
	h.environ = environ

	return true
}

func (h *serviceHandlerRepo) GetRepoPath() string {
	if h.isWiki {
		return h.repo.WikiPath()
	}
	return h.repo.RepoPath()
}

func (h *serviceHandlerRepo) GetEnviron() []string {
	return h.environ
}
