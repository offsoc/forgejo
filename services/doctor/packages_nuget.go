// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package doctor

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	nuget_module "forgejo.org/modules/packages/nuget"
	packages_service "forgejo.org/services/packages"

	"xorm.io/builder"
)

func init() {
	Register(&Check{
		Title:       "Extract Nuget Nuspec Files to content store",
		Name:        "packages-nuget-nuspec",
		IsDefault:   false,
		Run:         PackagesNugetNuspecCheck,
		Priority:    15,
		InitStorage: true,
	})
}

func PackagesNugetNuspecCheck(ctx context.Context, logger log.Logger, autofix bool) error {
	found := 0
	fixed := 0
	errors := 0

	err := db.Iterate(ctx, builder.Eq{"package.type": packages.TypeNuGet, "package.is_internal": false}, func(ctx context.Context, pkg *packages.Package) error {
		logger.Info("Processing package %s", pkg.Name)

		pvs, _, err := packages.SearchVersions(ctx, &packages.PackageSearchOptions{
			Type:      packages.TypeNuGet,
			PackageID: pkg.ID,
		})
		if err != nil {
			// Should never happen
			logger.Error("Failed to search for versions for package %s: %v", pkg.Name, err)
			return err
		}

		logger.Info("Found %d versions for package %s", len(pvs), pkg.Name)

		for _, pv := range pvs {
			pfs, err := packages.GetFilesByVersionID(ctx, pv.ID)
			if err != nil {
				logger.Error("Failed to get files for package version %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}

			if slices.ContainsFunc(pfs, func(pf *packages.PackageFile) bool { return strings.HasSuffix(pf.LowerName, ".nuspec") }) {
				logger.Debug("Nuspec file already exists for %s %s", pkg.Name, pv.Version)
				continue
			}

			nupkgIdx := slices.IndexFunc(pfs, func(pf *packages.PackageFile) bool { return pf.IsLead })

			if nupkgIdx < 0 {
				logger.Error("Missing nupkg file for %s %s", pkg.Name, pv.Version)
				errors++
				continue
			}

			pf := pfs[nupkgIdx]

			logger.Warn("Missing nuspec file found for %s %s", pkg.Name, pv.Version)
			found++

			if !autofix {
				continue
			}

			s, _, _, err := packages_service.GetPackageFileStream(ctx, pf)
			if err != nil {
				logger.Error("Failed to get nupkg file stream for %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}
			defer s.Close()

			buf, err := packages_module.CreateHashedBufferFromReader(s)
			if err != nil {
				logger.Error("Failed to create hashed buffer for nupkg from reader for %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}
			defer buf.Close()

			np, err := nuget_module.ParsePackageMetaData(buf, buf.Size())
			if err != nil {
				logger.Error("Failed to parse package metadata for %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}

			nuspecBuf, err := packages_module.CreateHashedBufferFromReaderWithSize(np.NuspecContent, np.NuspecContent.Len())
			if err != nil {
				logger.Error("Failed to create hashed buffer for nuspec from reader for %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}
			defer nuspecBuf.Close()

			_, err = packages_service.AddFileToPackageVersionInternal(
				ctx,
				pv,
				&packages_service.PackageFileCreationInfo{
					PackageFileInfo: packages_service.PackageFileInfo{
						Filename: fmt.Sprintf("%s.nuspec", pkg.LowerName),
					},
					Data:   nuspecBuf,
					IsLead: false,
				},
			)
			if err != nil {
				logger.Error("Failed to add nuspec file for %s %s: %v", pkg.Name, pv.Version, err)
				errors++
				continue
			}

			fixed++
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to iterate over users: %v", err)
		return err
	}

	if autofix {
		if fixed > 0 {
			logger.Info("Fixed %d package versions by extracting nuspec files", fixed)
		} else {
			logger.Info("No package versions with missing nuspec files found")
		}
	} else {
		if found > 0 {
			logger.Info("Found %d package versions with missing nuspec files", found)
		} else {
			logger.Info("No package versions with missing nuspec files found")
		}
	}

	if errors > 0 {
		return fmt.Errorf("failed to fix %d nuspec files", errors)
	}

	return nil
}
