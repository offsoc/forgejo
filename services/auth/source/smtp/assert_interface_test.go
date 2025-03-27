// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package smtp_test

import (
	auth_model "forgejo.org/models/auth"
	"forgejo.org/services/auth"
	"forgejo.org/services/auth/source/smtp"
)

// This test file exists to assert that our Source exposes the interfaces that we expect
// It tightly binds the interfaces and implementation without breaking go import cycles

type sourceInterface interface {
	auth.PasswordAuthenticator
	auth_model.Config
	auth_model.SkipVerifiable
	auth_model.HasTLSer
	auth_model.UseTLSer
	auth_model.SourceSettable
}

var _ (sourceInterface) = &smtp.Source{}
