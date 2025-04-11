// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestAlreadyReported(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	reported := unittest.AssertExistsAndLoadBean(t, &AbuseReport{
		ID:          6,
		Status:      ReportStatusTypeOpen,
		ContentType: ReportedContentTypeIssue,
		ReporterID:  1,
		Category:    AbuseCategoryTypeSpam,
		CreatedUnix: timeutil.TimeStampNow(),
	})
	assert.True(t, AlreadyReported(db.DefaultContext, reported.ContentType, reported.ID))
}
