// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
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
}
