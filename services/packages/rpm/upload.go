// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package rpm

import (
	"context"
	"fmt"
	"io"

	packages_model "code.gitea.io/gitea/models/packages"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	packages_module "code.gitea.io/gitea/modules/packages"
	rpm_module "code.gitea.io/gitea/modules/packages/rpm"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	packages_service "code.gitea.io/gitea/services/packages"
)

func UploadPackage(ctx context.Context, sign bool, group string, reader io.Reader, owner, doer *user_model.User) (*packages_model.PackageVersion, error) {
	buf, err := packages_module.CreateHashedBufferFromReader(reader)
	if err != nil {
		return nil, err
	}
	defer buf.Close()
	// if rpm sign enabled
	if setting.Packages.DefaultRPMSignEnabled || sign {
		pri, _, err := GetOrCreateKeyPair(ctx, owner.ID)
		if err != nil {
			return nil, err
		}
		signedBuf, err := NewSignedRPMBuffer(buf, pri)
		if err != nil {
			// Not in rpm format, parsing failed.
			return nil, util.NewInvalidArgumentErrorf("invalid format")
		}
		defer signedBuf.Close()
		buf = signedBuf
	}

	pck, err := rpm_module.ParsePackage(buf)
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

	pv, _, err := packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeRpm,
				Name:        pck.Name,
				Version:     pck.Version,
			},
			Creator:  doer,
			Metadata: pck.VersionMetadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     fmt.Sprintf("%s-%s.%s.rpm", pck.Name, pck.Version, pck.FileMetadata.Architecture),
				CompositeKey: group,
			},
			Creator: doer,
			Data:    buf,
			IsLead:  true,
			Properties: map[string]string{
				rpm_module.PropertyGroup:        group,
				rpm_module.PropertyArchitecture: pck.FileMetadata.Architecture,
				rpm_module.PropertyMetadata:     string(fileMetadataRaw),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if err := BuildSpecificRepositoryFiles(ctx, owner.ID, group); err != nil {
		return nil, err
	}

	return pv, err
}
