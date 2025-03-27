// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
)

// QuotaInfo
// swagger:response QuotaInfo
type swaggerResponseQuotaInfo struct {
	// in:body
	Body api.QuotaInfo `json:"body"`
}

// QuotaRuleInfoList
// swagger:response QuotaRuleInfoList
type swaggerResponseQuotaRuleInfoList struct {
	// in:body
	Body []api.QuotaRuleInfo `json:"body"`
}

// QuotaRuleInfo
// swagger:response QuotaRuleInfo
type swaggerResponseQuotaRuleInfo struct {
	// in:body
	Body api.QuotaRuleInfo `json:"body"`
}

// QuotaUsedAttachmentList
// swagger:response QuotaUsedAttachmentList
type swaggerQuotaUsedAttachmentList struct {
	// in:body
	Body api.QuotaUsedAttachmentList `json:"body"`
}

// QuotaUsedPackageList
// swagger:response QuotaUsedPackageList
type swaggerQuotaUsedPackageList struct {
	// in:body
	Body api.QuotaUsedPackageList `json:"body"`
}

// QuotaUsedArtifactList
// swagger:response QuotaUsedArtifactList
type swaggerQuotaUsedArtifactList struct {
	// in:body
	Body api.QuotaUsedArtifactList `json:"body"`
}

// QuotaGroup
// swagger:response QuotaGroup
type swaggerResponseQuotaGroup struct {
	// in:body
	Body api.QuotaGroup `json:"body"`
}

// QuotaGroupList
// swagger:response QuotaGroupList
type swaggerResponseQuotaGroupList struct {
	// in:body
	Body api.QuotaGroupList `json:"body"`
}
