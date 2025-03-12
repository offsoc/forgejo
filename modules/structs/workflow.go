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
	ReturnRunInfo bool `json:"return_run_info" default:"false"`
}

// DispatchWorkflowRun represents a workflow run
// swagger:model
type DispatchWorkflowRun struct {
	// the workflow run id
	ID int64 `json:"id"`
	// the jobs name
	Jobs []string `json:"jobs"`
}
