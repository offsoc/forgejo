// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	gitea_context "forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
)

func verifyHTTPUserOrInstanceSignature(ctx *gitea_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		return false, err
	}

	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateFederatedUserKey(ctx.Base, v.KeyId())
	if err != nil || pubKey == nil {
		pubKey, err = federation.FindOrCreateFederationHostKey(ctx.Base, v.KeyId())
		if err != nil {
			return false, err
		}
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		return false, err
	}
	return true, nil
}

func verifyHTTPUserSignature(ctx *gitea_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		return false, err
	}

	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateFederatedUserKey(ctx.Base, v.KeyId())
	if err != nil {
		return false, err
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReqHTTPSignature function
func ReqHTTPUserOrInstanceSignature() func(ctx *gitea_context.APIContext) {
	return func(ctx *gitea_context.APIContext) {
		if authenticated, err := verifyHTTPUserOrInstanceSignature(ctx); err != nil {
			log.Warn("verifyHttpSignatures failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}

// ReqHTTPSignature function
func ReqHTTPUserSignature() func(ctx *gitea_context.APIContext) {
	return func(ctx *gitea_context.APIContext) {
		if authenticated, err := verifyHTTPUserSignature(ctx); err != nil {
			log.Warn("verifyHttpSignatures failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}
