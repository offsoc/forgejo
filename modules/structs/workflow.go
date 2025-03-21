// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package structs

// DispatchWorkflowOption options when dispatching a workflow
// swagger:model
type DispatchWorkflowOption struct {
	// Git reference for the workflow
	//
	// required: true
	Ref string `json:"ref"`
	// Input keys and values configured in the workflow file.
	Inputs map[string]string `json:"inputs"`
	// Flag to return the run info
	// default: false
	ReturnRunInfo bool `json:"return_run_info"`
}

// DispatchWorkflowRun represents a workflow run
// swagger:model
type DispatchWorkflowRun struct {
	// the workflow run id
	ID int64 `json:"id"`
	// a unique number for each run of a repository
	RunNumber int64 `json:"run_number"`
	// the jobs name
	Jobs []string `json:"jobs"`
}
