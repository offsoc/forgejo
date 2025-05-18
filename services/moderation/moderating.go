// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"forgejo.org/models/issues"
	"forgejo.org/models/moderation"
	"forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
)

// GetShadowCopyMap unmarshals the shadow copy raw value of the given abuse report and returns a list of <key, value> pairs
// (to be rendered when the report is reviewed by an admin).
// If the report does not have a shadow copy ID or the raw value is empty, returns nil.
// If the unmarshal fails a warning is added in the logs and returns nil.
func GetShadowCopyMap(ctx *context.Context, ard *moderation.AbuseReportDetailed) []moderation.ShadowCopyField {
	if ard.ShadowCopyID.Valid && len(ard.ShadowCopyRawValue) > 0 {
		var data moderation.ShadowCopyData

		switch ard.ContentType {
		case moderation.ReportedContentTypeUser:
			data = new(user.UserData)
		case moderation.ReportedContentTypeRepository:
			// TODO: implement ShadowCopyData.GetValueMap() for RepositoryData
			// data = new(repo.RepositoryData)
		case moderation.ReportedContentTypeIssue:
			// TODO: implement ShadowCopyData.GetValueMap() for IssueData
			// data = new(issues.IssueData)
		case moderation.ReportedContentTypeComment:
			data = new(issues.CommentData)
		}
		if err := json.Unmarshal([]byte(ard.ShadowCopyRawValue), &data); err != nil {
			log.Warn("Unmarshal failed for shadow copy #%d. %v", ard.ShadowCopyID.Int64, err)
			return nil
		}
		return data.GetValueMap()
	}
	return nil
}
