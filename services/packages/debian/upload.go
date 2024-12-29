// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package debian

import (
	"context"
	"fmt"
	"io"

	packages_model "code.gitea.io/gitea/models/packages"
	user_model "code.gitea.io/gitea/models/user"
	packages_module "code.gitea.io/gitea/modules/packages"
	debian_module "code.gitea.io/gitea/modules/packages/debian"
	"code.gitea.io/gitea/modules/util"
	packages_service "code.gitea.io/gitea/services/packages"
)

func UploadPackage(ctx context.Context, distribution, component string, reader io.Reader, owner, doer *user_model.User) (*packages_model.PackageVersion, error) {
	if distribution == "" || component == "" {
		return nil, util.ErrInvalidArgument
	}

	buf, err := packages_module.CreateHashedBufferFromReader(reader)
	if err != nil {
		return nil, err
	}
	defer buf.Close()

	pck, err := debian_module.ParsePackage(buf)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	pv, _, err := packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeDebian,
				Name:        pck.Name,
				Version:     pck.Version,
			},
			Creator:  doer,
			Metadata: pck.Metadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     fmt.Sprintf("%s_%s_%s.deb", pck.Name, pck.Version, pck.Architecture),
				CompositeKey: fmt.Sprintf("%s|%s", distribution, component),
			},
			Creator: doer,
			Data:    buf,
			IsLead:  true,
			Properties: map[string]string{
				debian_module.PropertyDistribution: distribution,
				debian_module.PropertyComponent:    component,
				debian_module.PropertyArchitecture: pck.Architecture,
				debian_module.PropertyControl:      pck.Control,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if err := BuildSpecificRepositoryFiles(ctx, owner.ID, distribution, component, pck.Architecture); err != nil {
		return nil, err
	}

	return pv, err
}
