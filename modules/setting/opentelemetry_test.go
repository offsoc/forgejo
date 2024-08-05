// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package setting

import (
	"net/url"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestExporterLoad(t *testing.T) {
	globalSetting := `
	[opentelemetry.exporter.otlp]
ENDPOINT=http://example.org:4318/
CERTIFICATE=/boo/bar
CLIENT_CERTIFICATE=/foo/bar
CLIENT_KEY=/bar/bar
COMPRESSION=
HEADERS=key=val,val=key
PROTOCOL=http/protobuf
TIMEOUT=20s
	`
	endpoint, err := url.Parse("http://example.org:4318/")
	require.NoError(t, err)
	expected := &OtelExporter{
		Endpoint:          endpoint,
		Certificate:       "/boo/bar",
		ClientCertificate: "/foo/bar",
		ClientKey:         "/bar/bar",
		Headers: map[string]string{
			"key": "val", "val": "key",
		},
		Timeout:  20 * time.Second,
		Protocol: "http/protobuf",
	}
	cfg, err := NewConfigProviderFromData(globalSetting)
	require.NoError(t, err)
	exp := createOtlpExporterConfig(cfg, ".traces")
	assert.Equal(t, expected, exp)
	localSetting := `
[opentelemetry.exporter.otlp.traces]
ENDPOINT=http://example.com:4318/
CERTIFICATE=/boo
CLIENT_CERTIFICATE=/foo
CLIENT_KEY=/bar
COMPRESSION=gzip
HEADERS=key=val2,val1=key
PROTOCOL=grpc
TIMEOUT=5s
	`
	endpoint, err = url.Parse("http://example.com:4318/")
	require.NoError(t, err)
	expected = &OtelExporter{
		Endpoint:          endpoint,
		Certificate:       "/boo",
		ClientCertificate: "/foo",
		ClientKey:         "/bar",
		Compression:       "gzip",
		Headers: map[string]string{
			"key": "val2", "val1": "key", "val": "key",
		},
		Timeout:  5 * time.Second,
		Protocol: "grpc",
	}

	cfg, err = NewConfigProviderFromData(globalSetting + localSetting)
	require.NoError(t, err)
	exp = createOtlpExporterConfig(cfg, ".traces")
	require.NoError(t, err)
	assert.Equal(t, expected, exp)
}

func TestOpenTelemetryConfiguration(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.Nil(t, OpenTelemetry.OtelTraces)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	ENABLED=true
	SERVICE_NAME = test service
	RESOURCE_ATTRIBUTES = foo=bar
	TRACES_SAMPLER = always_on

	[opentelemetry.exporter.otlp]
	ENDPOINT = http://jaeger:4317/
	TIMEOUT = 30s
	COMPRESSION = gzip
	INSECURE = TRUE
	HEADERS=foo=bar,overwrite=false
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "test service", OpenTelemetry.ServiceName)
	assert.Equal(t, "foo=bar", OpenTelemetry.ResourceAttributes)
	assert.Equal(t, 30*time.Second, OpenTelemetry.OtelTraces.Timeout)
	assert.Equal(t, "gzip", OpenTelemetry.OtelTraces.Compression)
	assert.Equal(t, sdktrace.AlwaysSample(), OpenTelemetry.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.OtelTraces.Endpoint.String())
	assert.Contains(t, OpenTelemetry.OtelTraces.Headers, "foo")
	assert.Equal(t, "bar", OpenTelemetry.OtelTraces.Headers["foo"])
	assert.Contains(t, OpenTelemetry.OtelTraces.Headers, "overwrite")
	assert.Equal(t, "false", OpenTelemetry.OtelTraces.Headers["overwrite"])
}

func TestOpenTelemetryTraceDisable(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.False(t, OpenTelemetry.Enabled)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	ENABLED=true
	EXPORTER_OTLP_ENDPOINT =
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.True(t, IsOpenTelemetryEnabled())
	endpoint, _ := url.Parse("http://localhost:4318/")
	assert.Equal(t, endpoint, OpenTelemetry.OtelTraces.Endpoint)
}

func TestSamplerCombinations(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	type config struct {
		IniCfg   string
		Expected sdktrace.Sampler
	}
	testSamplers := []config{
		{`[opentelemetry]
		ENABLED=true
  TRACES_SAMPLER = always_on
  TRACES_SAMPLER_ARG = nothing`, sdktrace.AlwaysSample()},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = always_off`, sdktrace.NeverSample()},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = traceidratio
  TRACES_SAMPLER_ARG = 0.7`, sdktrace.TraceIDRatioBased(0.7)},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = traceidratio
  TRACES_SAMPLER_ARG = badarg`, sdktrace.TraceIDRatioBased(1)},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = parentbased_always_off`, sdktrace.ParentBased(sdktrace.NeverSample())},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = parentbased_always_of`, sdktrace.ParentBased(sdktrace.AlwaysSample())},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = parentbased_traceidratio
  TRACES_SAMPLER_ARG = 0.3`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.3))},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = parentbased_traceidratio
  TRACES_SAMPLER_ARG = badarg`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1))},
		{`[opentelemetry]
	ENABLED=true
  TRACES_SAMPLER = not existing
  TRACES_SAMPLER_ARG = badarg`, sdktrace.ParentBased(sdktrace.AlwaysSample())},
	}

	for _, sampler := range testSamplers {
		cfg, err := NewConfigProviderFromData(sampler.IniCfg)
		require.NoError(t, err)
		loadOpenTelemetryFrom(cfg)
		assert.Equal(t, sampler.Expected, OpenTelemetry.Sampler)
	}
}

func TestOpentelemetryBadConfigs(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := `
	[opentelemetry]
	ENABLED=true

	[opentelemetry.exporter.otlp]
	ENDPOINT = jaeger:4317/
	`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "jaeger:4317/", OpenTelemetry.OtelTraces.Endpoint.String())

	iniStr = ``
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	ENABLED=true
	SERVICE_NAME =
  TRACES_SAMPLER = not existing one
	[opentelemetry.exporter.otlp]
	ENDPOINT = http://jaeger:4317/

	TIMEOUT = abc
	COMPRESSION = foo
	HEADERS=%s=bar,foo=%h,foo

	`

	cfg, err = NewConfigProviderFromData(iniStr)

	require.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "forgejo", OpenTelemetry.ServiceName)
	assert.Equal(t, 10*time.Second, OpenTelemetry.OtelTraces.Timeout)
	assert.Equal(t, sdktrace.ParentBased(sdktrace.AlwaysSample()), OpenTelemetry.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.OtelTraces.Endpoint.String())
	assert.Empty(t, OpenTelemetry.OtelTraces.Headers)
}
