// Copyright 2019 Gitea. All rights reserved.
// SPDX-License-Identifier: MIT

package task

import (
	"context"
	"errors"
	"fmt"

	admin_model "forgejo.org/models/admin"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	base "forgejo.org/modules/migration"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/secret"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	repo_service "forgejo.org/services/repository"
)

// taskQueue is a global queue of tasks
var taskQueue *queue.WorkerPoolQueue[*admin_model.Task]

// Run a task
func Run(ctx context.Context, t *admin_model.Task) error {
	switch t.Type {
	case structs.TaskTypeMigrateRepo:
		return runMigrateTask(ctx, t)
	default:
		return fmt.Errorf("Unknown task type: %d", t.Type)
	}
}

// Init will start the service to get all unfinished tasks and run them
func Init() error {
	taskQueue = queue.CreateSimpleQueue(graceful.GetManager().ShutdownContext(), "task", handler)
	if taskQueue == nil {
		return errors.New("unable to create task queue")
	}
	go graceful.GetManager().RunWithCancel(taskQueue)
	return nil
}

func handler(items ...*admin_model.Task) []*admin_model.Task {
	for _, task := range items {
		if err := Run(db.DefaultContext, task); err != nil {
			log.Error("Run task failed: %v", err)
		}
	}
	return nil
}

// MigrateRepository add migration repository to task
func MigrateRepository(ctx context.Context, doer, u *user_model.User, opts base.MigrateOptions) error {
	task, err := CreateMigrateTask(ctx, doer, u, opts)
	if err != nil {
		return err
	}

	return taskQueue.Push(task)
}

// CreateMigrateTask creates a migrate task
func CreateMigrateTask(ctx context.Context, doer, u *user_model.User, opts base.MigrateOptions) (*admin_model.Task, error) {
	// encrypt credentials for persistence
	var err error
	opts.CloneAddrEncrypted, err = secret.EncryptSecret(setting.SecretKey, opts.CloneAddr)
	if err != nil {
		return nil, err
	}
	opts.CloneAddr = util.SanitizeCredentialURLs(opts.CloneAddr)
	opts.AuthPasswordEncrypted, err = secret.EncryptSecret(setting.SecretKey, opts.AuthPassword)
	if err != nil {
		return nil, err
	}
	opts.AuthPassword = ""
	opts.AuthTokenEncrypted, err = secret.EncryptSecret(setting.SecretKey, opts.AuthToken)
	if err != nil {
		return nil, err
	}
	opts.AuthToken = ""
	bs, err := json.Marshal(&opts)
	if err != nil {
		return nil, err
	}

	task := &admin_model.Task{
		DoerID:         doer.ID,
		OwnerID:        u.ID,
		Type:           structs.TaskTypeMigrateRepo,
		Status:         structs.TaskStatusQueued,
		PayloadContent: string(bs),
	}

	if err := admin_model.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	repo, err := repo_service.CreateRepositoryDirectly(ctx, doer, u, repo_service.CreateRepoOptions{
		Name:           opts.RepoName,
		Description:    opts.Description,
		OriginalURL:    opts.OriginalURL,
		GitServiceType: opts.GitServiceType,
		IsPrivate:      opts.Private || setting.Repository.ForcePrivate,
		IsMirror:       opts.Mirror,
		Status:         repo_model.RepositoryBeingMigrated,
	})
	if err != nil {
		task.EndTime = timeutil.TimeStampNow()
		task.Status = structs.TaskStatusFailed
		err2 := task.UpdateCols(ctx, "end_time", "status")
		if err2 != nil {
			log.Error("UpdateCols Failed: %v", err2.Error())
		}
		return nil, err
	}

	task.RepoID = repo.ID
	if err = task.UpdateCols(ctx, "repo_id"); err != nil {
		return nil, err
	}

	return task, nil
}

// RetryMigrateTask retry a migrate task
func RetryMigrateTask(ctx context.Context, repoID int64) error {
	migratingTask, err := admin_model.GetMigratingTask(ctx, repoID)
	if err != nil {
		log.Error("GetMigratingTask: %v", err)
		return err
	}
	if migratingTask.Status == structs.TaskStatusQueued || migratingTask.Status == structs.TaskStatusRunning {
		return nil
	}

	// TODO Need to removing the storage/database garbage brought by the failed task

	// Reset task status and messages
	migratingTask.Status = structs.TaskStatusQueued
	migratingTask.Message = ""
	if err = migratingTask.UpdateCols(ctx, "status", "message"); err != nil {
		log.Error("task.UpdateCols failed: %v", err)
		return err
	}

	return taskQueue.Push(migratingTask)
}

func SetMigrateTaskMessage(ctx context.Context, repoID int64, message string) error {
	migratingTask, err := admin_model.GetMigratingTask(ctx, repoID)
	if err != nil {
		log.Error("GetMigratingTask: %v", err)
		return err
	}

	migratingTask.Message = message
	if err = migratingTask.UpdateCols(ctx, "message"); err != nil {
		log.Error("task.UpdateCols failed: %v", err)
		return err
	}
	return nil
}
