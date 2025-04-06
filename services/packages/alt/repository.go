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
	"path"
	"time"

	packages_model "forgejo.org/models/packages"
	alt_model "forgejo.org/models/packages/alt"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/json"
	packages_module "forgejo.org/modules/packages"
	rpm_module "forgejo.org/modules/packages/rpm"
	"forgejo.org/modules/setting"
	packages_service "forgejo.org/services/packages"

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

type indexWithData struct {
	index *RPMHdrIndex
	data  []any
}

type headerWithIndexes struct {
	header  *RPMHeader
	indexes []indexWithData
}

// https://refspecs.linuxbase.org/LSB_4.0.0/LSB-Core-generic/LSB-Core-generic/pkgformat.html
func buildPackageLists(ctx context.Context, pv *packages_model.PackageVersion, pfs []*packages_model.PackageFile, c packageCache, group string) (map[string][]*repoData, error) {
	packagesByArch := map[string][]*packages_model.PackageFile{}

	for _, pf := range pfs {
		pd := c[pf]

		packageArch := pd.FileMetadata.Architecture
		if packages, ok := packagesByArch[packageArch]; ok {
			packagesByArch[packageArch] = append(packages, pf)
		} else {
			packagesByArch[packageArch] = []*packages_model.PackageFile{pf}
		}
	}

	repoDataListByArch := make(map[string][]*repoData)

	for architecture, pfs := range packagesByArch {
		repoDataList := []*repoData{}
		orderedHeaders := []headerWithIndexes{}

		for _, pf := range pfs {
			pd := c[pf]

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
					dir, file := path.Split(entry.Path)

					baseNames = append(baseNames, file)
					dirNames = append(dirNames, dir)
					baseNamesSize += len(file) + 1
					dirNamesSize += len(dir) + 1
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
			orderedHeader := headerWithIndexes{hdr, []indexWithData{}}

			/*Tags: */
			nameInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 232}),
				Type:   6,
				Offset: 0,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &nameInd,
				data:  []any{pd.Package.Name},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.Package.Name) + 1)

			// Индекс для версии пакета
			versionInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 233}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &versionInd,
				data:  []any{pd.FileMetadata.Version},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.FileMetadata.Version) + 1)

			summaryInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 236}),
				Type:   9,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &summaryInd,
				data:  []any{pd.VersionMetadata.Summary},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.VersionMetadata.Summary) + 1)

			descriptionInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 237}),
				Type:   9,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &descriptionInd,
				data:  []any{pd.VersionMetadata.Description},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.VersionMetadata.Description) + 1)

			releaseInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 234}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &releaseInd,
				data:  []any{pd.FileMetadata.Release},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.FileMetadata.Release) + 1)

			alignPadding(hdr, orderedHeader.indexes)

			sizeInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 241}),
				Type:   4,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &sizeInd,
				data:  []any{int32(pd.FileMetadata.InstalledSize)},
			})
			hdr.NIndex++
			hdr.HSize += 4

			buildTimeInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 238}),
				Type:   4,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &buildTimeInd,
				data:  []any{int32(pd.FileMetadata.BuildTime)},
			})
			hdr.NIndex++
			hdr.HSize += 4

			licenseInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 246}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &licenseInd,
				data:  []any{pd.VersionMetadata.License},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.VersionMetadata.License) + 1)

			packagerInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 247}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &packagerInd,
				data:  []any{pd.FileMetadata.Packager},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.FileMetadata.Packager) + 1)

			groupInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 248}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &groupInd,
				data:  []any{pd.FileMetadata.Group},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.FileMetadata.Group) + 1)

			urlInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 252}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &urlInd,
				data:  []any{pd.VersionMetadata.ProjectURL},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.VersionMetadata.ProjectURL) + 1)

			if len(changeLogNames) != 0 && len(changeLogTexts) != 0 && len(changeLogTimes) != 0 {
				alignPadding(hdr, orderedHeader.indexes)

				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x38}, 4, changeLogTimes, changeLogTimesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x39}, 8, changeLogNames, changeLogNamesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x3A}, 8, changeLogTexts, changeLogTextsSize)
			}

			archInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0, 0, 3, 254}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &archInd,
				data:  []any{pd.FileMetadata.Architecture},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.FileMetadata.Architecture) + 1)

			if len(provideNames) != 0 && len(provideVersions) != 0 && len(provideFlags) != 0 {
				alignPadding(hdr, orderedHeader.indexes)

				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x58}, 4, provideFlags, provideFlagsSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x17}, 8, provideNames, provideNamesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x59}, 8, provideVersions, provideVersionsSize)
			}

			sourceRpmInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x14}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &sourceRpmInd,
				data:  []any{pd.FileMetadata.SourceRpm},
			})
			hdr.NIndex++
			hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(len(pd.FileMetadata.SourceRpm) + 1)})

			if len(requireNames) != 0 && len(requireVersions) != 0 && len(requireFlags) != 0 {
				alignPadding(hdr, orderedHeader.indexes)

				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x18}, 4, requireFlags, requireFlagsSize)
				addRPMHdrIndex(&orderedHeader, []byte{0, 0, 4, 25}, 8, requireNames, requireNamesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x1A}, 8, requireVersions, requireVersionsSize)
			}

			if len(baseNames) != 0 {
				baseNamesInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x5D}),
					Type:   8,
					Offset: hdr.HSize,
					Count:  uint32(len(baseNames)),
				}
				orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
					index: &baseNamesInd,
					data:  baseNames,
				})
				hdr.NIndex++
				hdr.HSize += uint32(baseNamesSize)
			}

			if len(dirNames) != 0 {
				dirnamesInd := RPMHdrIndex{
					Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x04, 0x5E}),
					Type:   8,
					Offset: hdr.HSize,
					Count:  uint32(len(dirNames)),
				}
				orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
					index: &dirnamesInd,
					data:  dirNames,
				})
				hdr.NIndex++
				hdr.HSize += uint32(dirNamesSize)
			}

			filenameInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x40}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &filenameInd,
				data:  []any{pf.Name},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pf.Name) + 1)

			alignPadding(hdr, orderedHeader.indexes)

			filesizeInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x41}),
				Type:   4,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &filesizeInd,
				data:  []any{int32(pd.Blob.Size)},
			})
			hdr.NIndex++
			hdr.HSize += 4

			md5Ind := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x45}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &md5Ind,
				data:  []any{pd.Blob.HashMD5},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.Blob.HashMD5) + 1)

			blake2bInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x49}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &blake2bInd,
				data:  []any{pd.Blob.HashBlake2b},
			})
			hdr.NIndex++
			hdr.HSize += uint32(len(pd.Blob.HashBlake2b) + 1)

			if len(conflictNames) != 0 && len(conflictVersions) != 0 && len(conflictFlags) != 0 {
				alignPadding(hdr, orderedHeader.indexes)

				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x1D}, 4, conflictFlags, conflictFlagsSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x1E}, 8, conflictNames, conflictNamesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x1F}, 8, conflictVersions, conflictVersionsSize)
			}

			directoryInd := RPMHdrIndex{
				Tag:    binary.BigEndian.Uint32([]byte{0x00, 0x0F, 0x42, 0x4A}),
				Type:   6,
				Offset: hdr.HSize,
				Count:  1,
			}
			orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
				index: &directoryInd,
				data:  []any{"RPMS.classic"},
			})
			hdr.NIndex++
			hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(len("RPMS.classic") + 1)})

			if len(obsoleteNames) != 0 && len(obsoleteVersions) != 0 && len(obsoleteFlags) != 0 {
				alignPadding(hdr, orderedHeader.indexes)

				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x5A}, 4, obsoleteFlags, obsoleteFlagsSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x42}, 8, obsoleteNames, obsoleteNamesSize)
				addRPMHdrIndex(&orderedHeader, []byte{0x00, 0x00, 0x04, 0x5B}, 8, obsoleteVersions, obsoleteVersionsSize)
			}

			orderedHeaders = append(orderedHeaders, orderedHeader)
		}

		files := []string{"pkglist.classic", "pkglist.classic.xz"}
		for file := range files {
			fileInfo, err := addPkglistAsFileToRepo(ctx, pv, files[file], orderedHeaders, group, architecture)
			if err != nil {
				return nil, err
			}
			repoDataList = append(repoDataList, fileInfo)
		}
		repoDataListByArch[architecture] = repoDataList
	}
	return repoDataListByArch, nil
}

func alignPadding(hdr *RPMHeader, indexes []indexWithData) {
	/* Align to 4-bytes to add a 4-byte element. */
	padding := (4 - (hdr.HSize % 4)) % 4
	if padding == 4 {
		padding = 0
	}
	hdr.HSize += binary.BigEndian.Uint32([]byte{0, 0, 0, uint8(padding)})

	lastIndex := len(indexes) - 1
	for i := uint32(0); i < padding; i++ {
		for _, elem := range indexes[lastIndex].data {
			if str, ok := elem.(string); ok {
				indexes[lastIndex].data[len(indexes[lastIndex].data)-1] = str + "\x00"
			}
		}
	}
}

func addRPMHdrIndex(orderedHeader *headerWithIndexes, tag []byte, typeVal uint32, data []any, dataSize int) {
	index := RPMHdrIndex{
		Tag:    binary.BigEndian.Uint32(tag),
		Type:   typeVal,
		Offset: orderedHeader.header.HSize,
		Count:  uint32(len(data)),
	}
	orderedHeader.indexes = append(orderedHeader.indexes, indexWithData{
		index: &index,
		data:  data,
	})
	orderedHeader.header.NIndex++
	orderedHeader.header.HSize += uint32(dataSize)
}

// https://www.altlinux.org/APT_в_ALT_Linux/CreateRepository
func buildRelease(ctx context.Context, pv *packages_model.PackageVersion, pfs []*packages_model.PackageFile, c packageCache, group string, pkglist map[string][]*repoData) error {
	architectures := make(container.Set[string])

	for _, pf := range pfs {
		pd := c[pf]
		architectures.Add(pd.FileMetadata.Architecture)
	}

	for architecture := range architectures.Seq() {
		version := time.Now().Unix()
		label := setting.AppName
		data := fmt.Sprintf(`Archive: Alt Linux Team
Component: classic
Version: %d
Origin: Alt Linux Team
Label: %s
Architecture: %s
NotAutomatic: false
`,
			version, label, architecture)
		fileInfo, err := addReleaseAsFileToRepo(ctx, pv, "release.classic", data, group, architecture)
		if err != nil {
			return err
		}

		origin := setting.AppName
		codename := time.Now().Unix()
		date := time.Now().UTC().Format(time.RFC1123)

		var md5Sum string
		var blake2b string

		for _, pkglistByArch := range pkglist[architecture] {
			md5Sum += fmt.Sprintf(" %s %d %s\n", pkglistByArch.MD5Checksum.Value, pkglistByArch.Size, "base/"+pkglistByArch.Type)
			blake2b += fmt.Sprintf(" %s %d %s\n", pkglistByArch.Blake2bHash.Value, pkglistByArch.Size, "base/"+pkglistByArch.Type)
		}
		md5Sum += fmt.Sprintf(" %s %d %s\n", fileInfo.MD5Checksum.Value, fileInfo.Size, "base/"+fileInfo.Type)
		blake2b += fmt.Sprintf(" %s %d %s\n", fileInfo.Blake2bHash.Value, fileInfo.Size, "base/"+fileInfo.Type)

		data = fmt.Sprintf(`Origin: %s
Label: %s
Suite: Sisyphus
Codename: %d
Date: %s
Architectures: %s
MD5Sum:
%sBLAKE2b:
%s

`,
			origin, label, codename, date, architecture, md5Sum, blake2b)
		_, err = addReleaseAsFileToRepo(ctx, pv, "release", data, group, architecture)
		if err != nil {
			return err
		}
	}
	return nil
}

func addReleaseAsFileToRepo(ctx context.Context, pv *packages_model.PackageVersion, filename, obj, group, arch string) (*repoData, error) {
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

	return &repoData{
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
	}, nil
}

func addPkglistAsFileToRepo(ctx context.Context, pv *packages_model.PackageVersion, filename string, orderedHeaders []headerWithIndexes, group, arch string) (*repoData, error) {
	content, _ := packages_module.NewHashedBuffer()
	defer content.Close()

	h := sha256.New()
	w := io.MultiWriter(content, h)
	buf := &bytes.Buffer{}

	for _, hdr := range orderedHeaders {
		if err := binary.Write(buf, binary.BigEndian, *hdr.header); err != nil {
			return nil, err
		}

		for _, index := range hdr.indexes {
			if err := binary.Write(buf, binary.BigEndian, *index.index); err != nil {
				return nil, err
			}
		}

		for _, index := range hdr.indexes {
			for _, indexValue := range index.data {
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

	if path.Ext(filename) == ".xz" {
		xzContent, err := compressXZ(buf.Bytes())
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(xzContent); err != nil {
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

	return &repoData{
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
	}, nil
}

func compressXZ(data []byte) ([]byte, error) {
	var xzContent bytes.Buffer
	xzWriter, err := xz.NewWriter(&xzContent)
	if err != nil {
		return nil, err
	}

	_, err = xzWriter.Write(data)
	xzWriter.Close()
	if err != nil {
		return nil, err
	}

	return xzContent.Bytes(), nil
}
