// Copyright 2024  The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"html/template"
	"net/http"

	webhook_model "forgejo.org/models/webhook"
	webhook_module "forgejo.org/modules/webhook"
	"forgejo.org/services/forms"
	"forgejo.org/services/webhook/shared"
)

type gogsHandler struct{ defaultHandler }

func (gogsHandler) Type() webhook_module.HookType { return webhook_module.GOGS }
func (gogsHandler) Icon(size int) template.HTML   { return shared.ImgIcon("gogs.ico", size) }

func (gogsHandler) UnmarshalForm(bind func(any)) forms.WebhookForm {
	var form struct {
		forms.WebhookCoreForm
		PayloadURL  string `binding:"Required;ValidUrl"`
		ContentType int    `binding:"Required"`
		Secret      string
	}
	bind(&form)

	contentType := webhook_model.ContentTypeJSON
	if webhook_model.HookContentType(form.ContentType) == webhook_model.ContentTypeForm {
		contentType = webhook_model.ContentTypeForm
	}
	return forms.WebhookForm{
		WebhookCoreForm: form.WebhookCoreForm,
		URL:             form.PayloadURL,
		ContentType:     contentType,
		Secret:          form.Secret,
		HTTPMethod:      http.MethodPost,
		Metadata:        nil,
	}
}
