// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"forgejo.org/modules/util"
)

var (
	ErrLocaleAlreadyExist      = util.SilentWrap{Message: "lang already exists", Err: util.ErrAlreadyExist}
	ErrLocaleDoesNotExist      = util.SilentWrap{Message: "lang does not exist", Err: util.ErrNotExist}
	ErrTranslationDoesNotExist = util.SilentWrap{Message: "translation does not exist", Err: util.ErrNotExist}
	ErrUncertainArguments      = util.SilentWrap{Message: "arguments to i18n should not contain uncertain slices", Err: util.ErrInvalidArgument}
)
