// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"

	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

// Respond with an ActivityStreams object
func response(ctx *context.APIContext, v any) {
	binary, err := jsonld.WithContext(
		jsonld.IRI(ap.ActivityBaseURI),
		jsonld.IRI(ap.SecurityContextURI),
		jsonld.IRI(forgefed.ForgeFedNamespaceURI),
	).Marshal(v)
	if err != nil {
		ctx.ServerError("Marshal", err)
		return
	}

	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}
