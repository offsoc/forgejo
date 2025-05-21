package http

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	packages_module "forgejo.org/modules/packages"
	opentofu_state_module "forgejo.org/modules/packages/opentofu/state"
	"forgejo.org/modules/setting"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

// apiError logs and processes a REST API error.
func apiError(ctx *context.Context, status int, obj any) {
	type Error struct {
		Code    string `json:"code"`
		Message string `json:"message,omitempty"`
	}

	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		ctx.JSON(status, Error{
			Code:    http.StatusText(status),
			Message: message,
		})
	})
}

func GetLockId(ctx *context.Context) (string, error) {
	panic("Not yet implemented")
}

// GetState returns the latest version of the state file if any.
func GetState(ctx *context.Context) {
	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package fetch request: %s", packageName)

	// Search for the latest versions of the package/state file.
	pvs, _, err := packages_model.SearchLatestVersions(ctx, &packages_model.PackageSearchOptions{
		OwnerID:         ctx.Package.Owner.ID,
		Type:            packages_model.TypeOpenTofuState,
		Name:            packages_model.SearchValue{ExactMatch: true, Value: packageName},
		HasFileWithName: opentofu_state_module.OpenTofuStateFilename,
		IsInternal:      optional.Some(false),
		Sort:            packages_model.SortCreatedDesc,
	})
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Failed to search for the latest versions of the state file: %w", err))
		return
	}

	// If the package/state file does not exist.
	if len(pvs) == 0 {
		apiError(ctx, http.StatusNoContent, "No state file available to download.")
		return
	}

	// Get the latest version of the package/state file now that we know its version
	// number.
	s, _, pf, err := packages_service.GetFileStreamByPackageNameAndVersion(
		ctx,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeOpenTofuState,
			Name:        packageName,
			Version:     pvs[0].Version,
		},
		&packages_service.PackageFileInfo{
			Filename: opentofu_state_module.OpenTofuStateFilename,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, packages_model.ErrPackageNotExist):
			apiError(ctx, http.StatusNotFound, "The package cannot be found.")
		case errors.Is(err, packages_model.ErrPackageFileNotExist):
			apiError(ctx, http.StatusNotFound, "The state file cannot be found in the package.")
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Failed to find the package and its state file: %w", err))
		}

		return
	}
	defer s.Close()

	// Return the state file.
	helper.ServePackageFile(ctx, s, nil, pf, &context.ServeHeaderOptions{
		ContentType:  "application/json",
		Filename:     pf.Name,
		LastModified: pf.CreatedUnix.AsLocalTime(),
	})
}

// UpdateState processes the REST API requests received to create/update an
// OpenTofu/Terraform state file as Forgejo package.
func UpdateState(ctx *context.Context) {
	defer ctx.Req.Body.Close()

	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package update request: %s", packageName)

	// Check the size of the state file.
	contentLength := ctx.Req.ContentLength
	log.Debug("Update request's content length: %d", contentLength)
	if contentLength == -1 {
		apiError(ctx, http.StatusLengthRequired, "The content length is unknown.")
		return
	} else if contentLength == 0 {
		apiError(ctx, http.StatusBadRequest, "The body is empty.")
		return
	} else if setting.Packages.LimitSizeOpenTofuState > -1 && contentLength > setting.Packages.LimitSizeOpenTofuState {
		apiError(ctx, http.StatusRequestEntityTooLarge, "The request body exceeds the package size limit defined by the server.")
		return
	}

	// Get the optional lock ID from the request.
	lockID := ctx.Req.Header.Get("ID")
	if lockID != "" {
		log.Debug("Update request has lock ID: %s", lockID)

		// TODO
		panic("Not yet implemented")
	}

	// Read the state file from the request body.
	//
	// The amount of bytes to read is limited by the value of the request's content
	// length to avoid denial of service attacks.
	stateFile, err := io.ReadAll(http.MaxBytesReader(ctx.Resp, ctx.Req.Body, contentLength))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Failed to read the state file from the request body: %w", err))
		return
	}

	var md5Hash [16]byte

	// If the request contains an MD5 checksum in its headers, check if it matches
	// the request body.
	md5Checksum := ctx.Req.Header.Get("Content-MD5")
	if md5Checksum != "" {
		log.Debug("Update request has an MD5 checksum: %s", md5Checksum)

		md5Hash = md5.Sum([]byte(stateFile))
		md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

		if md5Checksum != md5Base64 {
			apiError(ctx, http.StatusBadRequest, "The MD5 checksum sent with the request does not match the body contents.")
			return
		}
	}

	// Parse the state file to extract metadata.
	metadata, err := opentofu_state_module.ParseMetadataFromStateFile(&stateFile, &md5Hash)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, "Failed to parse the state file.")
		return
	}

	// Prepare the state file to be stored as a Forgejo package.
	packageData, err := packages_module.CreateHashedBufferFromReader(bytes.NewReader(stateFile))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Failed to create an hashed buffer from the state file: %w", err))
		return
	}
	defer packageData.Close()

	// Create the package.
	_, _, err = packages_service.CreatePackageAndAddFile(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       ctx.Package.Owner,
				PackageType: packages_model.TypeOpenTofuState,
				Name:        packageName,
				Version:     strconv.FormatUint(metadata.Serial, 10),
			},
			Creator:  ctx.Doer,
			Metadata: metadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: opentofu_state_module.OpenTofuStateFilename,
			},
			Creator: ctx.Doer,
			Data:    packageData,
			IsLead:  true,
		},
	)
	if err != nil {
		switch err {
		case packages_model.ErrDuplicatePackageVersion:
			apiError(ctx, http.StatusConflict, "A package with the same version number already exists.")
			return
		case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
			apiError(ctx, http.StatusForbidden, fmt.Errorf("Quota exceeded: %v.", err))
			return
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Failed to create the package and add the files to it: %w", err))
		}

		return
	}

	ctx.JSON(http.StatusOK, map[string]string{
		"message": "State file successfully uploaded.",
		"package": packageName,
	})
}

func LockState(ctx *context.Context) {
	panic("Not yet implemented")
}

func UnlockState(ctx *context.Context) {
	panic("Not yet implemented")
}

func DeleteState(ctx *context.Context) {
	panic("Not yet implemented")
}
