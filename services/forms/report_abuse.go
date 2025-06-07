// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forms

import (
	"net/http"

	"forgejo.org/models/moderation"
	"forgejo.org/modules/web/middleware"
	"forgejo.org/services/context"

	"code.forgejo.org/go-chi/binding"
)

// ReportAbuseForm is used to interact with the UI of the form that submits new abuse reports.
type ReportAbuseForm struct {
	ContentID     int64
	ContentType   moderation.ReportedContentType
	AbuseCategory moderation.AbuseCategoryType `binding:"Required" locale:"moderation.abuse_category"`
	Remarks       string                       `binding:"Required;MinSize(20);MaxSize(500)" preprocess:"TrimSpace" locale:"moderation.report_remarks"`
}

// Validate validates the fields of ReportAbuseForm.
func (f *ReportAbuseForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
