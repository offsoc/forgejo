// Copyright 2021 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"strings"
	"testing"
	"time"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMessageID(t *testing.T) {
	defer test.MockVariableValue(&setting.MailService, &setting.Mailer{
		From: "test@gitea.com",
	})()
	defer test.MockVariableValue(&setting.Domain, "localhost")()

	date := time.Date(2000, 1, 2, 3, 4, 5, 6, time.UTC)
	m := NewMessageFrom("", "display-name", "from-address", "subject", "body")
	m.Date = date
	gm := m.ToMessage()
	assert.Equal(t, "<autogen-946782245000-41e8fc54a8ad3a3f@localhost>", gm.GetHeader("Message-ID")[0])

	m = NewMessageFrom("a@b.com", "display-name", "from-address", "subject", "body")
	m.Date = date
	gm = m.ToMessage()
	assert.Equal(t, "<autogen-946782245000-cc88ce3cfe9bd04f@localhost>", gm.GetHeader("Message-ID")[0])

	m = NewMessageFrom("a@b.com", "display-name", "from-address", "subject", "body")
	m.SetHeader("Message-ID", "<msg-d@domain.com>")
	gm = m.ToMessage()
	assert.Equal(t, "<msg-d@domain.com>", gm.GetHeader("Message-ID")[0])
}

func TestGenerateMessageIDForRelease(t *testing.T) {
	defer test.MockVariableValue(&setting.Domain, "localhost")()

	rel := repo_model.Release{
		ID: 42,
		Repo: &repo_model.Repository{
			OwnerName: "test",
			Name:      "tag-test",
		},
	}
	m := createMessageIDForRelease(&rel)
	assert.Equal(t, "<test/tag-test/releases/42@localhost>", m)
}

func TestToMessage(t *testing.T) {
	defer test.MockVariableValue(&setting.MailService, &setting.Mailer{
		From: "test@gitea.com",
	})()
	defer test.MockVariableValue(&setting.Domain, "localhost")()

	m1 := Message{
		Info:            "info",
		FromAddress:     "test@gitea.com",
		FromDisplayName: "Test Gitea",
		To:              "a@b.com",
		Subject:         "Issue X Closed",
		Body:            "Some Issue got closed by Y-Man",
	}

	buf := &strings.Builder{}
	_, err := m1.ToMessage().WriteTo(buf)
	require.NoError(t, err)
	header, _ := extractMailHeaderAndContent(t, buf.String())
	assert.Equal(t, map[string]string{
		"Content-Type":             "multipart/alternative;",
		"Date":                     "Mon, 01 Jan 0001 00:00:00 +0000",
		"From":                     "\"Test Gitea\" <test@gitea.com>",
		"Message-ID":               "<autogen--6795364578871-69c000786adc60dc@localhost>",
		"Mime-Version":             "1.0",
		"Subject":                  "Issue X Closed",
		"To":                       "a@b.com",
		"X-Auto-Response-Suppress": "All",
	}, header)

	setting.MailService.OverrideHeader = map[string][]string{
		"Message-ID":     {""},               // delete message id
		"Auto-Submitted": {"auto-generated"}, // suppress auto replay
	}

	buf = &strings.Builder{}
	_, err = m1.ToMessage().WriteTo(buf)
	require.NoError(t, err)
	header, _ = extractMailHeaderAndContent(t, buf.String())
	assert.Equal(t, map[string]string{
		"Content-Type":             "multipart/alternative;",
		"Date":                     "Mon, 01 Jan 0001 00:00:00 +0000",
		"From":                     "\"Test Gitea\" <test@gitea.com>",
		"Message-ID":               "",
		"Mime-Version":             "1.0",
		"Subject":                  "Issue X Closed",
		"To":                       "a@b.com",
		"X-Auto-Response-Suppress": "All",
		"Auto-Submitted":           "auto-generated",
	}, header)
}

func extractMailHeaderAndContent(t *testing.T, mail string) (map[string]string, string) {
	header := make(map[string]string)

	parts := strings.SplitN(mail, "boundary=", 2)
	if !assert.Len(t, parts, 2) {
		return nil, ""
	}
	content := strings.TrimSpace("boundary=" + parts[1])

	hParts := strings.Split(parts[0], "\n")

	for _, hPart := range hParts {
		parts := strings.SplitN(hPart, ":", 2)
		hk := strings.TrimSpace(parts[0])
		if hk != "" {
			header[hk] = strings.TrimSpace(parts[1])
		}
	}

	return header, content
}
