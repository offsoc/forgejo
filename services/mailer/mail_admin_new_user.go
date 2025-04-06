// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package mailer

import (
	"bytes"
	"context"
	"strconv"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/templates"
	"forgejo.org/modules/translation"
)

const (
	tplNewUserMail base.TplName = "notify/admin_new_user"
)

// MailNewUser sends notification emails on new user registrations to all admins
func MailNewUser(ctx context.Context, u *user_model.User) {
	if !setting.Admin.SendNotificationEmailOnNewUser {
		return
	}
	if setting.MailService == nil {
		// No mail service configured
		return
	}

	recipients, err := user_model.GetAllAdmins(ctx)
	if err != nil {
		log.Error("user_model.GetAllAdmins: %v", err)
		return
	}

	langMap := make(map[string][]string)
	for _, r := range recipients {
		langMap[r.Language] = append(langMap[r.Language], r.Email)
	}

	for lang, tos := range langMap {
		mailNewUser(ctx, u, lang, tos)
	}
}

func mailNewUser(_ context.Context, u *user_model.User, lang string, tos []string) {
	locale := translation.NewLocale(lang)

	manageUserURL := setting.AppURL + "admin/users/" + strconv.FormatInt(u.ID, 10)
	subject := locale.TrString("mail.admin.new_user.subject", u.Name)
	body := locale.TrString("mail.admin.new_user.text", manageUserURL)
	mailMeta := map[string]any{
		"NewUser":      u,
		"NewUserUrl":   u.HTMLURL(),
		"Subject":      subject,
		"Body":         body,
		"Language":     locale.Language(),
		"Locale":       locale,
		"SanitizeHTML": templates.SanitizeHTML,
	}

	var mailBody bytes.Buffer

	if err := bodyTemplates.ExecuteTemplate(&mailBody, string(tplNewUserMail), mailMeta); err != nil {
		log.Error("ExecuteTemplate [%s]: %v", string(tplNewUserMail)+"/body", err)
		return
	}

	msgs := make([]*Message, 0, len(tos))
	for _, to := range tos {
		msg := NewMessage(to, subject, mailBody.String())
		msg.Info = subject
		msgs = append(msgs, msg)
	}
	SendAsync(msgs...)
}
