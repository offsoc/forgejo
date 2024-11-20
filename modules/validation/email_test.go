// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved
// SPDX-License-Identifier: MIT

package validation

import (
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
)

func TestEmailAddressValidate(t *testing.T) {
	kases := map[string]error{
		"abc@gmail.com":                  nil,
		"132@hotmail.com":                nil,
		"1-3-2@test.org":                 nil,
		"1.3.2@test.org":                 nil,
		"a_123@test.org.cn":              nil,
		`first.last@iana.org`:            nil,
		`first!last@iana.org`:            nil,
		`first#last@iana.org`:            nil,
		`first$last@iana.org`:            nil,
		`first%last@iana.org`:            nil,
		`first&last@iana.org`:            nil,
		`first'last@iana.org`:            nil,
		`first*last@iana.org`:            nil,
		`first+last@iana.org`:            nil,
		`first/last@iana.org`:            nil,
		`first=last@iana.org`:            nil,
		`first?last@iana.org`:            nil,
		`first^last@iana.org`:            nil,
		"first`last@iana.org":            nil,
		`first{last@iana.org`:            nil,
		`first|last@iana.org`:            nil,
		`first}last@iana.org`:            nil,
		`first~last@iana.org`:            nil,
		`first;last@iana.org`:            ErrEmailCharIsNotSupported{`first;last@iana.org`},
		".233@qq.com":                    ErrEmailInvalid{".233@qq.com"},
		"!233@qq.com":                    nil,
		"#233@qq.com":                    nil,
		"$233@qq.com":                    nil,
		"%233@qq.com":                    nil,
		"&233@qq.com":                    nil,
		"'233@qq.com":                    nil,
		"*233@qq.com":                    nil,
		"+233@qq.com":                    nil,
		"-233@qq.com":                    ErrEmailInvalid{"-233@qq.com"},
		"/233@qq.com":                    nil,
		"=233@qq.com":                    nil,
		"?233@qq.com":                    nil,
		"^233@qq.com":                    nil,
		"_233@qq.com":                    nil,
		"`233@qq.com":                    nil,
		"{233@qq.com":                    nil,
		"|233@qq.com":                    nil,
		"}233@qq.com":                    nil,
		"~233@qq.com":                    nil,
		";233@qq.com":                    ErrEmailCharIsNotSupported{";233@qq.com"},
		"Foo <foo@bar.com>":              ErrEmailCharIsNotSupported{"Foo <foo@bar.com>"},
		string([]byte{0xE2, 0x84, 0xAA}): ErrEmailCharIsNotSupported{string([]byte{0xE2, 0x84, 0xAA})},
	}
	for kase, err := range kases {
		t.Run(kase, func(t *testing.T) {
			assert.EqualValues(t, err, ValidateEmail(kase))
		})
	}
}

func TestEmailDomainAllowList(t *testing.T) {
	res := IsEmailDomainAllowed("someuser@localhost.localdomain")
	assert.True(t, res)
}

func TestEmailDomainAllowListInternal(t *testing.T) {
	domain, _ := glob.Compile("domain.de", ',')
	emailDomainAllowList := []glob.Glob{domain}
	emailDomainBlockList := []glob.Glob{}

	res := isEmailDomainAllowedInternal(
		"user@repo.domain.de",
		emailDomainAllowList,
		emailDomainBlockList,
		false,
		"https://repo.domain.de")
	assert.False(t, res)

	res = isEmailDomainAllowedInternal(
		"user@repo.domain.de",
		emailDomainAllowList,
		emailDomainBlockList,
		true,
		"xttps://repo")
	assert.False(t, res)

	res = isEmailDomainAllowedInternal(
		"user@repo.Domain.de",
		emailDomainAllowList,
		emailDomainBlockList,
		true,
		"https://repo.domain.de")
	assert.True(t, res)

	res = isEmailDomainAllowedInternal(
		"user@repo.domain.de",
		emailDomainAllowList,
		emailDomainBlockList,
		true,
		"https://repo.domain.de")
	assert.True(t, res)
}
