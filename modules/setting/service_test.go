// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"forgejo.org/modules/structs"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/publicsuffix"
)

func match(globs []glob.Glob, s string) bool {
	for _, g := range globs {
		if g.Match(s) {
			return true
		}
	}
	return false
}

func TestLoadServices(t *testing.T) {
	oldService := Service
	defer func() {
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[service]
EMAIL_DOMAIN_WHITELIST = d1, *.w
EMAIL_DOMAIN_ALLOWLIST = d2, *.a
EMAIL_DOMAIN_BLOCKLIST = d3, *.b
`)
	require.NoError(t, err)
	loadServiceFrom(cfg)

	assert.True(t, match(Service.EmailDomainAllowList, "d1"))
	assert.True(t, match(Service.EmailDomainAllowList, "foo.w"))
	assert.True(t, match(Service.EmailDomainAllowList, "d2"))
	assert.True(t, match(Service.EmailDomainAllowList, "foo.a"))
	assert.False(t, match(Service.EmailDomainAllowList, "d3"))

	assert.True(t, match(Service.EmailDomainBlockList, "d3"))
	assert.True(t, match(Service.EmailDomainBlockList, "foo.b"))
	assert.False(t, match(Service.EmailDomainBlockList, "d1"))
}

func TestLoadServiceBlockDisposable(t *testing.T) {
	oldService := Service
	defer func() {
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[service]
EMAIL_DOMAIN_BLOCK_DISPOSABLE = true
`)

	require.NoError(t, err)
	loadServiceFrom(cfg)

	for _, domain := range DisposableEmailDomains() {
		require.True(t, match(Service.EmailDomainBlockList, domain))
	}

	require.Len(t, Service.EmailDomainBlockList, len(DisposableEmailDomains()))

	knownGood := [...]string{
		"aol.com",
		"gmx.com",
		"mail.com",
		"zoho.com",
		"proton.me",
		"gmail.com",
		"yahoo.com",
		"icloud.com",
		"outlook.com",
		"protonmail.com",
	}

	for _, domain := range knownGood {
		require.False(t, match(Service.EmailDomainBlockList, domain))
	}
}

func TestLoadServiceBlockDisposableWithExistingGlobs(t *testing.T) {
	oldService := Service
	defer func() {
		Service = oldService
	}()

	tldCounts := make(map[string]int)
	for _, domain := range DisposableEmailDomains() {
		tld, _ := publicsuffix.PublicSuffix(domain)
		tldCounts[tld]++
	}

	type tldkv struct {
		Tld   string
		Count int
	}

	sortedTldCounts := make([]tldkv, 0)
	for tld, count := range tldCounts {
		sortedTldCounts = append(sortedTldCounts, tldkv{tld, count})
	}

	sort.Slice(sortedTldCounts, func(i, j int) bool {
		return sortedTldCounts[i].Count > sortedTldCounts[j].Count
	})
	require.GreaterOrEqual(t, len(sortedTldCounts), 2)

	blockString := fmt.Sprintf("*.%s,*.%s", sortedTldCounts[0].Tld, sortedTldCounts[1].Tld)

	cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[service]
EMAIL_DOMAIN_BLOCKLIST = %s
EMAIL_DOMAIN_BLOCK_DISPOSABLE = true
`, blockString))

	require.NoError(t, err)
	loadServiceFrom(cfg)

	for _, domain := range DisposableEmailDomains() {
		require.True(t, match(Service.EmailDomainBlockList, domain))
	}

	redundant := 0
	for _, val := range DisposableEmailDomains() {
		if strings.HasSuffix(val, sortedTldCounts[0].Tld) ||
			strings.HasSuffix(val, sortedTldCounts[1].Tld) {
			redundant++
		}
	}

	expected := len(DisposableEmailDomains()) - redundant + 2
	require.Len(t, Service.EmailDomainBlockList, expected)
}

func TestLoadServiceBlockDisposableWithComplementGlobs(t *testing.T) {
	oldService := Service
	defer func() {
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[service]
EMAIL_DOMAIN_BLOCKLIST = *.random
EMAIL_DOMAIN_BLOCK_DISPOSABLE = true
`)

	require.NoError(t, err)
	loadServiceFrom(cfg)

	for _, domain := range DisposableEmailDomains() {
		require.True(t, match(Service.EmailDomainBlockList, domain))
	}

	expected := len(DisposableEmailDomains()) + 1
	require.Len(t, Service.EmailDomainBlockList, expected)
}

func TestLoadServiceVisibilityModes(t *testing.T) {
	oldService := Service
	defer func() {
		Service = oldService
	}()

	kases := map[string]func(){
		`
[service]
DEFAULT_USER_VISIBILITY = public
ALLOWED_USER_VISIBILITY_MODES = public,limited,private
`: func() {
			assert.Equal(t, "public", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypePublic, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"public", "limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
		[service]
		DEFAULT_USER_VISIBILITY = public
		`: func() {
			assert.Equal(t, "public", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypePublic, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"public", "limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
		[service]
		DEFAULT_USER_VISIBILITY = limited
		`: func() {
			assert.Equal(t, "limited", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypeLimited, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"public", "limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
[service]
ALLOWED_USER_VISIBILITY_MODES = public,limited,private
`: func() {
			assert.Equal(t, "public", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypePublic, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"public", "limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
[service]
DEFAULT_USER_VISIBILITY = public
ALLOWED_USER_VISIBILITY_MODES = limited,private
`: func() {
			assert.Equal(t, "limited", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypeLimited, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
[service]
DEFAULT_USER_VISIBILITY = my_type
ALLOWED_USER_VISIBILITY_MODES = limited,private
`: func() {
			assert.Equal(t, "limited", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypeLimited, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"limited", "private"}, Service.AllowedUserVisibilityModes)
		},
		`
[service]
DEFAULT_USER_VISIBILITY = public
ALLOWED_USER_VISIBILITY_MODES = public, limit, privated
`: func() {
			assert.Equal(t, "public", Service.DefaultUserVisibility)
			assert.Equal(t, structs.VisibleTypePublic, Service.DefaultUserVisibilityMode)
			assert.Equal(t, []string{"public"}, Service.AllowedUserVisibilityModes)
		},
	}

	for kase, fun := range kases {
		t.Run(kase, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData(kase)
			require.NoError(t, err)
			loadServiceFrom(cfg)
			fun()
			// reset
			Service.AllowedUserVisibilityModesSlice = []bool{true, true, true}
			Service.AllowedUserVisibilityModes = []string{}
			Service.DefaultUserVisibility = ""
			Service.DefaultUserVisibilityMode = structs.VisibleTypePublic
		})
	}
}
