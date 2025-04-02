// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseTagData(t *testing.T) {
	testData := []struct {
		data []byte
		tag  Tag
	}{
		{data: []byte(`object 3b114ab800c6432ad42387ccf6bc8d4388a2885a
type commit
tag 1.22.0
tagger Lucas Michot <lucas@semalead.com> 1484491741 +0100

`), tag: Tag{
			Name:      "",
			ID:        Sha1ObjectFormat.EmptyObjectID(),
			Object:    &Sha1Hash{0x3b, 0x11, 0x4a, 0xb8, 0x0, 0xc6, 0x43, 0x2a, 0xd4, 0x23, 0x87, 0xcc, 0xf6, 0xbc, 0x8d, 0x43, 0x88, 0xa2, 0x88, 0x5a},
			Type:      "commit",
			Tagger:    &Signature{Name: "Lucas Michot", Email: "lucas@semalead.com", When: time.Unix(1484491741, 0)},
			Message:   "",
			Signature: nil,
		}},
		{data: []byte(`object 7cdf42c0b1cc763ab7e4c33c47a24e27c66bfccc
type commit
tag 1.22.1
tagger Lucas Michot <lucas@semalead.com> 1484553735 +0100

test message
o

ono`), tag: Tag{
			Name:      "",
			ID:        Sha1ObjectFormat.EmptyObjectID(),
			Object:    &Sha1Hash{0x7c, 0xdf, 0x42, 0xc0, 0xb1, 0xcc, 0x76, 0x3a, 0xb7, 0xe4, 0xc3, 0x3c, 0x47, 0xa2, 0x4e, 0x27, 0xc6, 0x6b, 0xfc, 0xcc},
			Type:      "commit",
			Tagger:    &Signature{Name: "Lucas Michot", Email: "lucas@semalead.com", When: time.Unix(1484553735, 0)},
			Message:   "test message\no\n\nono",
			Signature: nil,
		}},
		{data: []byte(`object d8d1fdb5b20eaca882e34ee510eb55941a242b24
type commit
tag v0
tagger Jane Doe <jane.doe@example.com> 1709146405 +0100

v0
-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgvD4pK7baygXxoWoVoKjVEc/xZh
6w+1FUn5hypFqJXNAAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQKFeTnxi9ssRqSg+sJcmjAgpgoPq1k5SXm306+mJmkPwvhim8f9Gz6uy1AddPmXaD7
5LVB3fV2GmmFDKGB+wCAo=
-----END SSH SIGNATURE-----
`), tag: Tag{
			Name:    "",
			ID:      Sha1ObjectFormat.EmptyObjectID(),
			Object:  &Sha1Hash{0xd8, 0xd1, 0xfd, 0xb5, 0xb2, 0x0e, 0xac, 0xa8, 0x82, 0xe3, 0x4e, 0xe5, 0x10, 0xeb, 0x55, 0x94, 0x1a, 0x24, 0x2b, 0x24},
			Type:    "commit",
			Tagger:  &Signature{Name: "Jane Doe", Email: "jane.doe@example.com", When: time.Unix(1709146405, 0)},
			Message: "v0\n",
			Signature: &ObjectSignature{
				Signature: `-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgvD4pK7baygXxoWoVoKjVEc/xZh
6w+1FUn5hypFqJXNAAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQKFeTnxi9ssRqSg+sJcmjAgpgoPq1k5SXm306+mJmkPwvhim8f9Gz6uy1AddPmXaD7
5LVB3fV2GmmFDKGB+wCAo=
-----END SSH SIGNATURE-----`,
				Payload: `object d8d1fdb5b20eaca882e34ee510eb55941a242b24
type commit
tag v0
tagger Jane Doe <jane.doe@example.com> 1709146405 +0100

v0
`,
			},
		}},
	}

	for _, test := range testData {
		tag, err := parseTagData(Sha1ObjectFormat, test.data)
		require.NoError(t, err)
		assert.Equal(t, test.tag.ID, tag.ID)
		assert.Equal(t, test.tag.Object, tag.Object)
		assert.Equal(t, test.tag.Name, tag.Name)
		assert.Equal(t, test.tag.Message, tag.Message)
		assert.Equal(t, test.tag.Type, tag.Type)
		if test.tag.Signature != nil && assert.NotNil(t, tag.Signature) {
			assert.Equal(t, test.tag.Signature.Signature, tag.Signature.Signature)
			assert.Equal(t, test.tag.Signature.Payload, tag.Signature.Payload)
		} else {
			assert.Nil(t, tag.Signature)
		}
		if test.tag.Tagger != nil && assert.NotNil(t, tag.Tagger) {
			assert.Equal(t, test.tag.Tagger.Name, tag.Tagger.Name)
			assert.Equal(t, test.tag.Tagger.Email, tag.Tagger.Email)
			assert.Equal(t, test.tag.Tagger.When.Unix(), tag.Tagger.When.Unix())
		} else {
			assert.Nil(t, tag.Tagger)
		}
	}
}
