// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package routers

import (
	"context"
	"reflect"
	"runtime"

	"forgejo.org/models"
	asymkey_model "forgejo.org/models/asymkey"
	authmodel "forgejo.org/models/auth"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/eventsource"
	"forgejo.org/modules/git"
	"forgejo.org/modules/highlight"
	"forgejo.org/modules/log"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/external"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/ssh"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/svg"
	"forgejo.org/modules/system"
	"forgejo.org/modules/templates"
	"forgejo.org/modules/translation"
	"forgejo.org/modules/web"
	actions_router "forgejo.org/routers/api/actions"
	forgejo "forgejo.org/routers/api/forgejo/v1"
	packages_router "forgejo.org/routers/api/packages"
	apiv1 "forgejo.org/routers/api/v1"
	"forgejo.org/routers/common"
	"forgejo.org/routers/private"
	web_routers "forgejo.org/routers/web"
	actions_service "forgejo.org/services/actions"
	"forgejo.org/services/auth"
	"forgejo.org/services/auth/source/oauth2"
	"forgejo.org/services/automerge"
	"forgejo.org/services/cron"
	feed_service "forgejo.org/services/feed"
	indexer_service "forgejo.org/services/indexer"
	"forgejo.org/services/mailer"
	mailer_incoming "forgejo.org/services/mailer/incoming"
	markup_service "forgejo.org/services/markup"
	repo_migrations "forgejo.org/services/migrations"
	mirror_service "forgejo.org/services/mirror"
	pull_service "forgejo.org/services/pull"
	release_service "forgejo.org/services/release"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/services/repository/archiver"
	"forgejo.org/services/task"
	"forgejo.org/services/uinotification"
	"forgejo.org/services/webhook"
)

func mustInit(fn func() error) {
	err := fn()
	if err != nil {
		ptr := reflect.ValueOf(fn).Pointer()
		fi := runtime.FuncForPC(ptr)
		log.Fatal("%s failed: %v", fi.Name(), err)
	}
}

func mustInitCtx(ctx context.Context, fn func(ctx context.Context) error) {
	err := fn(ctx)
	if err != nil {
		ptr := reflect.ValueOf(fn).Pointer()
		fi := runtime.FuncForPC(ptr)
		log.Fatal("%s(ctx) failed: %v", fi.Name(), err)
	}
}

func syncAppConfForGit(ctx context.Context) error {
	runtimeState := new(system.RuntimeState)
	if err := system.AppState.Get(ctx, runtimeState); err != nil {
		return err
	}

	updated := false
	if runtimeState.LastAppPath != setting.AppPath {
		log.Info("AppPath changed from '%s' to '%s'", runtimeState.LastAppPath, setting.AppPath)
		runtimeState.LastAppPath = setting.AppPath
		updated = true
	}
	if runtimeState.LastCustomConf != setting.CustomConf {
		log.Info("CustomConf changed from '%s' to '%s'", runtimeState.LastCustomConf, setting.CustomConf)
		runtimeState.LastCustomConf = setting.CustomConf
		updated = true
	}

	if updated {
		log.Info("re-sync repository hooks ...")
		mustInitCtx(ctx, repo_service.SyncRepositoryHooks)

		log.Info("re-write ssh public keys ...")
		mustInitCtx(ctx, asymkey_model.RewriteAllPublicKeys)

		return system.AppState.Set(ctx, runtimeState)
	}
	return nil
}

func InitWebInstallPage(ctx context.Context) {
	translation.InitLocales(ctx)
	setting.LoadSettingsForInstall()
	mustInit(svg.Init)
}

// InitWebInstalled is for global installed configuration.
func InitWebInstalled(ctx context.Context) {
	mustInitCtx(ctx, git.InitFull)
	log.Info("Git version: %s (home: %s)", git.VersionInfo(), git.HomeDir())

	// Setup i18n
	translation.InitLocales(ctx)

	setting.LoadSettings()
	mustInit(storage.Init)

	mailer.NewContext(ctx)
	mustInit(cache.Init)
	mustInit(feed_service.Init)
	mustInit(uinotification.Init)
	mustInitCtx(ctx, archiver.Init)

	highlight.NewContext()
	external.RegisterRenderers()
	markup.Init(markup_service.ProcessorHelper())

	if setting.EnableSQLite3 {
		log.Info("SQLite3 support is enabled")
	} else if setting.Database.Type.IsSQLite3() {
		log.Fatal("SQLite3 support is disabled, but it is used for database setting. Please get or build a Forgejo release with SQLite3 support.")
	}

	mustInitCtx(ctx, common.InitDBEngine)
	log.Info("ORM engine initialization successful!")
	mustInit(system.Init)
	mustInitCtx(ctx, oauth2.Init)

	mustInit(release_service.Init)

	mustInitCtx(ctx, models.Init)
	mustInitCtx(ctx, authmodel.Init)
	mustInitCtx(ctx, repo_service.Init)

	// Booting long running goroutines.
	mustInit(indexer_service.Init)

	mirror_service.InitSyncMirrors()
	mustInit(webhook.Init)
	mustInit(pull_service.Init)
	mustInit(automerge.Init)
	mustInit(task.Init)
	mustInit(repo_migrations.Init)
	eventsource.GetManager().Init()
	mustInitCtx(ctx, mailer_incoming.Init)

	mustInitCtx(ctx, syncAppConfForGit)

	mustInit(ssh.Init)

	auth.Init()
	mustInit(svg.Init)

	actions_service.Init()

	// Finally start up the cron
	cron.NewContext(ctx)
}

// NormalRoutes represents non install routes
func NormalRoutes() *web.Route {
	_ = templates.HTMLRenderer()
	r := web.NewRoute()
	r.Use(common.ProtocolMiddlewares()...)

	r.Mount("/", web_routers.Routes())
	r.Mount("/api/v1", apiv1.Routes())
	r.Mount("/api/forgejo/v1", forgejo.Routes())
	r.Mount("/api/internal", private.Routes())

	r.Post("/-/fetch-redirect", common.FetchRedirectDelegate)

	if setting.Packages.Enabled {
		// This implements package support for most package managers
		r.Mount("/api/packages", packages_router.CommonRoutes())
		// This implements the OCI API (Note this is not preceded by /api but is instead /v2)
		r.Mount("/v2", packages_router.ContainerRoutes())
	}

	if setting.Actions.Enabled {
		prefix := "/api/actions"
		r.Mount(prefix, actions_router.Routes(prefix))

		// TODO: Pipeline api used for runner internal communication with gitea server. but only artifact is used for now.
		// In Github, it uses ACTIONS_RUNTIME_URL=https://pipelines.actions.githubusercontent.com/fLgcSHkPGySXeIFrg8W8OBSfeg3b5Fls1A1CwX566g8PayEGlg/
		// TODO: this prefix should be generated with a token string with runner ?
		prefix = "/api/actions_pipeline"
		r.Mount(prefix, actions_router.ArtifactsRoutes(prefix))
		prefix = actions_router.ArtifactV4RouteBase
		r.Mount(prefix, actions_router.ArtifactsV4Routes(prefix))
	}

	return r
}
