// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package packages

import (
	"io"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	packages_module "code.gitea.io/gitea/modules/packages"
)

// CreatePackage creates a new package
func CreatePackage(creator *models.User, repository *models.Repository, packageType models.PackageType, name, version string, metaData interface{}, allowDuplicate bool) (*models.Package, error) {
	p := &models.Package{
		RepositoryID: repository.ID,
		CreatorID:    creator.ID,
		Type:         packageType,
		Name:         name,
		LowerName:    strings.ToLower(name),
		Version:      version,
		MetaData:     metaData,
	}
	if err := models.TryInsertPackage(p); err != nil {
		if err == models.ErrDuplicatePackage {
			if allowDuplicate {
				return p, nil
			}
			return nil, err
		}
		log.Error("Error inserting package: %v", err)
		return nil, err
	}
	return p, nil
}

// AddFileToPackage adds a new file to package and stores its content
func AddFileToPackage(p *models.Package, filename string, size int64, r io.Reader) (*models.PackageFile, error) {
	pf := &models.PackageFile{
		PackageID: p.ID,
		Size:      size,
		Name:      filename,
		LowerName: strings.ToLower(filename),
	}
	if err := models.InsertPackageFile(pf); err != nil {
		log.Error("Error inserting package file: %v", err)
		return nil, err
	}

	packageStore := packages_module.NewContentStore()
	if err := packageStore.Save(pf.ID, r, size); err != nil {
		log.Error("Error saving package file: %v", err)
		if err := models.DeletePackageFileByID(pf.ID); err != nil {
			log.Error("Error deleting package file: %v", err)
		}
		return nil, err
	}
	return pf, nil
}

// DeletePackage deletes a package and all associated files
func DeletePackage(repository *models.Repository, packageType models.PackageType, name, version string) error {
	p, err := models.GetPackageByNameAndVersion(repository.ID, packageType, name, version)
	if err != nil {
		if err == models.ErrPackageNotExist {
			return err
		}
		log.Error("Error getting package: %v", err)
		return err
	}

	pfs, err := p.GetFiles()
	if err != nil {
		log.Error("Error getting package files: %v", err)
		return err
	}

	contentStore := packages_module.NewContentStore()
	for _, pf := range pfs {
		if err := contentStore.Delete(pf.ID); err != nil {
			log.Error("Error deleting package file: %v", err)
			return err
		}
	}

	if err := models.DeletePackageByID(p.ID); err != nil {
		log.Error("Error deleting package: %v", err)
		return err
	}

	return nil
}
