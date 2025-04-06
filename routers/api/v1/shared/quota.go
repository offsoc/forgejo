// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package shared

import (
	"net/http"

	quota_model "forgejo.org/models/quota"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

func GetQuota(ctx *context.APIContext, userID int64) {
	used, err := quota_model.GetUsedForUser(ctx, userID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetUsedForUser", err)
		return
	}

	groups, err := quota_model.GetGroupsForUser(ctx, userID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetGroupsForUser", err)
		return
	}

	result := convert.ToQuotaInfo(used, groups, false)
	ctx.JSON(http.StatusOK, &result)
}

func CheckQuota(ctx *context.APIContext, userID int64) {
	subjectQuery := ctx.FormTrim("subject")

	subject, err := quota_model.ParseLimitSubject(subjectQuery)
	if err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "quota_model.ParseLimitSubject", err)
		return
	}

	ok, err := quota_model.EvaluateForUser(ctx, userID, subject)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.EvaluateForUser", err)
		return
	}

	ctx.JSON(http.StatusOK, &ok)
}

func ListQuotaAttachments(ctx *context.APIContext, userID int64) {
	opts := utils.GetListOptions(ctx)
	count, attachments, err := quota_model.GetQuotaAttachmentsForUser(ctx, userID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaAttachmentsForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedAttachmentList(ctx, *attachments)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedAttachmentList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}

func ListQuotaPackages(ctx *context.APIContext, userID int64) {
	opts := utils.GetListOptions(ctx)
	count, packages, err := quota_model.GetQuotaPackagesForUser(ctx, userID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaPackagesForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedPackageList(ctx, *packages)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedPackageList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}

func ListQuotaArtifacts(ctx *context.APIContext, userID int64) {
	opts := utils.GetListOptions(ctx)
	count, artifacts, err := quota_model.GetQuotaArtifactsForUser(ctx, userID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaArtifactsForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedArtifactList(ctx, *artifacts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedArtifactList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}
