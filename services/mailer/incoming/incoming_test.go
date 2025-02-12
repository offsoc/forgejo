// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotHandleTwice(t *testing.T) {
	handledSet := new(imap.SeqSet)
	msg := imap.NewMessage(90, []imap.FetchItem{imap.FetchBody})

	handled := isAlreadyHandled(handledSet, msg)
	assert.False(t, handled)

	handledSet.AddNum(msg.SeqNum)

	handled = isAlreadyHandled(handledSet, msg)
	assert.True(t, handled)
}

func TestIsAutomaticReply(t *testing.T) {
	cases := []struct {
		Headers  map[string]string
		Expected bool
	}{
		{
			Headers:  map[string]string{},
			Expected: false,
		},
		{
			Headers: map[string]string{
				"Auto-Submitted": "no",
			},
			Expected: false,
		},
		{
			Headers: map[string]string{
				"Auto-Submitted": "yes",
			},
			Expected: true,
		},
		{
			Headers: map[string]string{
				"X-Autoreply": "no",
			},
			Expected: false,
		},
		{
			Headers: map[string]string{
				"X-Autoreply": "yes",
			},
			Expected: true,
		},
		{
			Headers: map[string]string{
				"X-Autorespond": "yes",
			},
			Expected: true,
		},
		{
			Headers: map[string]string{
				"Precedence": "auto_reply",
			},
			Expected: true,
		},
	}

	for _, c := range cases {
		b := enmime.Builder().
			From("Dummy", "dummy@gitea.io").
			To("Dummy", "dummy@gitea.io")
		for k, v := range c.Headers {
			b = b.Header(k, v)
		}
		root, err := b.Build()
		require.NoError(t, err)
		env, err := enmime.EnvelopeFromPart(root)
		require.NoError(t, err)

		assert.Equal(t, c.Expected, isAutomaticReply(env))
	}
}

func TestGetContentFromMailReader(t *testing.T) {
	mailString := "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
		"\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: multipart/alternative; boundary=text-boundary\r\n" +
		"\r\n" +
		"--text-boundary\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		"mail content\r\n" +
		"--text-boundary--\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: attachment; filename=attachment.txt\r\n" +
		"\r\n" +
		"attachment content\r\n" +
		"--message-boundary--\r\n"

	env, err := enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content := getContentFromMailReader(env)
	assert.Equal(t, "mail content", content.Content)
	assert.Len(t, content.Attachments, 1)
	assert.Equal(t, "attachment.txt", content.Attachments[0].Name)
	assert.Equal(t, []byte("attachment content"), content.Attachments[0].Content)

	mailString = "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
		"\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: multipart/alternative; boundary=text-boundary\r\n" +
		"\r\n" +
		"--text-boundary\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		"mail content\r\n" +
		"--text-boundary--\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: inline; filename=attachment.txt\r\n" +
		"\r\n" +
		"attachment content\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: text/html\r\n" +
		"Content-Disposition: inline; filename=attachment.html\r\n" +
		"\r\n" +
		"<p>html attachment content</p>\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: image/png\r\n" +
		"Content-Disposition: inline; filename=attachment.png\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		"iVBORw0KGgoAAAANSUhEUgAAAAgAAAAIAQMAAAD+wSzIAAAABlBMVEX///+/v7+jQ3Y5AAAADklEQVQI12P4AIX8EAgALgAD/aNpbtEAAAAASUVORK5CYII\r\n" +
		"--message-boundary--\r\n"

	env, err = enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content = getContentFromMailReader(env)
	assert.Equal(t, "mail content\n--\nattachment content", content.Content)
	assert.Len(t, content.Attachments, 2)
	assert.Equal(t, "attachment.html", content.Attachments[0].Name)
	assert.Equal(t, []byte("<p>html attachment content</p>"), content.Attachments[0].Content)
	assert.Equal(t, "attachment.png", content.Attachments[1].Name)

	mailString = "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
		"\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: multipart/alternative; boundary=text-boundary\r\n" +
		"\r\n" +
		"--text-boundary\r\n" +
		"Content-Type: text/html\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		"<p>mail content</p>\r\n" +
		"--text-boundary--\r\n" +
		"--message-boundary--\r\n"

	env, err = enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content = getContentFromMailReader(env)
	assert.Equal(t, "mail content", content.Content)
	assert.Empty(t, content.Attachments)

	mailString = "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
		"\r\n" +
		"--message-boundary\r\n" +
		"Content-Type: multipart/alternative; boundary=text-boundary\r\n" +
		"\r\n" +
		"--text-boundary\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		"mail content without signature\r\n" +
		"----\r\n" +
		"signature\r\n" +
		"--text-boundary--\r\n" +
		"--message-boundary--\r\n"

	env, err = enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content = getContentFromMailReader(env)
	require.NoError(t, err)
	assert.Equal(t, "mail content without signature", content.Content)
	assert.Empty(t, content.Attachments)

	// Some versions of Outlook send inline attachments like this, inside a multipart/related part.
	// the attached image is from: https://openmoji.org/library/emoji-1F684
	mailString = "Content-Type: multipart/related; boundary=\"=_related boundary=\"\r\n" +
		"\r\n" +
		"This text is for clients unable to decode multipart/related with multipart/alternative.\r\n" +
		"\r\n" +
		"--=_related boundary=\r\n" +
		"Content-Type: multipart/alternative; boundary=\"=_alternative boundary=\"\r\n" +
		"\r\n" +
		"\r\n" +
		"\r\n" +
		"--=_alternative boundary=\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"This is the plaintext.\r\n" +
		"\r\n" +
		"--=_alternative boundary=\r\n" +
		"Content-Type: text/html\r\n" +
		"\r\n" +
		"<p>This is a mail with multipart/related. Here is an image sent with a filename.</p>\r\n" +
		"<img src=cid:_1_2845>\r\n" +
		"\r\n" +
		"--=_alternative boundary=--\r\n" +
		"\r\n" +
		"--=_related boundary=\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"Content-Type: image/png;\r\n" +
		"	name=\"image001.png\"\r\n" +
		"Content-ID: <_1_2845>\r\n" +
		"\r\n" +
		"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\r\n" +
		"exgAAAAASUVORK5CYII=\r\n" +
		"\r\n" +
		"--=_related boundary=--\r\n" +
		"\r\n"

	env, err = enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content = getContentFromMailReader(env)
	assert.Equal(t, "This is the plaintext.", content.Content)
	assert.Len(t, content.Attachments, 1)
	assert.Equal(t, "image001.png", content.Attachments[0].Name)
	expectedAttachment, err := base64.StdEncoding.DecodeString(
		"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\r\n" +
			"exgAAAAASUVORK5CYII=\r\n")
	require.NoError(t, err)
	assert.Equal(t, expectedAttachment, content.Attachments[0].Content)

	// HCL Notes inlines attachments like this: without a filename.
	// the attached image is from: https://openmoji.org/library/emoji-1F684
	mailString = "Content-Type: multipart/related; boundary=\"=_related boundary=\"\r\n" +
		"\r\n" +
		"This text is for clients unable to decode multipart/related with multipart/alternative.\r\n" +
		"\r\n" +
		"--=_related boundary=\r\n" +
		"Content-Type: multipart/alternative; boundary=\"=_alternative boundary=\"\r\n" +
		"\r\n" +
		"\r\n" +
		"\r\n" +
		"--=_alternative boundary=\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"This is the plaintext.\r\n" +
		"\r\n" +
		"--=_alternative boundary=\r\n" +
		"Content-Type: text/html\r\n" +
		"\r\n" +
		"<p>This is a mail with multipart/related. Here is an image sent without a filename.</p>\r\n" +
		"<img src=cid:_1_2845>\r\n" +
		"\r\n" +
		"--=_alternative boundary=--\r\n" +
		"\r\n" +
		"--=_related boundary=\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"Content-Type: image/png\r\n" +
		"Content-ID: <_1_2845>\r\n" +
		"\r\n" +
		"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\r\n" +
		"exgAAAAASUVORK5CYII=\r\n" +
		"\r\n" +
		"--=_related boundary=--\r\n" +
		"\r\n"

	env, err = enmime.ReadEnvelope(strings.NewReader(mailString))
	require.NoError(t, err)
	content = getContentFromMailReader(env)
	assert.Equal(t, "This is the plaintext.", content.Content)
	assert.Len(t, content.Attachments, 1)
	assert.Equal(t, "_1_2845.png", content.Attachments[0].Name)
	expectedAttachment, err = base64.StdEncoding.DecodeString(
		"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\r\n" +
			"exgAAAAASUVORK5CYII=\r\n")
	require.NoError(t, err)
	assert.Equal(t, expectedAttachment, content.Attachments[0].Content)
}
