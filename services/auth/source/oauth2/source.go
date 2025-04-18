// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"strings"

	"forgejo.org/models/auth"
	"forgejo.org/modules/json"
)

// Source holds configuration for the OAuth2 login source.
type Source struct {
	Provider                      string
	ClientID                      string
	ClientSecret                  string
	OpenIDConnectAutoDiscoveryURL string
	CustomURLMapping              *CustomURLMapping
	IconURL                       string

	Scopes                []string
	AttributeSSHPublicKey string
	RequiredClaimName     string
	RequiredClaimValue    string
	GroupClaimName        string
	AdminGroup            string
	GroupTeamMap          string
	GroupTeamMapRemoval   bool
	RestrictedGroup       string
	SkipLocalTwoFA        bool `json:",omitempty"`

	// reference to the authSource
	authSource *auth.Source
}

// FromDB fills up an OAuth2Config from serialized format.
func (source *Source) FromDB(bs []byte) error {
	return json.UnmarshalHandleDoubleEncode(bs, &source)
}

// ToDB exports an SMTPConfig to a serialized format.
func (source *Source) ToDB() ([]byte, error) {
	return json.Marshal(source)
}

// ProvidesSSHKeys returns if this source provides SSH Keys
func (source *Source) ProvidesSSHKeys() bool {
	return len(strings.TrimSpace(source.AttributeSSHPublicKey)) > 0
}

// SetAuthSource sets the related AuthSource
func (source *Source) SetAuthSource(authSource *auth.Source) {
	source.authSource = authSource
}

func init() {
	auth.RegisterTypeConfig(auth.OAuth2, &Source{})
}
