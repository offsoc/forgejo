// Copyright 2024 TheFox0x7. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package opentelemetry

import (
	"context"
	"slices"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestResourceServiceName(t *testing.T) {
	ctx := context.Background()

	resource, err := newResource(ctx)
	require.NoError(t, err)
	serviceKeyIdx := slices.IndexFunc(resource.Attributes(), func(v attribute.KeyValue) bool {
		return v.Key == semconv.ServiceNameKey
	})
	require.NotEqual(t, -1, serviceKeyIdx)

	assert.Equal(t, "forgejo", resource.Attributes()[serviceKeyIdx].Value.AsString())

	defer test.MockVariableValue(&setting.OpenTelemetry.ServiceName, "non-default value")()
	resource, err = newResource(ctx)
	require.NoError(t, err)

	serviceKeyIdx = slices.IndexFunc(resource.Attributes(), func(v attribute.KeyValue) bool {
		return v.Key == semconv.ServiceNameKey
	})
	require.NotEqual(t, -1, serviceKeyIdx)

	assert.Equal(t, "non-default value", resource.Attributes()[serviceKeyIdx].Value.AsString())
}

func TestResourceAttributes(t *testing.T) {
	ctx := context.Background()
	defer test.MockVariableValue(&setting.OpenTelemetry.ResourceDetectors, "foo")()
	defer test.MockVariableValue(&setting.OpenTelemetry.ResourceAttributes, "Test=LABEL,broken,unescape=%XXlabel")()
	res, err := newResource(ctx)
	require.NoError(t, err)
	expected, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(setting.OpenTelemetry.ServiceName),
		semconv.ServiceVersion(setting.ForgejoVersion),
		attribute.String("Test", "LABEL"),
		attribute.String("unescape", "%XXlabel"),
	))
	require.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDecoderParity(t *testing.T) {
	ctx := context.Background()
	defer test.MockVariableValue(&setting.OpenTelemetry.ResourceDetectors, "sdk,process,os,host")()
	exp, err := resource.New(
		ctx, resource.WithTelemetrySDK(), resource.WithOS(), resource.WithProcess(), resource.WithHost(), resource.WithAttributes(
			semconv.ServiceName(setting.OpenTelemetry.ServiceName), semconv.ServiceVersion(setting.ForgejoVersion),
		),
	)
	require.NoError(t, err)
	res2, err := newResource(ctx)
	require.NoError(t, err)
	assert.Equal(t, exp, res2)
}
