// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadIncomingEmailFrom(t *testing.T) {
	makeBaseConfig := func() (ConfigProvider, ConfigSection) {
		cfg, _ := NewConfigProviderFromData("")
		sec := cfg.Section("email.incoming")
		sec.NewKey("ENABLED", "true")
		sec.NewKey("REPLY_TO_ADDRESS", "forge+%{token}@example.com")

		return cfg, sec
	}
	resetIncomingEmailPort := func() func() {
		return func() {
			IncomingEmail.Port = 0
		}
	}

	t.Run("aliases", func(t *testing.T) {
		cfg, sec := makeBaseConfig()
		sec.NewKey("USER", "jane.doe@example.com")
		sec.NewKey("PASSWD", "y0u'll n3v3r gUess th1S!!1")

		loadIncomingEmailFrom(cfg)

		assert.Equal(t, "jane.doe@example.com", IncomingEmail.Username)
		assert.Equal(t, "y0u'll n3v3r gUess th1S!!1", IncomingEmail.Password)
	})

	t.Run("Port settings", func(t *testing.T) {
		t.Run("no port, no tls", func(t *testing.T) {
			defer resetIncomingEmailPort()()
			cfg, sec := makeBaseConfig()

			// False is the default, but we test it explicitly.
			sec.NewKey("USE_TLS", "false")

			loadIncomingEmailFrom(cfg)

			assert.Equal(t, 143, IncomingEmail.Port)
		})

		t.Run("no port, with tls", func(t *testing.T) {
			defer resetIncomingEmailPort()()
			cfg, sec := makeBaseConfig()

			sec.NewKey("USE_TLS", "true")

			loadIncomingEmailFrom(cfg)

			assert.Equal(t, 993, IncomingEmail.Port)
		})

		t.Run("port overrides tls", func(t *testing.T) {
			defer resetIncomingEmailPort()()
			cfg, sec := makeBaseConfig()

			sec.NewKey("PORT", "1993")
			sec.NewKey("USE_TLS", "true")

			loadIncomingEmailFrom(cfg)

			assert.Equal(t, 1993, IncomingEmail.Port)
		})
	})
}
