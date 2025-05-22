// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// ActionRunJob represents a job of a run
// swagger:model
type ActionRunJob struct {
	// the action run job id
	ID int64 `json:"id"`
	// the repository id
	RepoID int64 `json:"repo_id"`
	// the owner id
	OwnerID int64 `json:"owner_id"`
	// the action run job name
	Name string `json:"name"`
	// the action run job needed ids
	Needs []string `json:"needs"`
	// the action run job labels to run on
	RunsOn []string `json:"runs_on"`
	// the action run job latest task id
	TaskID int64 `json:"task_id"`
	// the action run job status
	Status string `json:"status"`
}

// ActionRun represents an action run
// swagger:model
type ActionRun struct {
	// the action run id
	ID    int64       `json:"id"`
	Title string      `json:"title"`
	Repo  *Repository `json:"repository"`
	// the name of workflow file
	WorkflowID string `json:"workflow_id"`
	// a unique number for each run of a repository
	Index       int64 `json:"index_in_repo"`
	TriggerUser *User `json:"trigger_user"`
	ScheduleID  int64
	// the commit/tag/… that caused the run
	PrettyRef    string `json:"prettyref"`
	IsRefDeleted bool   `json:"is_ref_deleted"`
	CommitSHA    string `json:"commit_sha"`
	// If this is triggered by a PR from a forked repository or an untrusted user, we need to check if it is approved and limit permissions when running the workflow.
	IsForkPullRequest bool `json:"is_fork_pull_request"`
	// may need approval if it's a fork pull request
	NeedApproval bool `json:"need_approval"`
	// who approved
	// TODO: should this be a user?
	// TODO: or should this just be removed?
	ApprovedBy int64 `json:"approved_by"`
	// the webhook event that causes the workflow to run
	Event        string `json:"event"`
	EventPayload string `json:"event_payload"`
	// the trigger event defined in the `on` configuration of the triggered workflow
	TriggerEvent string `json:"trigger_event"`
	Status       string `json:"status"`
	// Started and Stopped is used for recording last run time, if rerun happened, they will be reset to 0
	Started  time.Time     `json:"started,omitempty"`
	Stopped  time.Time     `json:"stopped,omitempty"`
	Created  time.Time     `json:"created,omitempty"`
	Updated  time.Time     `json:"updated,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	HTMLURL  string        `json:"run_html_url"`
}
