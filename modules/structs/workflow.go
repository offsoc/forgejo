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
}
