// Copyright 2024 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package container

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/modules/test"
	"forgejo.org/modules/timeutil"
	container_service "forgejo.org/services/packages/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupSHA256(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&container_service.SHA256BatchSize, 1)()

	ctx := db.DefaultContext

	createContainer := func(t *testing.T, name, version, digest string, created timeutil.TimeStamp) {
		t.Helper()

		ownerID := int64(2001)

		p := packages.Package{
			OwnerID:   ownerID,
			LowerName: name,
			Type:      packages.TypeContainer,
		}
		_, err := db.GetEngine(ctx).Insert(&p)
		require.NoError(t, err)

		var metadata string
		if digest != "" {
			m := container_module.Metadata{
				Manifests: []*container_module.Manifest{
					{
						Digest: digest,
					},
				},
			}
			mt, err := json.Marshal(m)
			require.NoError(t, err)
			metadata = string(mt)
		}
		v := packages.PackageVersion{
			PackageID:    p.ID,
			LowerVersion: version,
			MetadataJSON: metadata,
			CreatedUnix:  created,
		}
		_, err = db.GetEngine(ctx).NoAutoTime().Insert(&v)
		require.NoError(t, err)
	}

	cleanupAndCheckLogs := func(t *testing.T, olderThan time.Duration, expected ...string) {
		t.Helper()
		logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
		logChecker.Filter(expected...)
		logChecker.StopMark(container_service.SHA256LogFinish)
		defer cleanup()

		require.NoError(t, CleanupExpiredData(ctx, olderThan))

		logFiltered, logStopped := logChecker.Check(5 * time.Second)
		assert.True(t, logStopped)
		filtered := make([]bool, 0, len(expected))
		for range expected {
			filtered = append(filtered, true)
		}
		assert.Equal(t, filtered, logFiltered, expected)
	}

	ancient := 1 * time.Hour

	t.Run("no packages, cleanup nothing", func(t *testing.T) {
		cleanupAndCheckLogs(t, ancient, "Nothing to cleanup")
	})

	orphan := "orphan"
	createdLongAgo := timeutil.TimeStamp(time.Now().Add(-(ancient * 2)).Unix())
	createdRecently := timeutil.TimeStamp(time.Now().Add(-(ancient / 2)).Unix())

	t.Run("an orphaned package created a long time ago is removed", func(t *testing.T) {
		createContainer(t, orphan, "sha256:"+orphan, "", createdLongAgo)
		cleanupAndCheckLogs(t, ancient, "Removing 1 entries from `package_version`")
		cleanupAndCheckLogs(t, ancient, "Nothing to cleanup")
	})

	t.Run("a newly created orphaned package is not cleaned up", func(t *testing.T) {
		createContainer(t, orphan, "sha256:"+orphan, "", createdRecently)
		cleanupAndCheckLogs(t, ancient, "1 out of 1 container image(s) are not deleted because they were created less than")
		cleanupAndCheckLogs(t, 0, "Removing 1 entries from `package_version`")
		cleanupAndCheckLogs(t, 0, "Nothing to cleanup")
	})

	t.Run("a referenced package is not removed", func(t *testing.T) {
		referenced := "referenced"
		digest := "sha256:" + referenced
		createContainer(t, referenced, digest, "", createdRecently)
		index := "index"
		createContainer(t, index, index, digest, createdRecently)
		cleanupAndCheckLogs(t, ancient, "Nothing to cleanup")
	})
}
