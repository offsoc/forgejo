// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"net/http"
	"runtime/trace"
	"strings"

	"forgejo.org/modules/cache"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web/middleware"
	"forgejo.org/modules/web/routing"
	"forgejo.org/services/context"

	"code.forgejo.org/go-chi/session"
	"github.com/chi-middleware/proxy"
	chi "github.com/go-chi/chi/v5"
)

// ProtocolMiddlewares returns HTTP protocol related middlewares, and it provides a global panic recovery
func ProtocolMiddlewares() (handlers []any) {
	// first, normalize the URL path
	handlers = append(handlers, stripSlashesMiddleware)

	// prepare the ContextData and panic recovery
	handlers = append(handlers, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					RenderPanicErrorPage(resp, req, err) // it should never panic
				}
			}()
			req = req.WithContext(middleware.WithContextData(req.Context()))
			next.ServeHTTP(resp, req)
		})
	})

	// wrap the request and response, use the process context and add it to the process manager
	handlers = append(handlers, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			ctx, _, finished := process.GetManager().AddTypedContext(req.Context(), fmt.Sprintf("%s: %s", req.Method, req.RequestURI), process.RequestProcessType, true)
			defer finished()
			trace.Log(ctx, "method", req.Method)
			trace.Log(ctx, "url", req.RequestURI)
			next.ServeHTTP(context.WrapResponseWriter(resp), req.WithContext(cache.WithCacheContext(ctx)))
		})
	})

	if setting.ReverseProxyLimit > 0 {
		opt := proxy.NewForwardedHeadersOptions().
			WithForwardLimit(setting.ReverseProxyLimit).
			ClearTrustedProxies()
		for _, n := range setting.ReverseProxyTrustedProxies {
			if !strings.Contains(n, "/") {
				opt.AddTrustedProxy(n)
			} else {
				opt.AddTrustedNetwork(n)
			}
		}
		handlers = append(handlers, proxy.ForwardedHeaders(opt))
	}

	if setting.IsRouteLogEnabled() {
		handlers = append(handlers, routing.NewLoggerHandler())
	}

	if setting.IsAccessLogEnabled() {
		handlers = append(handlers, context.AccessLogger())
	}

	return handlers
}

func stripSlashesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Ensure that URL.RawPath is always set.
		req.URL.RawPath = req.URL.EscapedPath()

		sanitize := func(path string) string {
			sanitizedPath := &strings.Builder{}
			prevWasSlash := false
			for _, chr := range strings.TrimRight(path, "/") {
				if chr != '/' || !prevWasSlash {
					sanitizedPath.WriteRune(chr)
				}
				prevWasSlash = chr == '/'
			}
			return sanitizedPath.String()
		}

		// Sanitize the unescaped path for application logic.
		req.URL.Path = sanitize(req.URL.Path)
		rctx := chi.RouteContext(req.Context())
		if rctx != nil {
			// Sanitize the escaped path for routing.
			rctx.RoutePath = sanitize(req.URL.RawPath)
		}
		next.ServeHTTP(resp, req)
	})
}

func Sessioner() func(next http.Handler) http.Handler {
	return session.Sessioner(session.Options{
		Provider:       setting.SessionConfig.Provider,
		ProviderConfig: setting.SessionConfig.ProviderConfig,
		CookieName:     setting.SessionConfig.CookieName,
		CookiePath:     setting.SessionConfig.CookiePath,
		Gclifetime:     setting.SessionConfig.Gclifetime,
		Maxlifetime:    setting.SessionConfig.Maxlifetime,
		Secure:         setting.SessionConfig.Secure,
		SameSite:       setting.SessionConfig.SameSite,
		Domain:         setting.SessionConfig.Domain,
	})
}
