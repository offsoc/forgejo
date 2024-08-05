// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package opentelemetry

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestTraceGrpcExporter(t *testing.T) {
	grpcMethods := make(chan string)
	tlsConfig := generateTestTLS(t, os.TempDir(), "localhost,127.0.0.1")
	assert.NotNil(t, tlsConfig)

	collector := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)), grpc.UnknownServiceHandler(func(srv any, stream grpc.ServerStream) error {
		method, _ := grpc.Method(stream.Context())
		grpcMethods <- method
		return nil
	}))
	defer collector.GracefulStop()
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer ln.Close()
	go collector.Serve(ln)

	traceEndpoint, err := url.Parse("https://" + ln.Addr().String())
	require.NoError(t, err)
	config := &setting.OtelExporter{
		Endpoint:          traceEndpoint,
		Certificate:       os.TempDir() + "/cert.pem",
		ClientCertificate: os.TempDir() + "/cert.pem",
		ClientKey:         os.TempDir() + "/key.pem",
		Protocol:          "grpc",
	}

	defer test.MockVariableValue(&setting.OpenTelemetry.ServiceName, "forgejo-certs")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Enabled, true)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces, "otlp")()
	defer test.MockVariableValue(&setting.OpenTelemetry.OtelTraces, config)()
	ctx := context.Background()
	require.NoError(t, Init(ctx))

	tracer := otel.Tracer("test_tls")
	_, span := tracer.Start(ctx, "test span")
	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	// Give the exporter time to send the span
	select {
	case method := <-grpcMethods:
		assert.Equal(t, "/opentelemetry.proto.collector.trace.v1.TraceService/Export", method)
	case <-time.After(10 * time.Second):
		t.Fatal("no grpc call within 10s")
	}
}

func TestTraceHttpExporter(t *testing.T) {
	httpCalls := make(chan string)
	tlsConfig := generateTestTLS(t, os.TempDir(), "localhost,127.0.0.1")
	assert.NotNil(t, tlsConfig)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalls <- r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	server.TLS = tlsConfig

	traceEndpoint, err := url.Parse("http://" + server.Listener.Addr().String())
	require.NoError(t, err)
	config := &setting.OtelExporter{
		Endpoint:          traceEndpoint,
		Certificate:       os.TempDir() + "/cert.pem",
		ClientCertificate: os.TempDir() + "/cert.pem",
		ClientKey:         os.TempDir() + "/key.pem",
		Protocol:          "http/protobuf",
	}

	defer test.MockVariableValue(&setting.OpenTelemetry.ServiceName, "forgejo-certs")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Enabled, true)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces, "otlp")()
	defer test.MockVariableValue(&setting.OpenTelemetry.OtelTraces, config)()
	ctx := context.Background()
	require.NoError(t, Init(ctx))

	tracer := otel.Tracer("test_tls")
	_, span := tracer.Start(ctx, "test span")
	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	select {
	case path := <-httpCalls:
		assert.Equal(t, "/v1/traces", path)
	case <-time.After(10 * time.Second):
		t.Fatal("no http call within 10s")
	}
}
