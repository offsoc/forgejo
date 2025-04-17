// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"fmt"
	"sync"

	"forgejo.org/modules/base"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/hcaptcha"
	"forgejo.org/modules/log"
	"forgejo.org/modules/mcaptcha"
	"forgejo.org/modules/recaptcha"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/turnstile"

	mc "code.forgejo.org/go-chi/cache"
	"code.forgejo.org/go-chi/captcha"
)

var (
	imageCaptchaOnce sync.Once
	imageCachePrefix = "captcha:"
)

type imageCaptchaStore struct {
	c mc.Cache
}

func (c *imageCaptchaStore) Set(id string, digits []byte) {
	if err := c.c.Put(imageCachePrefix+id, string(digits), int64(captcha.Expiration.Seconds())); err != nil {
		log.Error("Couldn't store captcha cache for %q: %v", id, err)
	}
}

func (c *imageCaptchaStore) Get(id string, clear bool) (digits []byte) {
	val, ok := c.c.Get(imageCachePrefix + id).(string)
	if !ok {
		return digits
	}

	if clear {
		if err := c.c.Delete(imageCachePrefix + id); err != nil {
			log.Error("Couldn't delete captcha cache for %q: %v", id, err)
		}
	}

	return []byte(val)
}

// GetImageCaptcha returns image captcha ID.
func GetImageCaptcha() string {
	imageCaptchaOnce.Do(func() {
		captcha.SetCustomStore(&imageCaptchaStore{c: cache.GetCache()})
	})
	return captcha.New()
}

// SetCaptchaData sets common captcha data
func SetCaptchaData(ctx *Context) {
	if !setting.Service.EnableCaptcha {
		return
	}
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha
	ctx.Data["RecaptchaURL"] = setting.Service.RecaptchaURL
	ctx.Data["Captcha"] = GetImageCaptcha()
	ctx.Data["CaptchaType"] = setting.Service.CaptchaType
	ctx.Data["RecaptchaSitekey"] = setting.Service.RecaptchaSitekey
	ctx.Data["HcaptchaSitekey"] = setting.Service.HcaptchaSitekey
	ctx.Data["McaptchaSitekey"] = setting.Service.McaptchaSitekey
	ctx.Data["McaptchaURL"] = setting.Service.McaptchaURL
	ctx.Data["CfTurnstileSitekey"] = setting.Service.CfTurnstileSitekey
}

const (
	imgCaptchaIDField        = "img-captcha-id"
	imgCaptchaResponseField  = "img-captcha-response"
	gRecaptchaResponseField  = "g-recaptcha-response"
	hCaptchaResponseField    = "h-captcha-response"
	mCaptchaResponseField    = "m-captcha-response"
	cfTurnstileResponseField = "cf-turnstile-response"
)

// VerifyCaptcha verifies Captcha data
// No-op if captchas are not enabled
func VerifyCaptcha(ctx *Context, tpl base.TplName, form any) {
	if !setting.Service.EnableCaptcha {
		return
	}

	var valid bool
	var err error
	switch setting.Service.CaptchaType {
	case setting.ImageCaptcha:
		valid = captcha.VerifyString(ctx.Req.Form.Get(imgCaptchaIDField), ctx.Req.Form.Get(imgCaptchaResponseField))
	case setting.ReCaptcha:
		valid, err = recaptcha.Verify(ctx, ctx.Req.Form.Get(gRecaptchaResponseField))
	case setting.HCaptcha:
		valid, err = hcaptcha.Verify(ctx, ctx.Req.Form.Get(hCaptchaResponseField))
	case setting.MCaptcha:
		valid, err = mcaptcha.Verify(ctx, ctx.Req.Form.Get(mCaptchaResponseField))
	case setting.CfTurnstile:
		valid, err = turnstile.Verify(ctx, ctx.Req.Form.Get(cfTurnstileResponseField))
	default:
		ctx.ServerError("Unknown Captcha Type", fmt.Errorf("unknown Captcha Type: %s", setting.Service.CaptchaType))
		return
	}
	if err != nil {
		log.Debug("Captcha Verify failed: %v", err)
	}

	if !valid {
		ctx.Data["Err_Captcha"] = true
		ctx.RenderWithErr(ctx.Tr("form.captcha_incorrect"), tpl, form)
	}
}
