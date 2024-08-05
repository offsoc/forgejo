// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package opentelemetry

import (
	"context"
	"crypto/tls"

	"code.gitea.io/gitea/modules/setting"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

func newGrpcExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	endpoint := setting.OpenTelemetry.OtelTraces.Endpoint

	opts := []otlptracegrpc.Option{}

	tlsConf := &tls.Config{}
	opts = append(opts, otlptracegrpc.WithEndpoint(endpoint.Host))
	opts = append(opts, otlptracegrpc.WithTimeout(setting.OpenTelemetry.OtelTraces.Timeout))
	switch setting.OpenTelemetry.OtelTraces.Endpoint.Scheme {
	case "http", "unix":
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if setting.OpenTelemetry.OtelTraces.Compression != "" {
		opts = append(opts, otlptracegrpc.WithCompressor(setting.OpenTelemetry.OtelTraces.Compression))
	}
	withCertPool(setting.OpenTelemetry.OtelTraces.Certificate, tlsConf)
	withClientCert(setting.OpenTelemetry.OtelTraces.ClientCertificate, setting.OpenTelemetry.OtelTraces.ClientKey, tlsConf)
	if tlsConf.RootCAs != nil || len(tlsConf.Certificates) > 0 {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(
			credentials.NewTLS(tlsConf),
		))
	}
	opts = append(opts, otlptracegrpc.WithHeaders(setting.OpenTelemetry.OtelTraces.Headers))

	return otlptracegrpc.New(ctx, opts...)
}

func newHTTPExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	endpoint := setting.OpenTelemetry.OtelTraces.Endpoint
	opts := []otlptracehttp.Option{}
	tlsConf := &tls.Config{}
	opts = append(opts, otlptracehttp.WithEndpoint(endpoint.Host))
	switch setting.OpenTelemetry.OtelTraces.Endpoint.Scheme {
	case "http", "unix":
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	switch setting.OpenTelemetry.OtelTraces.Compression {
	case "gzip":
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	default:
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
	}
	withCertPool(setting.OpenTelemetry.OtelTraces.Certificate, tlsConf)
	withClientCert(setting.OpenTelemetry.OtelTraces.ClientCertificate, setting.OpenTelemetry.OtelTraces.ClientKey, tlsConf)
	if tlsConf.RootCAs != nil || len(tlsConf.Certificates) > 0 {
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConf))
	}
	opts = append(opts, otlptracehttp.WithHeaders(setting.OpenTelemetry.OtelTraces.Headers))

	return otlptracehttp.New(ctx, opts...)
}

var exporter = map[string]func(context.Context) (sdktrace.SpanExporter, error){
	"http/protobuf": newHTTPExporter,
	"grpc":          newGrpcExporter,
}

// Create new and register trace provider from user defined configuration
func setupTraceProvider(ctx context.Context, r *resource.Resource) (func(context.Context) error, error) {
	var shutdown func(context.Context) error
	switch setting.OpenTelemetry.Traces {
	case "otlp":
		traceExporter, err := exporter[setting.OpenTelemetry.OtelTraces.Protocol](ctx)
		if err != nil {
			return nil, err
		}
		traceProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(setting.OpenTelemetry.Sampler),
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(r),
		)
		otel.SetTracerProvider(traceProvider)
		shutdown = traceProvider.Shutdown
	default:
		shutdown = func(ctx context.Context) error { return nil }
	}
	return shutdown, nil
}
