// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package actions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/perm"
	"forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/modules/webhook"
	"forgejo.org/services/convert"

	"github.com/nektos/act/pkg/jobparser"
	act_model "github.com/nektos/act/pkg/model"
)

type InputRequiredErr struct {
	Name string
}

func (err InputRequiredErr) Error() string {
	return fmt.Sprintf("input required for '%s'", err.Name)
}

func IsInputRequiredErr(err error) bool {
	_, ok := err.(InputRequiredErr)
	return ok
}

type Workflow struct {
	WorkflowID string
	Ref        string
	Commit     *git.Commit
	GitEntry   *git.TreeEntry
}

type InputValueGetter func(key string) string

func (entry *Workflow) Dispatch(ctx context.Context, inputGetter InputValueGetter, repo *repo_model.Repository, doer *user.User) (r *actions_model.ActionRun, j []string, err error) {
	content, err := actions.GetContentFromEntry(entry.GitEntry)
	if err != nil {
		return nil, nil, err
	}

	wf, err := act_model.ReadWorkflow(bytes.NewReader(content))
	if err != nil {
		return nil, nil, err
	}

	fullWorkflowID := ".forgejo/workflows/" + entry.WorkflowID

	title := wf.Name
	if len(title) < 1 {
		title = fullWorkflowID
	}

	inputs := make(map[string]string)
	if workflowDispatch := wf.WorkflowDispatchConfig(); workflowDispatch != nil {
		for key, input := range workflowDispatch.Inputs {
			val := inputGetter(key)
			if len(val) == 0 {
				val = input.Default
				if len(val) == 0 {
					if input.Required {
						name := input.Description
						if len(name) == 0 {
							name = key
						}
						return nil, nil, InputRequiredErr{Name: name}
					}
					continue
				}
			} else if input.Type == "boolean" {
				// Since "boolean" inputs are rendered as a checkbox in html, the value inside the form is "on"
				val = strconv.FormatBool(val == "on")
			}
			inputs[key] = val
		}
	}

	if int64(len(inputs)) > setting.Actions.LimitDispatchInputs {
		return nil, nil, errors.New("to many inputs")
	}

	jobNames := util.KeysOfMap(wf.Jobs)

	payload := &structs.WorkflowDispatchPayload{
		Inputs:     inputs,
		Ref:        entry.Ref,
		Repository: convert.ToRepo(ctx, repo, access.Permission{AccessMode: perm.AccessModeNone}),
		Sender:     convert.ToUser(ctx, doer, nil),
		Workflow:   fullWorkflowID,
	}

	p, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}

	run := &actions_model.ActionRun{
		Title:         title,
		RepoID:        repo.ID,
		Repo:          repo,
		OwnerID:       repo.OwnerID,
		WorkflowID:    entry.WorkflowID,
		TriggerUserID: doer.ID,
		TriggerUser:   doer,
		Ref:           entry.Ref,
		CommitSHA:     entry.Commit.ID.String(),
		Event:         webhook.HookEventWorkflowDispatch,
		EventPayload:  string(p),
		TriggerEvent:  string(webhook.HookEventWorkflowDispatch),
		Status:        actions_model.StatusWaiting,
	}

	vars, err := actions_model.GetVariablesOfRun(ctx, run)
	if err != nil {
		return nil, nil, err
	}

	jobs, err := jobparser.Parse(content, jobparser.WithVars(vars))
	if err != nil {
		return nil, nil, err
	}

	return run, jobNames, actions_model.InsertRun(ctx, run, jobs)
}

func GetWorkflowFromCommit(gitRepo *git.Repository, ref, workflowID string) (*Workflow, error) {
	ref, err := gitRepo.ExpandRef(ref)
	if err != nil {
		return nil, err
	}

	commit, err := gitRepo.GetCommit(ref)
	if err != nil {
		return nil, err
	}

	entries, err := actions.ListWorkflows(commit)
	if err != nil {
		return nil, err
	}

	var workflowEntry *git.TreeEntry
	for _, entry := range entries {
		if entry.Name() == workflowID {
			workflowEntry = entry
			break
		}
	}
	if workflowEntry == nil {
		return nil, errors.New("workflow not found")
	}

	return &Workflow{
		WorkflowID: workflowID,
		Ref:        ref,
		Commit:     commit,
		GitEntry:   workflowEntry,
	}, nil
}
