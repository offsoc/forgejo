// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// QuotaInfo represents information about a user's quota
type QuotaInfo struct {
	Used   QuotaUsed      `json:"used"`
	Groups QuotaGroupList `json:"groups"`
}

// QuotaUsed represents the quota usage of a user
type QuotaUsed struct {
	Size QuotaUsedSize `json:"size"`
}

// QuotaUsedSize represents the size-based quota usage of a user
type QuotaUsedSize struct {
	Repos  QuotaUsedSizeRepos  `json:"repos"`
	Git    QuotaUsedSizeGit    `json:"git"`
	Assets QuotaUsedSizeAssets `json:"assets"`
}

// QuotaUsedSizeRepos represents the size-based repository quota usage of a user
type QuotaUsedSizeRepos struct {
	// Storage size of the user's public repositories
	Public int64 `json:"public"`
	// Storage size of the user's private repositories
	Private int64 `json:"private"`
}

// QuotaUsedSizeGit represents the size-based git (lfs) quota usage of a user
type QuotaUsedSizeGit struct {
	// Storage size of the user's Git LFS objects
	LFS int64 `json:"LFS"`
}

// QuotaUsedSizeAssets represents the size-based asset usage of a user
type QuotaUsedSizeAssets struct {
	Attachments QuotaUsedSizeAssetsAttachments `json:"attachments"`
	// Storage size used for the user's artifacts
	Artifacts int64                       `json:"artifacts"`
	Packages  QuotaUsedSizeAssetsPackages `json:"packages"`
}

// QuotaUsedSizeAssetsAttachments represents the size-based attachment quota usage of a user
type QuotaUsedSizeAssetsAttachments struct {
	// Storage size used for the user's issue & comment attachments
	Issues int64 `json:"issues"`
	// Storage size used for the user's release attachments
	Releases int64 `json:"releases"`
}

// QuotaUsedSizeAssetsPackages represents the size-based package quota usage of a user
type QuotaUsedSizeAssetsPackages struct {
	// Storage suze used for the user's packages
	All int64 `json:"all"`
}

// QuotaRuleInfo contains information about a quota rule
type QuotaRuleInfo struct {
	// Name of the rule (only shown to admins)
	Name string `json:"name,omitempty"`
	// The limit set by the rule
	Limit int64 `json:"limit"`
	// Subjects the rule affects
	Subjects []string `json:"subjects,omitempty"`
}

// QuotaGroupList represents a list of quota groups
type QuotaGroupList []QuotaGroup

// QuotaGroup represents a quota group
type QuotaGroup struct {
	// Name of the group
	Name string `json:"name,omitempty"`
	// Rules associated with the group
	Rules []QuotaRuleInfo `json:"rules"`
}

// CreateQutaGroupOptions represents the options for creating a quota group
type CreateQuotaGroupOptions struct {
	// Name of the quota group to create
	Name string `json:"name" binding:"Required"`
	// Rules to add to the newly created group.
	// If a rule does not exist, it will be created.
	Rules []CreateQuotaRuleOptions `json:"rules"`
}

// CreateQuotaRuleOptions represents the options for creating a quota rule
type CreateQuotaRuleOptions struct {
	// Name of the rule to create
	Name string `json:"name" binding:"Required"`
	// The limit set by the rule
	Limit *int64 `json:"limit"`
	// The subjects affected by the rule
	Subjects []string `json:"subjects"`
}

// EditQuotaRuleOptions represents the options for editing a quota rule
type EditQuotaRuleOptions struct {
	// The limit set by the rule
	Limit *int64 `json:"limit"`
	// The subjects affected by the rule
	Subjects *[]string `json:"subjects"`
}

// SetUserQuotaGroupsOptions represents the quota groups of a user
type SetUserQuotaGroupsOptions struct {
	// Quota groups the user shall have
	// required: true
	Groups *[]string `json:"groups"`
}

// QuotaUsedAttachmentList represents a list of attachment counting towards a user's quota
type QuotaUsedAttachmentList []*QuotaUsedAttachment

// QuotaUsedAttachment represents an attachment counting towards a user's quota
type QuotaUsedAttachment struct {
	// Filename of the attachment
	Name string `json:"name"`
	// Size of the attachment (in bytes)
	Size int64 `json:"size"`
	// API URL for the attachment
	APIURL string `json:"api_url"`
	// Context for the attachment: URLs to the containing object
	ContainedIn struct {
		// API URL for the object that contains this attachment
		APIURL string `json:"api_url"`
		// HTML URL for the object that contains this attachment
		HTMLURL string `json:"html_url"`
	} `json:"contained_in"`
}

// QuotaUsedPackageList represents a list of packages counting towards a user's quota
type QuotaUsedPackageList []*QuotaUsedPackage

// QuotaUsedPackage represents a package counting towards a user's quota
type QuotaUsedPackage struct {
	// Name of the package
	Name string `json:"name"`
	// Type of the package
	Type string `json:"type"`
	// Version of the package
	Version string `json:"version"`
	// Size of the package version
	Size int64 `json:"size"`
	// HTML URL to the package version
	HTMLURL string `json:"html_url"`
}

// QuotaUsedArtifactList represents a list of artifacts counting towards a user's quota
type QuotaUsedArtifactList []*QuotaUsedArtifact

// QuotaUsedArtifact represents an artifact counting towards a user's quota
type QuotaUsedArtifact struct {
	// Name of the artifact
	Name string `json:"name"`
	// Size of the artifact (compressed)
	Size int64 `json:"size"`
	// HTML URL to the action run containing the artifact
	HTMLURL string `json:"html_url"`
}
