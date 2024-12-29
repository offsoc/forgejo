// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package alpine

import (
	"context"
	"fmt"
	"io"

	packages_model "code.gitea.io/gitea/models/packages"
	alpine_model "code.gitea.io/gitea/models/packages/alpine"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	packages_module "code.gitea.io/gitea/modules/packages"
	alpine_module "code.gitea.io/gitea/modules/packages/alpine"
	packages_service "code.gitea.io/gitea/services/packages"
)

func createOrAddToExisting(ctx context.Context, pck *alpine_module.Package, branch, repository, architecture string, buf packages_module.HashedSizeReader, fileMetadataRaw []byte, owner, doer *user_model.User) (*packages_model.PackageVersion, error) {
	pv, _, err := packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeAlpine,
				Name:        pck.Name,
				Version:     pck.Version,
			},
			Creator:  doer,
			Metadata: pck.VersionMetadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     fmt.Sprintf("%s-%s.apk", pck.Name, pck.Version),
				CompositeKey: fmt.Sprintf("%s|%s|%s", branch, repository, architecture),
			},
			Creator: doer,
			Data:    buf,
			IsLead:  true,
			Properties: map[string]string{
				alpine_module.PropertyBranch:       branch,
				alpine_module.PropertyRepository:   repository,
				alpine_module.PropertyArchitecture: architecture,
				alpine_module.PropertyMetadata:     string(fileMetadataRaw),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if err := BuildSpecificRepositoryFiles(ctx, owner.ID, branch, repository, pck.FileMetadata.Architecture); err != nil {
		return nil, err
	}

	return pv, nil
}

func UploadPackage(ctx context.Context, branch, repository string, reader io.Reader, owner, doer *user_model.User) (*packages_model.PackageVersion, error) {
	buf, err := packages_module.CreateHashedBufferFromReader(reader)
	if err != nil {
		return nil, err
	}
	defer buf.Close()

	pck, err := alpine_module.ParsePackage(buf)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	fileMetadataRaw, err := json.Marshal(pck.FileMetadata)
	if err != nil {
		return nil, err
	}

	// Check whether the package being uploaded has no architecture defined.
	// If true, loop through the available architectures in the repo and create
	// the package file for the each architecture. If there are no architectures
	// available on the repository, fallback to x86_64
	if pck.FileMetadata.Architecture == "noarch" {
		architectures, err := alpine_model.GetArchitectures(ctx, owner.ID, repository)
		if err != nil {
			return nil, err
		}

		if len(architectures) == 0 {
			architectures = []string{
				"x86_64",
			}
		}

		var pv *packages_model.PackageVersion
		for _, arch := range architectures {
			pck.FileMetadata.Architecture = arch

			fileMetadataRaw, err := json.Marshal(pck.FileMetadata)
			if err != nil {
				return nil, err
			}

			pv, err = createOrAddToExisting(ctx, pck, branch, repository, pck.FileMetadata.Architecture, buf, fileMetadataRaw, owner, doer)
			if err != nil {
				return nil, err
			}
		}

		return pv, nil
	} else {
		return createOrAddToExisting(ctx, pck, branch, repository, pck.FileMetadata.Architecture, buf, fileMetadataRaw, owner, doer)
	}
}
