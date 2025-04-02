// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package remote

import (
	"forgejo.org/models/auth"
	"forgejo.org/modules/json"
)

type Source struct {
	URL            string
	MatchingSource string

	// reference to the authSource
	authSource *auth.Source
}

func (source *Source) FromDB(bs []byte) error {
	return json.UnmarshalHandleDoubleEncode(bs, &source)
}

func (source *Source) ToDB() ([]byte, error) {
	return json.Marshal(source)
}

func (source *Source) SetAuthSource(authSource *auth.Source) {
	source.authSource = authSource
}

func init() {
	auth.RegisterTypeConfig(auth.Remote, &Source{})
}
