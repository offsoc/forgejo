// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package install

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/opentelemetry"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutes(t *testing.T) {
	r := Routes()
	assert.NotNil(t, r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `class="page-content install"`)

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/no-such", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 404, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/assets/img/gitea.svg", nil)
	r.ServeHTTP(w, req)
	assert.EqualValues(t, 200, w.Code)
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestOtelChi(t *testing.T) {
	ServiceName := "forgejo-otelchi" + uuid.NewString()

	otelURL, ok := os.LookupEnv("TEST_OTEL_URL")
	if !ok {
		t.Skip("TEST_OTEL_URL not set")
	}
	traceEndpoint, err := url.Parse(otelURL)
	require.NoError(t, err)
	config := &setting.OtelExporter{
		Endpoint: traceEndpoint,
		Protocol: "grpc",
	}

	defer test.MockVariableValue(&setting.OpenTelemetry.Enabled, true)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces, "otlp")() // Required due to lazy loading
	defer test.MockVariableValue(&setting.OpenTelemetry.ServiceName, ServiceName)()
	defer test.MockVariableValue(&setting.OpenTelemetry.OtelTraces, config)()

	require.NoError(t, opentelemetry.Init(context.Background()))
	r := Routes()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/e/img/gitea.svg", nil)
	r.ServeHTTP(w, req)

	traceEndpoint.Host = traceEndpoint.Hostname() + ":16686"
	traceEndpoint.Path = "/api/services"

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := http.Get(traceEndpoint.String())
		require.NoError(t, err)

		apiResponse, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Contains(collect, string(apiResponse), ServiceName)
	}, 15*time.Second, 1*time.Second)
}
