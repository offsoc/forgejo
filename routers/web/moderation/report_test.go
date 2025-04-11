// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"testing"

	"forgejo.org/models/db"
	abuse_report_model "forgejo.org/models/moderation"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/web"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/forms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_UserCannotReportAlreadyReported(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "moderation/new_abuse_report")
	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: false,
		ID:      2,
	})

	r := unittest.AssertExistsAndLoadBean(t, &abuse_report_model.AbuseReport{
		ContentID:   1,
		ContentType: abuse_report_model.ReportedContentTypeIssue,
		Status:      abuse_report_model.ReportStatusTypeOpen,
	})

	ctx.Doer = u

	form := forms.ReportAbuseForm{
		ContentID:     1,
		ContentType:   abuse_report_model.ReportedContentTypeIssue,
		AbuseCategory: abuse_report_model.AbuseCategoryTypeIllegalContent,
		Remarks:       "Test content",
	}

	web.SetForm(ctx, &form)
	CreatePost(ctx)
	assert.NotEmpty(t, ctx.Flash.ErrorMsg)

	_, err := db.GetEngine(ctx).Get(&r)

	require.NoError(t, err)
	assert.Equal(t, 1, r.ContentID)
	assert.Equal(t, abuse_report_model.AbuseCategoryTypeIllegalContent, r.Category)
}
