// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package container

import (
	"context"
	"strings"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	container_module "code.gitea.io/gitea/modules/packages/container"
)

var (
	SHA256BatchSize = 500
	SHA256Log       = "cleanup dangling images with a sha256:* version"
	SHA256LogStart  = "Start to " + SHA256Log
	SHA256LogFinish = "Finished to " + SHA256Log
)

func CleanupSHA256(ctx context.Context, olderThan time.Duration) error {
	log.Info(SHA256LogStart)
	err := cleanupSHA256(ctx, olderThan)
	log.Info(SHA256LogFinish)
	return err
}

func cleanupSHA256(outerCtx context.Context, olderThan time.Duration) error {
	ctx, committer, err := db.TxContext(outerCtx)
	if err != nil {
		return err
	}
	defer committer.Close()

	foundAtLeastOneSHA256 := false
	shaToVersionID := make(map[string]int64, 100)
	knownSHA := make(map[string]any, 100)

	log.Debug("Look for all package_version.version that start with sha256:")

	old := time.Now().Add(-olderThan).Unix()

	// Iterate over all container versions in ascending order and store
	// in shaToVersionID all versions with a sha256: prefix. If an index
	// manifest is found, the sha256: digest it references are removed
	// from shaToVersionID. If the sha256: digest found in an index
	// manifest is not already in shaToVersionID, it is stored in
	// knownSHA to be dealt with later.
	//
	// Although it is theoretically possible that a sha256: is uploaded
	// after the index manifest that references it, this is not the
	// normal order of operations. First the sha256: version is uploaded
	// and then the index manifest. When the iteration completes,
	// knownSHA will therefore be empty most of the time and
	// shaToVersionID will only contain unreferenced sha256: versions.
	if err := db.GetEngine(ctx).
		Select("`package_version`.`id`, `package_version`.`lower_version`, `package_version`.`metadata_json`").
		Join("INNER", "`package`", "`package`.`id` = `package_version`.`package_id`").
		Where("`package`.`type` = ? AND `package_version`.`created_unix` < ?", packages.TypeContainer, old).
		OrderBy("`package_version`.`id` ASC").
		Iterate(new(packages.PackageVersion), func(_ int, bean any) error {
			v := bean.(*packages.PackageVersion)
			if strings.HasPrefix(v.LowerVersion, "sha256:") {
				shaToVersionID[v.LowerVersion] = v.ID
				foundAtLeastOneSHA256 = true
			} else if strings.Contains(v.MetadataJSON, `"manifests":[{`) {
				var metadata container_module.Metadata
				if err := json.Unmarshal([]byte(v.MetadataJSON), &metadata); err != nil {
					log.Error("package_version.id = %d package_version.metadata_json %s is not a JSON string containing valid metadata. It was ignored but it is an inconsistency in the database that should be looked at. %v", v.ID, v.MetadataJSON, err)
					return nil
				}
				for _, manifest := range metadata.Manifests {
					if _, ok := shaToVersionID[manifest.Digest]; ok {
						delete(shaToVersionID, manifest.Digest)
					} else {
						knownSHA[manifest.Digest] = true
					}
				}
			}
			return nil
		}); err != nil {
		return err
	}

	for sha := range knownSHA {
		delete(shaToVersionID, sha)
	}

	if len(shaToVersionID) == 0 {
		if foundAtLeastOneSHA256 {
			log.Debug("All container images with a version matching sha256:* are referenced by an index manifest")
		} else {
			log.Debug("There are no container images with a version matching sha256:*")
		}
		log.Info("Nothing to cleanup")
		return nil
	}

	found := len(shaToVersionID)

	log.Warn("%d container image(s) with a version matching sha256:* are not referenced by an index manifest", found)

	log.Debug("Deleting unreferenced image versions from `package_version`, `package_file` and `package_property` (%d at a time)", SHA256BatchSize)

	packageVersionIDs := make([]int64, 0, SHA256BatchSize)
	for _, id := range shaToVersionID {
		packageVersionIDs = append(packageVersionIDs, id)
	}

	for len(packageVersionIDs) > 0 {
		upper := min(len(packageVersionIDs), SHA256BatchSize)
		versionIDs := packageVersionIDs[0:upper]

		var packageFileIDs []int64
		if err := db.GetEngine(ctx).Select("id").Table("package_file").In("version_id", versionIDs).Find(&packageFileIDs); err != nil {
			return err
		}
		log.Info("Removing %d entries from `package_file` and `package_property`", len(packageFileIDs))
		if _, err := db.GetEngine(ctx).In("id", packageFileIDs).Delete(&packages.PackageFile{}); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).In("ref_id", packageFileIDs).And("ref_type = ?", packages.PropertyTypeFile).Delete(&packages.PackageProperty{}); err != nil {
			return err
		}

		log.Info("Removing %d entries from `package_version` and `package_property`", upper)
		if _, err := db.GetEngine(ctx).In("id", versionIDs).Delete(&packages.PackageVersion{}); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).In("ref_id", versionIDs).And("ref_type = ?", packages.PropertyTypeVersion).Delete(&packages.PackageProperty{}); err != nil {
			return err
		}

		packageVersionIDs = packageVersionIDs[upper:]
	}

	return committer.Commit()
}
