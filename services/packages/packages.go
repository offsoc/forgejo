// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	packages_module "forgejo.org/modules/packages"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	notify_service "forgejo.org/services/notify"
)

var (
	ErrQuotaTypeSize   = errors.New("maximum allowed package type size exceeded")
	ErrQuotaTotalSize  = errors.New("maximum allowed package storage quota exceeded")
	ErrQuotaTotalCount = errors.New("maximum allowed package count exceeded")
)

// PackageInfo describes a package
type PackageInfo struct {
	Owner       *user_model.User
	PackageType packages_model.Type
	Name        string
	Version     string
}

// PackageCreationInfo describes a package to create
type PackageCreationInfo struct {
	PackageInfo
	SemverCompatible  bool
	Creator           *user_model.User
	Metadata          any
	PackageProperties map[string]string
	VersionProperties map[string]string
}

// PackageFileInfo describes a package file
type PackageFileInfo struct {
	Filename     string
	CompositeKey string
}

// PackageFileCreationInfo describes a package file to create
type PackageFileCreationInfo struct {
	PackageFileInfo
	Creator           *user_model.User
	Data              packages_module.HashedSizeReader
	IsLead            bool
	Properties        map[string]string
	OverwriteExisting bool
}

// CreatePackageAndAddFile creates a package with a file. If the same package exists already, ErrDuplicatePackageVersion is returned
func CreatePackageAndAddFile(ctx context.Context, pvci *PackageCreationInfo, pfci *PackageFileCreationInfo) (*packages_model.PackageVersion, *packages_model.PackageFile, error) {
	return createPackageAndAddFile(ctx, pvci, pfci, false)
}

// CreatePackageOrAddFileToExisting creates a package with a file or adds the file if the package exists already
func CreatePackageOrAddFileToExisting(ctx context.Context, pvci *PackageCreationInfo, pfci *PackageFileCreationInfo) (*packages_model.PackageVersion, *packages_model.PackageFile, error) {
	return createPackageAndAddFile(ctx, pvci, pfci, true)
}

func createPackageAndAddFile(ctx context.Context, pvci *PackageCreationInfo, pfci *PackageFileCreationInfo, allowDuplicate bool) (*packages_model.PackageVersion, *packages_model.PackageFile, error) {
	dbCtx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer committer.Close()

	pv, created, err := createPackageAndVersion(dbCtx, pvci, allowDuplicate)
	if err != nil {
		return nil, nil, err
	}

	pf, pb, blobCreated, err := addFileToPackageVersion(dbCtx, pv, &pvci.PackageInfo, pfci)
	removeBlob := false
	defer func() {
		if blobCreated && removeBlob {
			contentStore := packages_module.NewContentStore()
			if err := contentStore.Delete(packages_module.BlobHash256Key(pb.HashSHA256)); err != nil {
				log.Error("Error deleting package blob from content store: %v", err)
			}
		}
	}()
	if err != nil {
		removeBlob = true
		return nil, nil, err
	}

	if err := committer.Commit(); err != nil {
		removeBlob = true
		return nil, nil, err
	}

	if created {
		pd, err := packages_model.GetPackageDescriptor(ctx, pv)
		if err != nil {
			return nil, nil, err
		}

		notify_service.PackageCreate(ctx, pvci.Creator, pd)
	}

	return pv, pf, nil
}

func createPackageAndVersion(ctx context.Context, pvci *PackageCreationInfo, allowDuplicate bool) (*packages_model.PackageVersion, bool, error) {
	log.Trace("Creating package: %v, %v, %v, %s, %s, %+v, %+v, %v", pvci.Creator.ID, pvci.Owner.ID, pvci.PackageType, pvci.Name, pvci.Version, pvci.PackageProperties, pvci.VersionProperties, allowDuplicate)

	packageCreated := true
	p := &packages_model.Package{
		OwnerID:          pvci.Owner.ID,
		Type:             pvci.PackageType,
		Name:             pvci.Name,
		LowerName:        packages_model.ResolvePackageName(pvci.Name, pvci.PackageType),
		SemverCompatible: pvci.SemverCompatible,
	}
	var err error
	if p, err = packages_model.TryInsertPackage(ctx, p); err != nil {
		if errors.Is(err, packages_model.ErrDuplicatePackage) {
			packageCreated = false
		} else {
			log.Error("Error inserting package: %v", err)
			return nil, false, err
		}
	}

	if packageCreated {
		for name, value := range pvci.PackageProperties {
			if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypePackage, p.ID, name, value); err != nil {
				log.Error("Error setting package property: %v", err)
				return nil, false, err
			}
		}
	}

	metadataJSON, err := json.Marshal(pvci.Metadata)
	if err != nil {
		return nil, false, err
	}

	versionCreated := true
	pv := &packages_model.PackageVersion{
		PackageID:    p.ID,
		CreatorID:    pvci.Creator.ID,
		Version:      pvci.Version,
		LowerVersion: strings.ToLower(pvci.Version),
		MetadataJSON: string(metadataJSON),
	}
	if pv, err = packages_model.GetOrInsertVersion(ctx, pv); err != nil {
		if err == packages_model.ErrDuplicatePackageVersion {
			versionCreated = false
		} else {
			log.Error("Error inserting package: %v", err)
			return nil, false, err
		}

		if !allowDuplicate {
			// no need to log an error
			return nil, false, err
		}
	}

	if versionCreated {
		if err := CheckCountQuotaExceeded(ctx, pvci.Creator, pvci.Owner); err != nil {
			return nil, false, err
		}

		for name, value := range pvci.VersionProperties {
			if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypeVersion, pv.ID, name, value); err != nil {
				log.Error("Error setting package version property: %v", err)
				return nil, false, err
			}
		}
	}

	return pv, versionCreated, nil
}

// AddFileToExistingPackage adds a file to an existing package. If the package does not exist, ErrPackageNotExist is returned
func AddFileToExistingPackage(ctx context.Context, pvi *PackageInfo, pfci *PackageFileCreationInfo) (*packages_model.PackageFile, error) {
	return addFileToPackageWrapper(ctx, func(ctx context.Context) (*packages_model.PackageFile, *packages_model.PackageBlob, bool, error) {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx, pvi.Owner.ID, pvi.PackageType, pvi.Name, pvi.Version)
		if err != nil {
			return nil, nil, false, err
		}

		return addFileToPackageVersion(ctx, pv, pvi, pfci)
	})
}

// AddFileToPackageVersionInternal adds a file to the package
// This method skips quota checks and should only be used for system-managed packages.
func AddFileToPackageVersionInternal(ctx context.Context, pv *packages_model.PackageVersion, pfci *PackageFileCreationInfo) (*packages_model.PackageFile, error) {
	return addFileToPackageWrapper(ctx, func(ctx context.Context) (*packages_model.PackageFile, *packages_model.PackageBlob, bool, error) {
		return addFileToPackageVersionUnchecked(ctx, pv, pfci, "")
	})
}

func addFileToPackageWrapper(ctx context.Context, fn func(ctx context.Context) (*packages_model.PackageFile, *packages_model.PackageBlob, bool, error)) (*packages_model.PackageFile, error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	pf, pb, blobCreated, err := fn(ctx)
	removeBlob := false
	defer func() {
		if removeBlob {
			contentStore := packages_module.NewContentStore()
			if err := contentStore.Delete(packages_module.BlobHash256Key(pb.HashSHA256)); err != nil {
				log.Error("Error deleting package blob from content store: %v", err)
			}
		}
	}()
	if err != nil {
		removeBlob = blobCreated
		return nil, err
	}

	if err := committer.Commit(); err != nil {
		removeBlob = blobCreated
		return nil, err
	}

	return pf, nil
}

// NewPackageBlob creates a package blob instance
func NewPackageBlob(hsr packages_module.HashedSizeReader) *packages_model.PackageBlob {
	hashMD5, hashSHA1, hashSHA256, hashSHA512, hashBlake2b := hsr.Sums()

	return &packages_model.PackageBlob{
		Size:        hsr.Size(),
		HashMD5:     hex.EncodeToString(hashMD5),
		HashSHA1:    hex.EncodeToString(hashSHA1),
		HashSHA256:  hex.EncodeToString(hashSHA256),
		HashSHA512:  hex.EncodeToString(hashSHA512),
		HashBlake2b: hex.EncodeToString(hashBlake2b),
	}
}

func addFileToPackageVersion(ctx context.Context, pv *packages_model.PackageVersion, pvi *PackageInfo, pfci *PackageFileCreationInfo) (*packages_model.PackageFile, *packages_model.PackageBlob, bool, error) {
	if err := CheckSizeQuotaExceeded(ctx, pfci.Creator, pvi.Owner, pvi.PackageType, pfci.Data.Size()); err != nil {
		return nil, nil, false, err
	}

	return addFileToPackageVersionUnchecked(ctx, pv, pfci, pvi.PackageType)
}

func addFileToPackageVersionUnchecked(ctx context.Context, pv *packages_model.PackageVersion, pfci *PackageFileCreationInfo, packageType packages_model.Type) (*packages_model.PackageFile, *packages_model.PackageBlob, bool, error) {
	log.Trace("Adding package file: %v, %s", pv.ID, pfci.Filename)

	pb, exists, err := packages_model.GetOrInsertBlob(ctx, NewPackageBlob(pfci.Data))
	if err != nil {
		log.Error("Error inserting package blob: %v", err)
		return nil, nil, false, err
	}
	if !exists {
		contentStore := packages_module.NewContentStore()
		if err := contentStore.Save(packages_module.BlobHash256Key(pb.HashSHA256), pfci.Data, pfci.Data.Size()); err != nil {
			log.Error("Error saving package blob in content store: %v", err)
			return nil, nil, false, err
		}
	}

	if pfci.OverwriteExisting {
		pf, err := packages_model.GetFileForVersionByName(ctx, pv.ID, pfci.Filename, pfci.CompositeKey)
		if err != nil && err != packages_model.ErrPackageFileNotExist {
			return nil, pb, !exists, err
		}
		if pf != nil {
			// Short circuit if blob is the same
			if pf.BlobID == pb.ID {
				return pf, pb, !exists, nil
			}

			if err := packages_model.DeleteAllProperties(ctx, packages_model.PropertyTypeFile, pf.ID); err != nil {
				return nil, pb, !exists, err
			}
			if err := packages_model.DeleteFileByID(ctx, pf.ID); err != nil {
				return nil, pb, !exists, err
			}
		}
	}

	pf := &packages_model.PackageFile{
		VersionID:    pv.ID,
		BlobID:       pb.ID,
		Name:         pfci.Filename,
		LowerName:    packages_model.ResolvePackageName(pfci.Filename, packageType),
		CompositeKey: pfci.CompositeKey,
		IsLead:       pfci.IsLead,
	}
	if pf, err = packages_model.TryInsertFile(ctx, pf); err != nil {
		if err != packages_model.ErrDuplicatePackageFile {
			log.Error("Error inserting package file: %v", err)
		}
		return nil, pb, !exists, err
	}

	for name, value := range pfci.Properties {
		if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypeFile, pf.ID, name, value); err != nil {
			log.Error("Error setting package file property: %v", err)
			return pf, pb, !exists, err
		}
	}

	return pf, pb, !exists, nil
}

// CheckCountQuotaExceeded checks if the owner has more than the allowed packages
// The check is skipped if the doer is an admin.
func CheckCountQuotaExceeded(ctx context.Context, doer, owner *user_model.User) error {
	if doer.IsAdmin {
		return nil
	}

	if setting.Packages.LimitTotalOwnerCount > -1 {
		totalCount, err := packages_model.CountVersions(ctx, &packages_model.PackageSearchOptions{
			OwnerID:    owner.ID,
			IsInternal: optional.Some(false),
		})
		if err != nil {
			log.Error("CountVersions failed: %v", err)
			return err
		}
		if totalCount > setting.Packages.LimitTotalOwnerCount {
			return ErrQuotaTotalCount
		}
	}

	return nil
}

// CheckSizeQuotaExceeded checks if the upload size is bigger than the allowed size
// The check is skipped if the doer is an admin.
func CheckSizeQuotaExceeded(ctx context.Context, doer, owner *user_model.User, packageType packages_model.Type, uploadSize int64) error {
	if doer.IsAdmin {
		return nil
	}

	var typeSpecificSize int64
	switch packageType {
	case packages_model.TypeAlpine:
		typeSpecificSize = setting.Packages.LimitSizeAlpine
	case packages_model.TypeArch:
		typeSpecificSize = setting.Packages.LimitSizeArch
	case packages_model.TypeCargo:
		typeSpecificSize = setting.Packages.LimitSizeCargo
	case packages_model.TypeChef:
		typeSpecificSize = setting.Packages.LimitSizeChef
	case packages_model.TypeComposer:
		typeSpecificSize = setting.Packages.LimitSizeComposer
	case packages_model.TypeConan:
		typeSpecificSize = setting.Packages.LimitSizeConan
	case packages_model.TypeConda:
		typeSpecificSize = setting.Packages.LimitSizeConda
	case packages_model.TypeContainer:
		typeSpecificSize = setting.Packages.LimitSizeContainer
	case packages_model.TypeCran:
		typeSpecificSize = setting.Packages.LimitSizeCran
	case packages_model.TypeDebian:
		typeSpecificSize = setting.Packages.LimitSizeDebian
	case packages_model.TypeGeneric:
		typeSpecificSize = setting.Packages.LimitSizeGeneric
	case packages_model.TypeGo:
		typeSpecificSize = setting.Packages.LimitSizeGo
	case packages_model.TypeHelm:
		typeSpecificSize = setting.Packages.LimitSizeHelm
	case packages_model.TypeMaven:
		typeSpecificSize = setting.Packages.LimitSizeMaven
	case packages_model.TypeNpm:
		typeSpecificSize = setting.Packages.LimitSizeNpm
	case packages_model.TypeNuGet:
		typeSpecificSize = setting.Packages.LimitSizeNuGet
	case packages_model.TypePub:
		typeSpecificSize = setting.Packages.LimitSizePub
	case packages_model.TypePyPI:
		typeSpecificSize = setting.Packages.LimitSizePyPI
	case packages_model.TypeRpm:
		typeSpecificSize = setting.Packages.LimitSizeRpm
	case packages_model.TypeAlt:
		typeSpecificSize = setting.Packages.LimitSizeAlt
	case packages_model.TypeRubyGems:
		typeSpecificSize = setting.Packages.LimitSizeRubyGems
	case packages_model.TypeSwift:
		typeSpecificSize = setting.Packages.LimitSizeSwift
	case packages_model.TypeVagrant:
		typeSpecificSize = setting.Packages.LimitSizeVagrant
	}
	if typeSpecificSize > -1 && typeSpecificSize < uploadSize {
		return ErrQuotaTypeSize
	}

	if setting.Packages.LimitTotalOwnerSize > -1 {
		totalSize, err := packages_model.CalculateFileSize(ctx, &packages_model.PackageFileSearchOptions{
			OwnerID: owner.ID,
		})
		if err != nil {
			log.Error("CalculateFileSize failed: %v", err)
			return err
		}
		if totalSize+uploadSize > setting.Packages.LimitTotalOwnerSize {
			return ErrQuotaTotalSize
		}
	}

	return nil
}

// GetOrCreateInternalPackageVersion gets or creates an internal package
// Some package types need such internal packages for housekeeping.
func GetOrCreateInternalPackageVersion(ctx context.Context, ownerID int64, packageType packages_model.Type, name, version string) (*packages_model.PackageVersion, error) {
	var pv *packages_model.PackageVersion

	return pv, db.WithTx(ctx, func(ctx context.Context) error {
		p := &packages_model.Package{
			OwnerID:    ownerID,
			Type:       packageType,
			Name:       name,
			LowerName:  name,
			IsInternal: true,
		}
		var err error
		if p, err = packages_model.TryInsertPackage(ctx, p); err != nil {
			if err != packages_model.ErrDuplicatePackage {
				log.Error("Error inserting package: %v", err)
				return err
			}
		}

		pv = &packages_model.PackageVersion{
			PackageID:    p.ID,
			CreatorID:    ownerID,
			Version:      version,
			LowerVersion: version,
			IsInternal:   true,
			MetadataJSON: "null",
		}
		if pv, err = packages_model.GetOrInsertVersion(ctx, pv); err != nil {
			if err != packages_model.ErrDuplicatePackageVersion {
				log.Error("Error inserting package version: %v", err)
				return err
			}
		}

		return nil
	})
}

// RemovePackageVersionByNameAndVersion deletes a package version and all associated files
func RemovePackageVersionByNameAndVersion(ctx context.Context, doer *user_model.User, pvi *PackageInfo) error {
	pv, err := packages_model.GetVersionByNameAndVersion(ctx, pvi.Owner.ID, pvi.PackageType, pvi.Name, pvi.Version)
	if err != nil {
		return err
	}

	return RemovePackageVersion(ctx, doer, pv)
}

// RemovePackageVersion deletes the package version and all associated files
func RemovePackageVersion(ctx context.Context, doer *user_model.User, pv *packages_model.PackageVersion) error {
	dbCtx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	pd, err := packages_model.GetPackageDescriptor(dbCtx, pv)
	if err != nil {
		return err
	}

	log.Trace("Deleting package: %v", pv.ID)

	if err := DeletePackageVersionAndReferences(dbCtx, pv); err != nil {
		return err
	}

	if err := committer.Commit(); err != nil {
		return err
	}

	notify_service.PackageDelete(ctx, doer, pd)

	return nil
}

// RemovePackageFileAndVersionIfUnreferenced deletes the package file and the version if there are no referenced files afterwards
func RemovePackageFileAndVersionIfUnreferenced(ctx context.Context, doer *user_model.User, pf *packages_model.PackageFile) error {
	var pd *packages_model.PackageDescriptor

	if err := db.WithTx(ctx, func(ctx context.Context) error {
		if err := DeletePackageFile(ctx, pf); err != nil {
			return err
		}

		has, err := packages_model.HasVersionFileReferences(ctx, pf.VersionID)
		if err != nil {
			return err
		}
		if !has {
			pv, err := packages_model.GetVersionByID(ctx, pf.VersionID)
			if err != nil {
				return err
			}

			pd, err = packages_model.GetPackageDescriptor(ctx, pv)
			if err != nil {
				return err
			}

			if err := DeletePackageVersionAndReferences(ctx, pv); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if pd != nil {
		notify_service.PackageDelete(ctx, doer, pd)
	}

	return nil
}

// DeletePackageVersionAndReferences deletes the package version and its properties and files
func DeletePackageVersionAndReferences(ctx context.Context, pv *packages_model.PackageVersion) error {
	if err := packages_model.DeleteAllProperties(ctx, packages_model.PropertyTypeVersion, pv.ID); err != nil {
		return err
	}

	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return err
	}

	for _, pf := range pfs {
		if err := DeletePackageFile(ctx, pf); err != nil {
			return err
		}
	}

	return packages_model.DeleteVersionByID(ctx, pv.ID)
}

// DeletePackageFile deletes the package file and its properties
func DeletePackageFile(ctx context.Context, pf *packages_model.PackageFile) error {
	if err := packages_model.DeleteAllProperties(ctx, packages_model.PropertyTypeFile, pf.ID); err != nil {
		return err
	}
	return packages_model.DeleteFileByID(ctx, pf.ID)
}

// GetFileStreamByPackageNameAndVersion returns the content of the specific package file
func GetFileStreamByPackageNameAndVersion(ctx context.Context, pvi *PackageInfo, pfi *PackageFileInfo) (io.ReadSeekCloser, *url.URL, *packages_model.PackageFile, error) {
	log.Trace("Getting package file stream: %v, %v, %s, %s, %s, %s", pvi.Owner.ID, pvi.PackageType, pvi.Name, pvi.Version, pfi.Filename, pfi.CompositeKey)

	pv, err := packages_model.GetVersionByNameAndVersion(ctx, pvi.Owner.ID, pvi.PackageType, pvi.Name, pvi.Version)
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			return nil, nil, nil, err
		}
		log.Error("Error getting package: %v", err)
		return nil, nil, nil, err
	}

	return GetFileStreamByPackageVersion(ctx, pv, pfi)
}

// GetFileStreamByPackageVersion returns the content of the specific package file
func GetFileStreamByPackageVersion(ctx context.Context, pv *packages_model.PackageVersion, pfi *PackageFileInfo) (io.ReadSeekCloser, *url.URL, *packages_model.PackageFile, error) {
	pf, err := packages_model.GetFileForVersionByName(ctx, pv.ID, pfi.Filename, pfi.CompositeKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return GetPackageFileStream(ctx, pf)
}

// GetPackageFileStream returns the content of the specific package file
func GetPackageFileStream(ctx context.Context, pf *packages_model.PackageFile) (io.ReadSeekCloser, *url.URL, *packages_model.PackageFile, error) {
	pb, err := packages_model.GetBlobByID(ctx, pf.BlobID)
	if err != nil {
		return nil, nil, nil, err
	}

	return GetPackageBlobStream(ctx, pf, pb, nil)
}

// GetPackageBlobStream returns the content of the specific package blob
// If the storage supports direct serving and it's enabled, only the direct serving url is returned.
func GetPackageBlobStream(ctx context.Context, pf *packages_model.PackageFile, pb *packages_model.PackageBlob, serveDirectReqParams url.Values) (io.ReadSeekCloser, *url.URL, *packages_model.PackageFile, error) {
	key := packages_module.BlobHash256Key(pb.HashSHA256)

	cs := packages_module.NewContentStore()

	var s io.ReadSeekCloser
	var u *url.URL
	var err error

	if cs.ShouldServeDirect() {
		u, err = cs.GetServeDirectURL(key, pf.Name, serveDirectReqParams)
		if err != nil && !errors.Is(err, storage.ErrURLNotSupported) {
			log.Error("Error getting serve direct url: %v", err)
		}
	}
	if u == nil {
		s, err = cs.Get(key)
	}

	if err == nil {
		if pf.IsLead {
			if err := packages_model.IncrementDownloadCounter(ctx, pf.VersionID); err != nil {
				log.Error("Error incrementing download counter: %v", err)
			}
		}
	}
	return s, u, pf, err
}

// RemoveAllPackages for User
func RemoveAllPackages(ctx context.Context, userID int64) (int, error) {
	count := 0
	for {
		pkgVersions, _, err := packages_model.SearchVersions(ctx, &packages_model.PackageSearchOptions{
			Paginator: &db.ListOptions{
				PageSize: repo_model.RepositoryListDefaultPageSize,
				Page:     1,
			},
			OwnerID:    userID,
			IsInternal: optional.None[bool](),
		})
		if err != nil {
			return count, fmt.Errorf("GetOwnedPackages[%d]: %w", userID, err)
		}
		if len(pkgVersions) == 0 {
			break
		}
		for _, pv := range pkgVersions {
			if err := DeletePackageVersionAndReferences(ctx, pv); err != nil {
				return count, fmt.Errorf("unable to delete package %d:%s[%d]. Error: %w", pv.PackageID, pv.Version, pv.ID, err)
			}
			count++
		}
	}
	return count, nil
}
