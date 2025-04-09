// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"forgejo.org/modules/log"

	"github.com/42wim/httpsig"
)

// Federation settings
var (
	Federation = struct {
		Enabled             bool
		ShareUserStatistics bool
		MaxSize             int64
		SignatureAlgorithms []string
		DigestAlgorithm     string
		GetHeaders          []string
		PostHeaders         []string
		SignatureEnforced   bool
	}{
		Enabled:             false,
		ShareUserStatistics: true,
		MaxSize:             4,
		SignatureAlgorithms: []string{"rsa-sha256", "rsa-sha512", "ed25519"},
		DigestAlgorithm:     "SHA-256",
		GetHeaders:          []string{"(request-target)", "Date", "Host"},
		PostHeaders:         []string{"(request-target)", "Date", "Host", "Digest"},
		SignatureEnforced:   true,
	}
)

// HttpsigAlgs is a constant slice of httpsig algorithm objects
var HttpsigAlgs []httpsig.Algorithm

func loadFederationFrom(rootCfg ConfigProvider) {
	if err := rootCfg.Section("federation").MapTo(&Federation); err != nil {
		log.Fatal("Failed to map Federation settings: %v", err)
	} else if !httpsig.IsSupportedDigestAlgorithm(Federation.DigestAlgorithm) {
		log.Fatal("unsupported digest algorithm: %s", Federation.DigestAlgorithm)
		return
	}

	// Get MaxSize in bytes instead of MiB
	Federation.MaxSize = 1 << 20 * Federation.MaxSize

	HttpsigAlgs = make([]httpsig.Algorithm, len(Federation.SignatureAlgorithms))
	for i, alg := range Federation.SignatureAlgorithms {
		HttpsigAlgs[i] = httpsig.Algorithm(alg)
	}
}
