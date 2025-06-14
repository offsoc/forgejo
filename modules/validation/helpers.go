// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"forgejo.org/modules/setting"
)

var externalTrackerRegex = regexp.MustCompile(`({?)(?:user|repo|index)+?(}?)`)

func isLoopbackIP(ip string) bool {
	return net.ParseIP(ip).IsLoopback()
}

// IsValidURL checks if URL is valid
func IsValidURL(uri string) bool {
	if u, err := url.ParseRequestURI(uri); err != nil ||
		(u.Scheme != "http" && u.Scheme != "https") ||
		!validPort(portOnly(u.Host)) {
		return false
	}

	return true
}

// IsValidSiteURL checks if URL is valid
func IsValidSiteURL(uri string) bool {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return false
	}

	if !validPort(portOnly(u.Host)) {
		return false
	}

	for _, scheme := range setting.Service.ValidSiteURLSchemes {
		if scheme == u.Scheme {
			return true
		}
	}
	return false
}

// IsAPIURL checks if URL is current Gitea instance API URL
func IsAPIURL(uri string) bool {
	return strings.HasPrefix(strings.ToLower(uri), strings.ToLower(setting.AppURL+"api"))
}

// IsValidExternalURL checks if URL is valid external URL
func IsValidExternalURL(uri string) bool {
	if !IsValidURL(uri) || IsAPIURL(uri) {
		return false
	}

	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return false
	}

	// Currently check only if not loopback IP is provided to keep compatibility
	if isLoopbackIP(u.Hostname()) || strings.ToLower(u.Hostname()) == "localhost" {
		return false
	}

	// TODO: Later it should be added to allow local network IP addresses
	//       only if allowed by special setting

	return true
}

// IsValidReleaseAssetURL checks if the URL is valid for external release assets
func IsValidReleaseAssetURL(uri string) bool {
	return IsValidURL(uri)
}

// IsValidExternalTrackerURLFormat checks if URL matches required syntax for external trackers
func IsValidExternalTrackerURLFormat(uri string) bool {
	if !IsValidExternalURL(uri) {
		return false
	}

	// check for typoed variables like /{index/ or /[repo}
	for _, match := range externalTrackerRegex.FindAllStringSubmatch(uri, -1) {
		if (match[1] == "{" || match[2] == "}") && (match[1] != "{" || match[2] != "}") {
			return false
		}
	}

	return true
}

var (
	validUsernamePatternWithDots    = regexp.MustCompile(`^[\da-zA-Z][-.\w]*$`)
	validUsernamePatternWithoutDots = regexp.MustCompile(`^[\da-zA-Z][-\w]*$`)

	// No consecutive or trailing non-alphanumeric chars, catches both cases
	invalidUsernamePattern = regexp.MustCompile(`[-._]{2,}|[-._]$`)
)

// IsValidUsername checks if username is valid
func IsValidUsername(name string) bool {
	// It is difficult to find a single pattern that is both readable and effective,
	// but it's easier to use positive and negative checks.
	if setting.Service.AllowDotsInUsernames {
		return validUsernamePatternWithDots.MatchString(name) && !invalidUsernamePattern.MatchString(name)
	}

	return validUsernamePatternWithoutDots.MatchString(name) && !invalidUsernamePattern.MatchString(name)
}
