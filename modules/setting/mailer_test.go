// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadMailerFrom(t *testing.T) {
	kases := map[string]*Mailer{
		"smtp.mydomain.com": {
			SMTPAddr: "smtp.mydomain.com",
			SMTPPort: "465",
		},
		"smtp.mydomain.com:123": {
			SMTPAddr: "smtp.mydomain.com",
			SMTPPort: "123",
		},
		":123": {
			SMTPAddr: "127.0.0.1",
			SMTPPort: "123",
		},
	}
	for host, kase := range kases {
		t.Run(host, func(t *testing.T) {
			cfg, _ := NewConfigProviderFromData("")
			sec := cfg.Section("mailer")
			sec.NewKey("ENABLED", "true")
			sec.NewKey("HOST", host)

			// Check mailer setting
			loadMailerFrom(cfg)

			assert.Equal(t, kase.SMTPAddr, MailService.SMTPAddr)
			assert.Equal(t, kase.SMTPPort, MailService.SMTPPort)
		})
	}

	t.Run("property aliases", func(t *testing.T) {
		cfg, _ := NewConfigProviderFromData("")
		sec := cfg.Section("mailer")
		sec.NewKey("ENABLED", "true")
		sec.NewKey("USERNAME", "jane.doe@example.com")
		sec.NewKey("PASSWORD", "y0u'll n3v3r gUess th1S!!1")

		loadMailerFrom(cfg)

		assert.Equal(t, "jane.doe@example.com", MailService.User)
		assert.Equal(t, "y0u'll n3v3r gUess th1S!!1", MailService.Passwd)
	})

	t.Run("sendmail argument sanitization", func(t *testing.T) {
		cfg, _ := NewConfigProviderFromData("")
		sec := cfg.Section("mailer")
		sec.NewKey("ENABLED", "true")
		sec.NewKey("PROTOCOL", "sendmail")
		sec.NewKey("SENDMAIL_ARGS", "-B 8BITMIME")

		loadMailerFrom(cfg)

		assert.Equal(t, []string{"-B", "8BITMIME", "--"}, MailService.SendmailArgs)
	})

	t.Run("empty sendmail args", func(t *testing.T) {
		cfg, _ := NewConfigProviderFromData("")
		sec := cfg.Section("mailer")
		sec.NewKey("ENABLED", "true")
		sec.NewKey("PROTOCOL", "sendmail")
		sec.NewKey("SENDMAIL_ARGS", "")

		loadMailerFrom(cfg)

		assert.Equal(t, []string{"--"}, MailService.SendmailArgs)
	})
}
