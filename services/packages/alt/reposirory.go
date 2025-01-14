// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package alt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	packages_model "code.gitea.io/gitea/models/packages"
	alt_model "code.gitea.io/gitea/models/packages/alt"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	packages_module "code.gitea.io/gitea/modules/packages"
	rpm_module "code.gitea.io/gitea/modules/packages/rpm"
	"code.gitea.io/gitea/modules/setting"
	packages_service "code.gitea.io/gitea/services/packages"

	"github.com/larzconwell/bzip2"
	"github.com/ulikunitz/xz"
)

// GetOrCreateRepositoryVersion gets or creates the internal repository package
// The RPM registry needs multiple metadata files which are stored in this package.
func GetOrCreateRepositoryVersion(ctx context.Context, ownerID int64) (*packages_model.PackageVersion, error) {
	return packages_service.GetOrCreateInternalPackageVersion(ctx, ownerID, packages_model.TypeAlt, rpm_module.RepositoryPackage, rpm_module.RepositoryVersion)
}

// BuildAllRepositoryFiles (re)builds all repository files for every available group
func BuildAllRepositoryFiles(ctx context.Context, ownerID int64) error {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return err
	}

	// 1. Delete all existing repository files
	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return err
	}

	for _, pf := range pfs {
		if err := packages_service.DeletePackageFile(ctx, pf); err != nil {
			return err
		}
	}

	// 2. (Re)Build repository files for existing packages
	groups, err := alt_model.GetGroups(ctx, ownerID)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if err := BuildSpecificRepositoryFiles(ctx, ownerID, group); err != nil {
			return fmt.Errorf("failed to build repository files [%s]: %w", group, err)
		}
	}

	return nil
}

type repoChecksum struct {
	Value string `xml:",chardata"`
	Type  string `xml:"type,attr"`
}

type repoLocation struct {
	Href string `xml:"href,attr"`
}

type repoData struct {
	Type         string       `xml:"type,attr"`
	Checksum     repoChecksum `xml:"checksum"`
	MD5Checksum  repoChecksum `xml:"md5checksum"`
	Blake2bHash  repoChecksum `xml:"blake2bHash"`
	OpenChecksum repoChecksum `xml:"open-checksum"`
	Location     repoLocation `xml:"location"`
	Timestamp    int64        `xml:"timestamp"`
	Size         int64        `xml:"size"`
	OpenSize     int64        `xml:"open-size"`
}

type packageData struct {
	Package         *packages_model.Package
	Version         *packages_model.PackageVersion
	Blob            *packages_model.PackageBlob
	VersionMetadata *rpm_module.VersionMetadata
	FileMetadata    *rpm_module.FileMetadata
}

type packageCache = map[*packages_model.PackageFile]*packageData

// BuildSpecificRepositoryFiles builds metadata files for the repository
func BuildSpecificRepositoryFiles(ctx context.Context, ownerID int64, group string) error {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return err
	}

	pfs, _, err := packages_model.SearchFiles(ctx, &packages_model.PackageFileSearchOptions{
		OwnerID:      ownerID,
		PackageType:  packages_model.TypeAlt,
		Query:        "%.rpm",
		CompositeKey: group,
	})
	if err != nil {
		return err
	}

	// Delete the repository files if there are no packages
	if len(pfs) == 0 {
		pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
		if err != nil {
			return err
		}
		for _, pf := range pfs {
			if err := packages_service.DeletePackageFile(ctx, pf); err != nil {
				return err
			}
		}

		return nil
	}

	// Cache data needed for all repository files
	cache := make(packageCache)
	for _, pf := range pfs {
		pv, err := packages_model.GetVersionByID(ctx, pf.VersionID)
		if err != nil {
			return err
		}
		p, err := packages_model.GetPackageByID(ctx, pv.PackageID)
		if err != nil {
			return err
		}
		pb, err := packages_model.GetBlobByID(ctx, pf.BlobID)
		if err != nil {
			return err
		}
		pps, err := packages_model.GetPropertiesByName(ctx, packages_model.PropertyTypeFile, pf.ID, rpm_module.PropertyMetadata)
		if err != nil {
			return err
		}

		pd := &packageData{
			Package: p,
			Version: pv,
			Blob:    pb,
		}

		if err := json.Unmarshal([]byte(pv.MetadataJSON), &pd.VersionMetadata); err != nil {
			return err
		}
		if len(pps) > 0 {
			if err := json.Unmarshal([]byte(pps[0].Value), &pd.FileMetadata); err != nil {
				return err
			}
		}

		cache[pf] = pd
	}

	pkglist, err := buildPackageLists(ctx, pv, pfs, cache, group)
	if err != nil {
		return err
	}

	err = buildRelease(ctx, pv, pfs, cache, group, pkglist)
	if err != nil {
		return err
	}

	return nil
}

type RPMHeader struct {
	Magic    [4]byte
	Reserved [4]byte
	NIndex   uint32
	HSize    uint32
}

type RPMHdrIndex struct {
	Tag    uint32
	Type   uint32
	Offset uint32
	Count  uint32
}

// https://refspecs.linuxbase.org/LSB_4.0.0/LSB-Core-generic/LSB-Core-generic/pkgformat.html
func buildPackageLists(ctx context.Context, pv *packages_model.PackageVersion, pfs []*packages_model.PackageFile, c packageCache, group string) (map[string][]any, error) {
	architectures := []string{}

	for _, pf := range pfs {
		pd := c[pf]

		if !slices.Contains(architectures, pd.FileMetadata.Architecture) {
			architectures = append(architectures, pd.FileMetadata.Architecture)
		}
	}

	repoDataListByArch := make(map[string][]any)
	repoDataList := []any{}
	orderedHeaders := []*RPMHeader{}

	for i := range architectures {
		headersWithIndexes := make(map[*RPMHeader]map[*RPMHdrIndex][]any)
		headersWithPtrs := make(map[*RPMHeader][]*RPMHdrIndex)
		indexPtrs := []*RPMHdrIndex{}
		indexes := make(map[*RPMHdrIndex][]any)

		for _, pf := range pfs {
			pd := c[pf]

			if pd.FileMetadata.Architecture == architectures[i] {
				var requireNames []any
				var requireVersions []any
				var requireFlags []any
				requireNamesSize := 0
				requireVersionsSize := 0
				requireFlagsSize := 0

				for _, entry := range pd.FileMetadata.Requires {
					if entry != nil {
						requireNames = append(requireNames, entry.Name)
						requireVersions = append(requireVersions, entry.Version)
						requireFlags = append(requireFlags, entry.AltFlags)
						requireNamesSize += len(entry.Name) + 1
						requireVersionsSize += len(entry.Version) + 1
						requireFlagsSize += 4
					}
				}

				var conflictNames []any
				var conflictVersions []any
				var conflictFlags []any
				conflictNamesSize := 0
				conflictVersionsSize := 0
				conflictFlagsSize := 0

				for _, entry := range pd.FileMetadata.Conflicts {
					if entry != nil {
						conflictNames = append(conflictNames, entry.Name)
						conflictVersions = append(conflictVersions, entry.Version)
						conflictFlags = append(conflictFlags, entry.AltFlags)
						conflictNamesSize += len(entry.Name) + 1
						conflictVersionsSize += len(entry.Version) + 1
						conflictFlagsSize += 4
					}
				}

				var baseNames []any
				var dirNames []any
				baseNamesSize := 0
				dirNamesSize := 0

				for _, entry := range pd.FileMetadata.Files {
					if entry != nil {
						re := regexp.MustCompile(`(.*?/)([^/]*)$`)
						matches := re.FindStringSubmatch(entry.Path)
						if len(matches) == 3 {
							baseNames = append(baseNames, matches[2])
							dirNames = append(dirNames, matches[1])
							baseNamesSize += len(matches[2]) + 1
							dirNamesSize += len(matches[1]) + 1
						}
					}
				}

				var provideNames []any
				var provideVersions []any
				var provideFlags []any
				provideNamesSize := 0
				provideVersionsSize := 0
				provideFlagsSize := 0

				for _, entry := range pd.FileMetadata.Provides {
					if entry != nil {
						provideNames = append(provideNames, entry.Name)
						provideVersions = append(provideVersions, entry.Version)
						provideFlags = append(provideFlags, entry.AltFlags)
						provideNamesSize += len(entry.Name) + 1
						provideVersionsSize += len(entry.Version) + 1
						provideFlagsSize += 4
					}
				}

				var obsoleteNames []any
				var obsoleteVersions []any
				var obsoleteFlags []any
				obsoleteNamesSize := 0
				obsoleteVersionsSize := 0
				obsoleteFlagsSize := 0

				for _, entry := range pd.FileMetadata.Obsoletes {
					if entry != nil {
						obsoleteNames = append(obsoleteNames, entry.Name)
						obsoleteVersions = append(obsoleteVersions, entry.Version)
						obsoleteFlags = append(obsoleteFlags, entry.AltFlags)
						obsoleteNamesSize += len(entry.Name) + 1
						obsoleteVersionsSize += len(entry.Version) + 1
						obsoleteFlagsSize += 4
					}
				}

				var changeLogTimes []any
				var changeLogNames []any
				var changeLogTexts []any
				changeLogTimesSize := 0
				changeLogNamesSize := 0
				changeLogTextsSize := 0

				for _, entry := range pd.FileMetadata.Changelogs {
					if entry != nil {
						changeLogNames = append(changeLogNames, entry.Author)
						changeLogTexts = append(changeLogTexts, entry.Text)
						changeLogTimes = append(changeLogTimes, uint32(int64(entry.Date)))
						changeLogNamesSize += len(entry.Author) + 1
						changeLogTextsSize += len(entry.Text) + 1
						changeLogTimesSize += 4
					}
				}

				/*Header*/
				hdr := &RPMHeader{
					Magic:    [4]byte{0x8E, 0xAD, 0xE8, 0x01},
					Reserved: [4]byte{0, 0, 0, 0},
					NIndex:   binary.BigEndian.Uint32([]byte{0, 0, 0, 0}),
					HSize:    binary.BigEndian.Uint32([]byte{0, 0, 0, 0}),
				}
				orderedHeaders = append(orderedHeaders, hdr)

				/*Tags: */

				nameInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 232}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: 0,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &nameInd)
				indexes[&nameInd] = append(indexes[&nameInd], pd.Package.Name)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.Package.Name) + 1)

				// Индекс для версии пакета
				versionInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 233}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &versionInd)
				indexes[&versionInd] = append(indexes[&versionInd], pd.FileMetadata.Version)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.FileMetadata.Version) + 1)

				summaryInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 236}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 9}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &summaryInd)
				indexes[&summaryInd] = append(indexes[&summaryInd], pd.VersionMetadata.Summary)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.VersionMetadata.Summary) + 1)

				descriptionInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 237}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 9}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &descriptionInd)
				indexes[&descriptionInd] = append(indexes[&descriptionInd], pd.VersionMetadata.Description)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.VersionMetadata.Description) + 1)

				releaseInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 234}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &releaseInd)
				indexes[&releaseInd] = append(indexes[&releaseInd], pd.FileMetadata.Release)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.FileMetadata.Release) + 1)

				alignPadding(hdr, indexes, &releaseInd)

				sizeInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 241}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 4}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &sizeInd)
				indexes[&sizeInd] = append(indexes[&sizeInd], int32(pd.FileMetadata.InstalledSize))
				hdr.NIndex++
				hdr.HSize += 4

				buildTimeInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 238}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 4}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &buildTimeInd)
				indexes[&buildTimeInd] = append(indexes[&buildTimeInd], int32(pd.FileMetadata.BuildTime))
				hdr.NIndex++
				hdr.HSize += 4

				licenseInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 246}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &licenseInd)
				indexes[&licenseInd] = append(indexes[&licenseInd], pd.VersionMetadata.License)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.VersionMetadata.License) + 1)

				packagerInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 247}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &packagerInd)
				indexes[&packagerInd] = append(indexes[&packagerInd], pd.FileMetadata.Packager)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.FileMetadata.Packager) + 1)

				groupInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 248}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &groupInd)
				indexes[&groupInd] = append(indexes[&groupInd], pd.FileMetadata.Group)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.FileMetadata.Group) + 1)

				urlInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 252}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &urlInd)
				indexes[&urlInd] = append(indexes[&urlInd], pd.VersionMetadata.ProjectURL)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.VersionMetadata.ProjectURL) + 1)

				if len(changeLogNames) != 0 && len(changeLogTexts) != 0 && len(changeLogTimes) != 0 {
					alignPadding(hdr, indexes, &urlInd)

					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x38}, []byte{0, 0, 0, 4}, changeLogTimes, changeLogTimesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x39}, []byte{0, 0, 0, 8}, changeLogNames, changeLogNamesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x3A}, []byte{0, 0, 0, 8}, changeLogTexts, changeLogTextsSize)
				}

				archInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 254}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &archInd)
				indexes[&archInd] = append(indexes[&archInd], pd.FileMetadata.Architecture)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.FileMetadata.Architecture) + 1)

				if len(provideNames) != 0 && len(provideVersions) != 0 && len(provideFlags) != 0 {
					alignPadding(hdr, indexes, &archInd)

					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x58}, []byte{0, 0, 0, 4}, provideFlags, provideFlagsSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x17}, []byte{0, 0, 0, 8}, provideNames, provideNamesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x59}, []byte{0, 0, 0, 8}, provideVersions, provideVersionsSize)
				}

				sourceRpmInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x14}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &sourceRpmInd)
				indexes[&sourceRpmInd] = append(indexes[&sourceRpmInd], pd.FileMetadata.SourceRpm)
				hdr.NIndex++
				hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(len(pd.FileMetadata.SourceRpm) + 1)})

				if len(requireNames) != 0 && len(requireVersions) != 0 && len(requireFlags) != 0 {
					alignPadding(hdr, indexes, &sourceRpmInd)

					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x18}, []byte{0, 0, 0, 4}, requireFlags, requireFlagsSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0, 0, 4, 25}, []byte{0, 0, 0, 8}, requireNames, requireNamesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x1A}, []byte{0, 0, 0, 8}, requireVersions, requireVersionsSize)
				}

				if len(baseNames) != 0 {
					baseNamesInd := RPMHdrIndex{
						Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x5D}),
						Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 8}),
						Offset: hdr.HSize,
						Count:  uint32(len(baseNames)),
					}
					indexPtrs = append(indexPtrs, &baseNamesInd)
					indexes[&baseNamesInd] = baseNames
					hdr.NIndex++
					hdr.HSize += uint32(baseNamesSize)
				}

				if len(dirNames) != 0 {
					dirnamesInd := RPMHdrIndex{
						Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x5E}),
						Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 8}),
						Offset: hdr.HSize,
						Count:  uint32(len(dirNames)),
					}
					indexPtrs = append(indexPtrs, &dirnamesInd)
					indexes[&dirnamesInd] = dirNames
					hdr.NIndex++
					hdr.HSize += uint32(dirNamesSize)
				}

				filenameInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x40}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &filenameInd)
				indexes[&filenameInd] = append(indexes[&filenameInd], pf.Name)
				hdr.NIndex++
				hdr.HSize += uint32(len(pf.Name) + 1)

				alignPadding(hdr, indexes, &filenameInd)

				filesizeInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x41}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 4}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &filesizeInd)
				indexes[&filesizeInd] = append(indexes[&filesizeInd], int32(pd.Blob.Size))
				hdr.NIndex++
				hdr.HSize += 4

				md5Ind := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x45}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &md5Ind)
				indexes[&md5Ind] = append(indexes[&md5Ind], pd.Blob.HashMD5)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.Blob.HashMD5) + 1)

				blake2bInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x49}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &blake2bInd)
				indexes[&blake2bInd] = append(indexes[&blake2bInd], pd.Blob.HashBlake2b)
				hdr.NIndex++
				hdr.HSize += uint32(len(pd.Blob.HashBlake2b) + 1)

				if len(conflictNames) != 0 && len(conflictVersions) != 0 && len(conflictFlags) != 0 {
					alignPadding(hdr, indexes, &blake2bInd)

					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x1D}, []byte{0, 0, 0, 4}, conflictFlags, conflictFlagsSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x1E}, []byte{0, 0, 0, 8}, conflictNames, conflictNamesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x1F}, []byte{0, 0, 0, 8}, conflictVersions, conflictVersionsSize)
				}

				directoryInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x4A}),
					Type:   binary.BigEndian.Uint32([]byte{0, 0, 0, 6}),
					Offset: hdr.HSize,
					Count:  1,
				}
				indexPtrs = append(indexPtrs, &directoryInd)
				indexes[&directoryInd] = append(indexes[&directoryInd], "RPMS.classic")
				hdr.NIndex++
				hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(len("RPMS.classic") + 1)})

				if len(obsoleteNames) != 0 && len(obsoleteVersions) != 0 && len(obsoleteFlags) != 0 {
					alignPadding(hdr, indexes, &directoryInd)

					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x5A}, []byte{0, 0, 0, 4}, obsoleteFlags, obsoleteFlagsSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x42}, []byte{0, 0, 0, 8}, obsoleteNames, obsoleteNamesSize)
					addRPMHdrIndex(hdr, &indexPtrs, indexes, []byte{0x00, 0x00, 0x04, 0x5B}, []byte{0, 0, 0, 8}, obsoleteVersions, obsoleteVersionsSize)
				}

				headersWithIndexes[hdr] = indexes
				headersWithPtrs[hdr] = indexPtrs

				indexPtrs = []*RPMHdrIndex{}
				indexes = make(map[*RPMHdrIndex][]any)
			}
		}

		files := []string{"pkglist.classic", "pkglist.classic.xz", "pkglist.classic.bz2"}
		for file := range files {
			fileInfo, err := addPkglistAsFileToRepo(ctx, pv, files[file], headersWithIndexes, headersWithPtrs, orderedHeaders, group, architectures[i])
			if err != nil {
				return nil, err
			}
			repoDataList = append(repoDataList, fileInfo)
			repoDataListByArch[architectures[i]] = repoDataList
		}
		repoDataList = []any{}
		orderedHeaders = []*RPMHeader{}
	}
	return repoDataListByArch, nil
}

func alignPadding(hdr *RPMHeader, indexes map[*RPMHdrIndex][]any, lastIndex *RPMHdrIndex) {
	/* Align to 4-bytes to add a 4-byte element. */
	padding := (4 - (hdr.HSize % 4)) % 4
	if padding == 4 {
		padding = 0
	}
	hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(padding)})

	for i := uint32(0); i < padding; i++ {
		for _, elem := range indexes[lastIndex] {
			if str, ok := elem.(string); ok {
				indexes[lastIndex][len(indexes[lastIndex])-1] = str + "\x00"
			}
		}
	}
}

func addRPMHdrIndex(hdr *RPMHeader, indexPtrs *[]*RPMHdrIndex, indexes map[*RPMHdrIndex][]any, tag, typeByte []byte, data []any, dataSize int) {
	index := RPMHdrIndex{
		Tag:    binary.BigEndian.Uint32(tag),
		Type:   binary.BigEndian.Uint32(typeByte),
		Offset: hdr.HSize,
		Count:  uint32(len(data)),
	}
	*indexPtrs = append(*indexPtrs, &index)
	indexes[&index] = data
	hdr.NIndex++
	hdr.HSize += uint32(dataSize)
}

// https://www.altlinux.org/APT_в_ALT_Linux/CreateRepository
func buildRelease(ctx context.Context, pv *packages_model.PackageVersion, pfs []*packages_model.PackageFile, c packageCache, group string, pkglist map[string][]any) error {
	var buf bytes.Buffer

	architectures := []string{}

	for _, pf := range pfs {
		pd := c[pf]
		if !slices.Contains(architectures, pd.FileMetadata.Architecture) {
			architectures = append(architectures, pd.FileMetadata.Architecture)
		}
	}

	for i := range architectures {
		archive := "Alt Linux Team"
		component := "classic"
		version := strconv.FormatInt(time.Now().Unix(), 10)
		architectures := architectures[i]
		origin := "Alt Linux Team"
		label := setting.AppName
		notautomatic := "false"
		data := fmt.Sprintf("Archive: %s\nComponent: %s\nVersion: %s\nOrigin: %s\nLabel: %s\nArchitecture: %s\nNotAutomatic: %s",
			archive, component, version, origin, label, architectures, notautomatic)
		buf.WriteString(data + "\n")
		fileInfo, err := addReleaseAsFileToRepo(ctx, pv, "release.classic", buf.String(), group, architectures)
		if err != nil {
			return err
		}
		buf.Reset()

		origin = setting.AppName
		suite := "Sisyphus"
		codename := strconv.FormatInt(time.Now().Unix(), 10)
		date := time.Now().UTC().Format(time.RFC1123)

		var md5Sum string
		var blake2b string

		for _, pkglistByArch := range pkglist[architectures] {
			md5Sum += fmt.Sprintf(" %s %s %s\n", pkglistByArch.([]string)[2], pkglistByArch.([]string)[4], "base/"+pkglistByArch.([]string)[0])
			blake2b += fmt.Sprintf(" %s %s %s\n", pkglistByArch.([]string)[3], pkglistByArch.([]string)[4], "base/"+pkglistByArch.([]string)[0])
		}
		md5Sum += fmt.Sprintf(" %s %s %s\n", fileInfo[2], fileInfo[4], "base/"+fileInfo[0])
		blake2b += fmt.Sprintf(" %s %s %s\n", fileInfo[3], fileInfo[4], "base/"+fileInfo[0])

		data = fmt.Sprintf("Origin: %s\nLabel: %s\nSuite: %s\nCodename: %s\nDate: %s\nArchitectures: %s\nMD5Sum:\n%sBLAKE2b:\n%s\n",
			origin, label, suite, codename, date, architectures, md5Sum, blake2b)
		buf.WriteString(data + "\n")
		_, err = addReleaseAsFileToRepo(ctx, pv, "release", buf.String(), group, architectures)
		if err != nil {
			return err
		}
		buf.Reset()
	}
	return nil
}

func addReleaseAsFileToRepo(ctx context.Context, pv *packages_model.PackageVersion, filename, obj, group, arch string) ([]string, error) {
	content, _ := packages_module.NewHashedBuffer()
	defer content.Close()

	h := sha256.New()

	w := io.MultiWriter(content, h)
	if _, err := w.Write([]byte(obj)); err != nil {
		return nil, err
	}

	_, err := packages_service.AddFileToPackageVersionInternal(
		ctx,
		pv,
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     filename,
				CompositeKey: arch + "__" + group,
			},
			Creator:           user_model.NewGhostUser(),
			Data:              content,
			IsLead:            false,
			OverwriteExisting: true,
		},
	)
	if err != nil {
		return nil, err
	}

	hashMD5, _, hashSHA256, _, hashBlake2b := content.Sums()

	if group == "" {
		group = "alt"
	}

	repoData := &repoData{
		Type: filename,
		Checksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(hashSHA256),
		},
		MD5Checksum: repoChecksum{
			Type:  "md5",
			Value: hex.EncodeToString(hashMD5),
		},
		OpenChecksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(h.Sum(nil)),
		},
		Blake2bHash: repoChecksum{
			Type:  "blake2b",
			Value: hex.EncodeToString(hashBlake2b),
		},
		Location: repoLocation{
			Href: group + ".repo/" + arch + "/base/" + filename,
		},
		Size: content.Size(),
		/* Unused values:
		Timestamp: time.Now().Unix(),
		OpenSize:  content.Size(), */
	}

	data := []string{
		repoData.Type, repoData.Checksum.Value,
		repoData.MD5Checksum.Value, repoData.Blake2bHash.Value, strconv.Itoa(int(repoData.Size)),
	}

	return data, nil
}

func addPkglistAsFileToRepo(ctx context.Context, pv *packages_model.PackageVersion, filename string, headersWithIndexes map[*RPMHeader]map[*RPMHdrIndex][]any, headersWithPtrs map[*RPMHeader][]*RPMHdrIndex, orderedHeaders []*RPMHeader, group, arch string) ([]string, error) {
	content, _ := packages_module.NewHashedBuffer()
	defer content.Close()

	h := sha256.New()
	w := io.MultiWriter(content, h)
	buf := &bytes.Buffer{}

	for _, hdr := range orderedHeaders {
		if err := binary.Write(buf, binary.BigEndian, hdr); err != nil {
			return nil, err
		}

		for _, indexPtr := range headersWithPtrs[hdr] {
			index := *indexPtr

			if err := binary.Write(buf, binary.BigEndian, index); err != nil {
				return nil, err
			}
		}

		for _, indexPtr := range headersWithPtrs[hdr] {
			for _, indexValue := range headersWithIndexes[hdr][indexPtr] {
				switch v := indexValue.(type) {
				case string:
					if _, err := buf.WriteString(v + "\x00"); err != nil {
						return nil, err
					}
				case int, int32, int64, uint32:
					if err := binary.Write(buf, binary.BigEndian, v); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	parts := strings.Split(filename, ".")

	if len(parts) == 3 && parts[len(parts)-1] == "xz" {
		xzContent, err := compressXZ(buf.Bytes())
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(xzContent); err != nil {
			return nil, err
		}
	} else if len(parts) == 3 && parts[len(parts)-1] == "bz2" {
		bz2Content, err := compressBZ2(buf.Bytes())
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(bz2Content); err != nil {
			return nil, err
		}
	} else {
		if _, err := w.Write(buf.Bytes()); err != nil {
			return nil, err
		}
	}

	_, err := packages_service.AddFileToPackageVersionInternal(
		ctx,
		pv,
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     filename,
				CompositeKey: arch + "__" + group,
			},
			Creator:           user_model.NewGhostUser(),
			Data:              content,
			IsLead:            false,
			OverwriteExisting: true,
		},
	)
	if err != nil {
		return nil, err
	}

	hashMD5, _, hashSHA256, _, hashBlake2b := content.Sums()

	if group == "" {
		group = "alt"
	}

	repoData := &repoData{
		Type: filename,
		Checksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(hashSHA256),
		},
		MD5Checksum: repoChecksum{
			Type:  "md5",
			Value: hex.EncodeToString(hashMD5),
		},
		OpenChecksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(h.Sum(nil)),
		},
		Blake2bHash: repoChecksum{
			Type:  "blake2b",
			Value: hex.EncodeToString(hashBlake2b),
		},
		Location: repoLocation{
			Href: group + ".repo/" + arch + "/base/" + filename,
		},
		Size: content.Size(),
		/* Unused values:
		Timestamp: time.Now().Unix(),
		OpenSize:  content.Size(), */
	}

	data := []string{
		repoData.Type, repoData.Checksum.Value,
		repoData.MD5Checksum.Value, repoData.Blake2bHash.Value, strconv.Itoa(int(repoData.Size)),
	}

	return data, nil
}

func compressXZ(data []byte) ([]byte, error) {
	var xzContent bytes.Buffer
	xzWriter, err := xz.NewWriter(&xzContent)
	if err != nil {
		return nil, err
	}
	defer xzWriter.Close()

	if _, err := xzWriter.Write(data); err != nil {
		return nil, err
	}
	if err := xzWriter.Close(); err != nil {
		return nil, err
	}

	return xzContent.Bytes(), nil
}

func compressBZ2(data []byte) ([]byte, error) {
	var bz2Content bytes.Buffer
	bz2Writer := bzip2.NewWriter(&bz2Content)
	defer bz2Writer.Close()

	if _, err := bz2Writer.Write(data); err != nil {
		return nil, err
	}
	if err := bz2Writer.Close(); err != nil {
		return nil, err
	}

	return bz2Content.Bytes(), nil
}
