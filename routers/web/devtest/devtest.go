// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package devtest

import (
	"errors"
	"net/http"
	"path"
	"strings"
	"time"

	"forgejo.org/models/asymkey"
	"forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/templates"
	"forgejo.org/services/context"
)

// List all devtest templates, they will be used for e2e tests for the UI components
func List(ctx *context.Context) {
	templateNames, err := templates.AssetFS().ListFiles("devtest", true)
	if err != nil {
		ctx.ServerError("AssetFS().ListFiles", err)
		return
	}
	var subNames []string
	for _, tmplName := range templateNames {
		subName := strings.TrimSuffix(tmplName, ".tmpl")
		if subName != "list" {
			subNames = append(subNames, subName)
		}
	}
	ctx.Data["SubNames"] = subNames
	ctx.HTML(http.StatusOK, "devtest/list")
}

func FetchActionTest(ctx *context.Context) {
	_ = ctx.Req.ParseForm()
	ctx.Flash.Info("fetch-action: " + ctx.Req.Method + " " + ctx.Req.RequestURI + "<br>" +
		"Form: " + ctx.Req.Form.Encode() + "<br>" +
		"PostForm: " + ctx.Req.PostForm.Encode(),
	)
	time.Sleep(2 * time.Second)
	ctx.JSONRedirect("")
}

func ErrorPage(ctx *context.Context) {
	if ctx.Params("errcode") == "404" {
		ctx.NotFound("Example error", errors.New("Example error"))
		return
	} else if ctx.Params("errcode") == "413" {
		ctx.HTML(http.StatusRequestEntityTooLarge, base.TplName("status/413"))
		return
	}
	ctx.ServerError("Example error", errors.New("Example error"))
}

func Tmpl(ctx *context.Context) {
	now := time.Now()
	ctx.Data["TimeNow"] = now
	ctx.Data["TimePast5s"] = now.Add(-5 * time.Second)
	ctx.Data["TimeFuture5s"] = now.Add(5 * time.Second)
	ctx.Data["TimePast2m"] = now.Add(-2 * time.Minute)
	ctx.Data["TimeFuture2m"] = now.Add(2 * time.Minute)
	ctx.Data["TimePast1y"] = now.Add(-1 * 366 * 86400 * time.Second)
	ctx.Data["TimeFuture1y"] = now.Add(1 * 366 * 86400 * time.Second)

	userNonZero := &user.User{ID: 1}
	ctx.Data["TrustedVerif"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userNonZero, TrustStatus: "trusted"}
	ctx.Data["UntrustedVerif"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userNonZero, TrustStatus: "untrusted"}
	ctx.Data["UnmatchedVerif"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userNonZero, TrustStatus: ""}
	ctx.Data["WarnVerif"] = &asymkey.ObjectVerification{Verified: false, Warning: true, Reason: asymkey.NotSigned, SigningUser: userNonZero}
	ctx.Data["UnknownVerif"] = &asymkey.ObjectVerification{Verified: false, Warning: false, Reason: asymkey.NotSigned, SigningUser: userNonZero}
	userUnknown := &user.User{ID: 0}
	ctx.Data["TrustedVerifUnk"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userUnknown, TrustStatus: "trusted"}
	ctx.Data["UntrustedVerifUnk"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userUnknown, TrustStatus: "untrusted"}
	ctx.Data["UnmatchedVerifUnk"] = &asymkey.ObjectVerification{Verified: true, Reason: asymkey.NotSigned, SigningUser: userUnknown, TrustStatus: ""}
	ctx.Data["WarnVerifUnk"] = &asymkey.ObjectVerification{Verified: false, Warning: true, Reason: asymkey.NotSigned, SigningUser: userUnknown}
	ctx.Data["UnknownVerifUnk"] = &asymkey.ObjectVerification{Verified: false, Warning: false, Reason: asymkey.NotSigned, SigningUser: userUnknown}

	if ctx.Req.Method == "POST" {
		_ = ctx.Req.ParseForm()
		ctx.Flash.Info("form: "+ctx.Req.Method+" "+ctx.Req.RequestURI+"<br>"+
			"Form: "+ctx.Req.Form.Encode()+"<br>"+
			"PostForm: "+ctx.Req.PostForm.Encode(),
			true,
		)
		time.Sleep(2 * time.Second)
	}

	ctx.HTML(http.StatusOK, base.TplName("devtest"+path.Clean("/"+ctx.Params("sub"))))
}
