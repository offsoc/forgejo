// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generic

import (
	"context"
	"io"
	"regexp"
	"strings"
	"unicode"

	packages_model "code.gitea.io/gitea/models/packages"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	packages_module "code.gitea.io/gitea/modules/packages"
	generic_packages_module "code.gitea.io/gitea/modules/packages/generic"
	packages_service "code.gitea.io/gitea/services/packages"
)

var (
	packageNameRegex = regexp.MustCompile(`\A[-_+.\w]+\z`)
	filenameRegex    = regexp.MustCompile(`\A[-_+=:;.()\[\]{}~!@#$%^& \w]+\z`)
)

func IsValidPackageName(packageName string) bool {
	if len(packageName) == 1 && !unicode.IsLetter(rune(packageName[0])) && !unicode.IsNumber(rune(packageName[0])) {
		return false
	}
	return packageNameRegex.MatchString(packageName) && packageName != ".."
}

func IsValidFileName(filename string) bool {
	return filenameRegex.MatchString(filename) &&
		strings.TrimSpace(filename) == filename &&
		filename != "." && filename != ".."
}

func UploadPackage(ctx context.Context, packageName, filename, packageVersion string, reader io.ReadCloser, owner, doer *user_model.User) (*packages_model.PackageVersion, error) {
	if !IsValidPackageName(packageName) {
		return nil, generic_packages_module.ErrInvalidName
	}

	if !IsValidFileName(filename) {
		return nil, generic_packages_module.ErrInvalidFilename
	}

	if packageVersion != strings.TrimSpace(packageVersion) {
		return nil, generic_packages_module.ErrInvalidVersion
	}

	buf, err := packages_module.CreateHashedBufferFromReader(reader)
	if err != nil {
		log.Error("Error creating hashed buffer: %v", err)
		return nil, err
	}
	defer buf.Close()

	pv, _, err := packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeGeneric,
				Name:        packageName,
				Version:     packageVersion,
			},
			Creator: doer,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: filename,
			},
			Creator: doer,
			Data:    buf,
			IsLead:  true,
		},
	)
	if err != nil {
		return nil, err
	}

	return pv, nil
}
