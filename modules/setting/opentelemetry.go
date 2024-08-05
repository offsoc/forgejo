// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package setting

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	opentelemetrySectionName string = "opentelemetry"
	exporter                 string = ".exporter"
	otlp                     string = ".otlp"
	alwaysOn                 string = "always_on"
	alwaysOff                string = "always_off"
	traceIDRatio             string = "traceidratio"
	parentBasedAlwaysOn      string = "parentbased_always_on"
	parentBasedAlwaysOff     string = "parentbased_always_off"
	parentBasedTraceIDRatio  string = "parentbased_traceidratio"
)

var OpenTelemetry = struct {
	// Inverse of OTEL_SDK_DISABLE, skips telemetry setup
	Enabled            bool
	ServiceName        string
	ResourceAttributes string
	ResourceDetectors  string
	Sampler            sdktrace.Sampler
	Traces             string

	OtelTraces *OtelExporter
}{
	ServiceName: "forgejo",
	Traces:      "otel",
}

type OtelExporter struct {
	Endpoint          *url.URL          `ini:"ENDPOINT"`
	Headers           map[string]string `ini:"-"`
	Compression       string            `ini:"COMPRESSION"`
	Certificate       string            `ini:"CERTIFICATE"`
	ClientKey         string            `ini:"CLIENT_KEY"`
	ClientCertificate string            `ini:"CLIENT_CERTIFICATE"`
	Timeout           time.Duration     `ini:"TIMEOUT"`
	Protocol          string            `ini:"-"`
}

func createOtlpExporterConfig(rootCfg ConfigProvider, section string) *OtelExporter {
	protocols := []string{"http/protobuf", "grpc"}
	endpoint, _ := url.Parse("http://localhost:4318/")
	exp := &OtelExporter{
		Endpoint: endpoint,
		Timeout:  10 * time.Second,
		Headers:  map[string]string{},
		Protocol: "http/protobuf",
	}

	loadSection := func(name string) {
		otlp := rootCfg.Section(name)
		if otlp.HasKey("ENDPOINT") {
			endpoint, err := url.Parse(otlp.Key("ENDPOINT").String())
			if err != nil {
				log.Warn("Endpoint parsing failed, section: %s, err %v", name, err)
			} else {
				exp.Endpoint = endpoint
			}
		}
		if err := otlp.MapTo(exp); err != nil {
			log.Warn("Mapping otlp settings failed, section: %s, err: %v", name, err)
		}

		exp.Protocol = otlp.Key("PROTOCOL").In(exp.Protocol, protocols)

		headers := otlp.Key("HEADERS").String()
		if headers != "" {
			for k, v := range _stringToHeader(headers) {
				exp.Headers[k] = v
			}
		}
	}
	loadSection("opentelemetry.exporter.otlp")

	loadSection("opentelemetry.exporter.otlp" + section)

	if len(exp.Certificate) > 0 && !filepath.IsAbs(exp.Certificate) {
		exp.Certificate = filepath.Join(CustomPath, exp.Certificate)
	}
	if len(exp.ClientCertificate) > 0 && !filepath.IsAbs(exp.ClientCertificate) {
		exp.ClientCertificate = filepath.Join(CustomPath, exp.ClientCertificate)
	}
	if len(exp.ClientKey) > 0 && !filepath.IsAbs(exp.ClientKey) {
		exp.ClientKey = filepath.Join(CustomPath, exp.ClientKey)
	}

	return exp
}

func loadOpenTelemetryFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section(opentelemetrySectionName)
	OpenTelemetry.Enabled = sec.Key("ENABLED").MustBool(false)
	if !OpenTelemetry.Enabled {
		return
	}

	// Load resource related settings
	OpenTelemetry.ServiceName = sec.Key("SERVICE_NAME").MustString("forgejo")
	OpenTelemetry.ResourceAttributes = sec.Key("RESOURCE_ATTRIBUTES").String()
	OpenTelemetry.ResourceDetectors = strings.ToLower(sec.Key("RESOURCE_DETECTORS").String())

	// Load tracing related settings
	samplers := make([]string, 0, len(sampler))
	for k := range sampler {
		samplers = append(samplers, k)
	}

	samplerName := sec.Key("TRACES_SAMPLER").In(parentBasedAlwaysOn, samplers)
	samplerArg := sec.Key("TRACES_SAMPLER_ARG").MustString("")
	OpenTelemetry.Sampler = sampler[samplerName](samplerArg)

	switch sec.Key("TRACES_EXPORTER").MustString("otlp") {
	case "none":
		OpenTelemetry.Traces = "none"
	default:
		OpenTelemetry.Traces = "otlp"
		OpenTelemetry.OtelTraces = createOtlpExporterConfig(rootCfg, ".traces")
	}
}

var sampler = map[string]func(arg string) sdktrace.Sampler{
	alwaysOff: func(_ string) sdktrace.Sampler {
		return sdktrace.NeverSample()
	},
	alwaysOn: func(_ string) sdktrace.Sampler {
		return sdktrace.AlwaysSample()
	},
	traceIDRatio: func(arg string) sdktrace.Sampler {
		ratio, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			ratio = 1
		}
		return sdktrace.TraceIDRatioBased(ratio)
	},
	parentBasedAlwaysOff: func(_ string) sdktrace.Sampler {
		return sdktrace.ParentBased(sdktrace.NeverSample())
	},
	parentBasedAlwaysOn: func(_ string) sdktrace.Sampler {
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	},
	parentBasedTraceIDRatio: func(arg string) sdktrace.Sampler {
		ratio, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			ratio = 1
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	},
}

// Opentelemetry SDK function port

func _stringToHeader(value string) map[string]string {
	headersPairs := strings.Split(value, ",")
	headers := make(map[string]string)

	for _, header := range headersPairs {
		n, v, found := strings.Cut(header, "=")
		if !found {
			log.Warn("Otel header ignored on %q: missing '='", header)
			continue
		}
		name, err := url.PathUnescape(n)
		if err != nil {
			log.Warn("Otel header ignored on %q, invalid header key: %s", header, n)
			continue
		}
		trimmedName := strings.TrimSpace(name)
		value, err := url.PathUnescape(v)
		if err != nil {
			log.Warn("Otel header ignored on %q, invalid header value: %s", header, v)
			continue
		}
		trimmedValue := strings.TrimSpace(value)

		headers[trimmedName] = trimmedValue
	}

	return headers
}

func IsOpenTelemetryEnabled() bool {
	return OpenTelemetry.Enabled
}
