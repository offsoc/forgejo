// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package opentelemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"

	"github.com/go-logr/logr/funcr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func Init(ctx context.Context) error {
	// Redirect otel logger to write to common forgejo log at info
	logWrap := funcr.New(func(prefix, args string) {
		log.Info(fmt.Sprint(prefix, args))
	}, funcr.Options{})
	otel.SetLogger(logWrap)
	// Redirect error handling to forgejo log as well
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(cause error) {
		log.Error("internal opentelemetry error was raised: %s", cause)
	}))
	var shutdownFuncs []func(context.Context) error
	shutdownCtx := context.Background()

	otel.SetTextMapPropagator(newPropagator())

	res, err := newResource(ctx)
	if err != nil {
		return err
	}

	traceShutdown, err := setupTraceProvider(ctx, res)
	if err != nil {
		log.Warn("OpenTelemetry trace setup failed, err=%s", err)
	} else {
		shutdownFuncs = append(shutdownFuncs, traceShutdown)
	}

	graceful.GetManager().RunAtShutdown(ctx, func() {
		for _, fn := range shutdownFuncs {
			if err := fn(shutdownCtx); err != nil {
				log.Warn("exporter shutdown failed, err=%s", err)
			}
		}
		shutdownFuncs = nil
	})

	return nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func withCertPool(path string, tlsConf *tls.Config) {
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		log.Warn("Otel: reading ca cert failed path=%s, err=%s", path, err)
		return
	}
	cp := x509.NewCertPool()
	if ok := cp.AppendCertsFromPEM(b); !ok {
		log.Warn("Otel: no valid PEM certificate found path=%s", path)
		return
	}
	tlsConf.RootCAs = cp
}

func withClientCert(nc, nk string, tlsConf *tls.Config) {
	if nc == "" || nk == "" {
		return
	}

	crt, err := tls.LoadX509KeyPair(nc, nk)
	if err != nil {
		log.Warn("Otel: create tls client key pair failed")
		return
	}

	tlsConf.Certificates = append(tlsConf.Certificates, crt)
}
