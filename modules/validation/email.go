// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"net/mail"
	"strings"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"

	"github.com/gobwas/glob"
)

// ErrEmailNotActivated e-mail address has not been activated error
var ErrEmailNotActivated = util.NewInvalidArgumentErrorf("e-mail address has not been activated")

// ErrEmailInvalid represents an error where the email address does not comply with RFC 5322
// or has a leading '-' character
type ErrEmailInvalid struct {
	Email string
}

// IsErrEmailInvalid checks if an error is an ErrEmailInvalid
func IsErrEmailInvalid(err error) bool {
	_, ok := err.(ErrEmailInvalid)
	return ok
}

func (err ErrEmailInvalid) Error() string {
	return fmt.Sprintf("e-mail invalid [email: %s]", err.Email)
}

func (err ErrEmailInvalid) Unwrap() error {
	return util.ErrInvalidArgument
}

// check if email is a valid address with allowed domain
func ValidateEmail(email string) error {
	if err := validateEmailBasic(email); err != nil {
		return err
	}
	return validateEmailDomain(email)
}

// check if email is a valid address when admins manually add or edit users
func ValidateEmailForAdmin(email string) error {
	return validateEmailBasic(email)
	// In this case we do not need to check the email domain
}

// validateEmailBasic checks whether the email complies with the rules
func validateEmailBasic(email string) error {
	if len(email) == 0 {
		return ErrEmailInvalid{email}
	}

	parsedAddress, err := mail.ParseAddress(email)
	if err != nil {
		return ErrEmailInvalid{email}
	}

	if parsedAddress.Name != "" {
		return ErrEmailInvalid{email}
	}

	return nil
}

func validateEmailDomain(email string) error {
	if !IsEmailDomainAllowed(email) {
		return ErrEmailInvalid{email}
	}

	return nil
}

func IsEmailDomainAllowed(email string) bool {
	return isEmailDomainAllowedInternal(
		email,
		setting.Service.EmailDomainAllowList,
		setting.Service.EmailDomainBlockList)
}

func isEmailDomainAllowedInternal(
	email string,
	emailDomainAllowList []glob.Glob,
	emailDomainBlockList []glob.Glob,
) bool {
	var result bool

	if len(emailDomainAllowList) == 0 {
		result = !isEmailDomainListed(emailDomainBlockList, email)
	} else {
		result = isEmailDomainListed(emailDomainAllowList, email)
	}
	return result
}

// isEmailDomainListed checks whether the domain of an email address
// matches a list of domains
func isEmailDomainListed(globs []glob.Glob, email string) bool {
	if len(globs) == 0 {
		return false
	}

	n := strings.LastIndex(email, "@")
	if n <= 0 {
		return false
	}

	domain := strings.ToLower(email[n+1:])

	for _, g := range globs {
		if g.Match(domain) {
			return true
		}
	}

	return false
}
