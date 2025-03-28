// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestShadowPassword(t *testing.T) {
	kases := []struct {
		Provider string
		CfgItem  string
		Result   string
	}{
		{
			Provider: "redis",
			CfgItem:  "network=tcp,addr=:6379,password=gitea,db=0,pool_size=100,idle_timeout=180",
			Result:   "network=tcp,addr=:6379,password=******,db=0,pool_size=100,idle_timeout=180",
		},
		{
			Provider: "mysql",
			CfgItem:  "root:@tcp(localhost:3306)/gitea?charset=utf8",
			Result:   "root:******@tcp(localhost:3306)/gitea?charset=utf8",
		},
		{
			Provider: "mysql",
			CfgItem:  "/gitea?charset=utf8",
			Result:   "/gitea?charset=utf8",
		},
		{
			Provider: "mysql",
			CfgItem:  "user:mypassword@/dbname",
			Result:   "user:******@/dbname",
		},
		{
			Provider: "postgres",
			CfgItem:  "user=pqgotest dbname=pqgotest sslmode=verify-full",
			Result:   "user=pqgotest dbname=pqgotest sslmode=verify-full",
		},
		{
			Provider: "postgres",
			CfgItem:  "user=pqgotest password= dbname=pqgotest sslmode=verify-full",
			Result:   "user=pqgotest password=****** dbname=pqgotest sslmode=verify-full",
		},
		{
			Provider: "postgres",
			CfgItem:  "postgres://user:pass@hostname/dbname",
			Result:   "postgres://user:******@hostname/dbname",
		},
		{
			Provider: "couchbase",
			CfgItem:  "http://dev-couchbase.example.com:8091/",
			Result:   "http://dev-couchbase.example.com:8091/",
		},
		{
			Provider: "couchbase",
			CfgItem:  "http://user:the_password@dev-couchbase.example.com:8091/",
			Result:   "http://user:******@dev-couchbase.example.com:8091/",
		},
	}

	for _, k := range kases {
		assert.Equal(t, k.Result, shadowPassword(k.Provider, k.CfgItem))
	}
}

func TestMonitorStats(t *testing.T) {
	unittest.PrepareTestEnv(t)

	t.Run("Normal", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Metrics.EnabledIssueByLabel, false)()
		defer test.MockVariableValue(&setting.Metrics.EnabledIssueByRepository, false)()

		ctx, _ := contexttest.MockContext(t, "admin/stats")
		MonitorStats(ctx)

		// Test some of the stats manually.
		mappedStats := ctx.Data["Stats"].(map[string]any)
		stats := activities_model.GetStatistic(ctx).Counter

		assert.EqualValues(t, stats.Comment, mappedStats["Comment"])
		assert.EqualValues(t, stats.Issue, mappedStats["Issue"])
		assert.EqualValues(t, stats.User, mappedStats["User"])
		assert.EqualValues(t, stats.Milestone, mappedStats["Milestone"])

		// Ensure that these aren't set.
		assert.Empty(t, stats.IssueByLabel)
		assert.Empty(t, stats.IssueByRepository)
		assert.Nil(t, mappedStats["IssueByLabel"])
		assert.Nil(t, mappedStats["IssueByRepository"])
	})

	t.Run("IssueByX", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Metrics.EnabledIssueByLabel, true)()
		defer test.MockVariableValue(&setting.Metrics.EnabledIssueByRepository, true)()

		ctx, _ := contexttest.MockContext(t, "admin/stats")
		MonitorStats(ctx)

		mappedStats := ctx.Data["Stats"].(map[string]any)
		stats := activities_model.GetStatistic(ctx).Counter

		assert.NotEmpty(t, stats.IssueByLabel)
		assert.NotEmpty(t, stats.IssueByRepository)
		assert.EqualValues(t, stats.IssueByLabel, mappedStats["IssueByLabel"])
		assert.EqualValues(t, stats.IssueByRepository, mappedStats["IssueByRepository"])
	})
}
