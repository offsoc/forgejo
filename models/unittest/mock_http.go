// Copyright 2017 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package unittest

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"forgejo.org/modules/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mocks HTTP responses of a third-party service (such as GitHub, GitLab…)
// This has two modes:
//   - live mode: the requests made to the mock HTTP server are transmitted to the live
//     service, and responses are saved as test data files
//   - test mode: the responses to requests to the mock HTTP server are read from the
//     test data files
func NewMockWebServer(t *testing.T, liveServerBaseURL, testDataDir string, liveMode bool) *httptest.Server {
	mockServerBaseURL := ""
	ignoredHeaders := []string{"cf-ray", "server", "date", "report-to", "nel", "x-request-id"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := NormalizedFullPath(r.URL)
		isGh := liveServerBaseURL == "https://api.github.com"
		if isGh {
			// Workaround for GitHub: trim `/api/v3` from the path
			path = strings.TrimPrefix(path, "/api/v3")
		}
		log.Info("Mock HTTP Server: got request for path %s", r.URL.Path)
		// TODO check request method (support POST?)
		fixturePath := fmt.Sprintf("%s/%s_%s", testDataDir, r.Method, url.PathEscape(path))
		if liveMode {
			liveURL := fmt.Sprintf("%s%s", liveServerBaseURL, path)

			request, err := http.NewRequest(r.Method, liveURL, nil)
			require.NoError(t, err, "constructing an HTTP request to %s failed", liveURL)
			for headerName, headerValues := range r.Header {
				// do not pass on the encoding: let the Transport of the HTTP client handle that for us
				if strings.ToLower(headerName) != "accept-encoding" {
					for _, headerValue := range headerValues {
						request.Header.Add(headerName, headerValue)
					}
				}
			}

			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err, "HTTP request to %s failed: %s", liveURL)
			assert.Less(t, response.StatusCode, 400, "unexpected status code for %s", liveURL)

			fixture, err := os.Create(fixturePath)
			require.NoError(t, err, "failed to open the fixture file %s for writing", fixturePath)
			defer fixture.Close()
			fixtureWriter := bufio.NewWriter(fixture)

			for headerName, headerValues := range response.Header {
				for _, headerValue := range headerValues {
					if !slices.Contains(ignoredHeaders, strings.ToLower(headerName)) {
						_, err := fmt.Fprintf(fixtureWriter, "%s: %s\n", headerName, headerValue)
						require.NoError(t, err, "writing the header of the HTTP response to the fixture file failed")
					}
				}
			}
			_, err = fixtureWriter.WriteString("\n")
			require.NoError(t, err, "writing the header of the HTTP response to the fixture file failed")
			fixtureWriter.Flush()

			log.Info("Mock HTTP Server: writing response to %s", fixturePath)
			_, err = io.Copy(fixture, response.Body)
			require.NoError(t, err, "writing the body of the HTTP response to %s failed", liveURL)

			err = fixture.Sync()
			require.NoError(t, err, "writing the body of the HTTP response to the fixture file failed")
		}

		fixture, err := os.ReadFile(fixturePath)
		require.NoError(t, err, "missing mock HTTP response: "+fixturePath)

		w.WriteHeader(http.StatusOK)

		// replace any mention of the live HTTP service by the mocked host
		stringFixture := strings.ReplaceAll(string(fixture), liveServerBaseURL, mockServerBaseURL)
		if isGh {
			// Workaround for GitHub: replace github.com by the mock server's base URL
			stringFixture = strings.ReplaceAll(stringFixture, "https://github.com", mockServerBaseURL)
		}
		// parse back the fixture file into a series of HTTP headers followed by response body
		lines := strings.Split(stringFixture, "\n")
		for idx, line := range lines {
			colonIndex := strings.Index(line, ": ")
			if colonIndex != -1 {
				w.Header().Set(line[0:colonIndex], line[colonIndex+2:])
			} else {
				// we reached the end of the headers (empty line), so what follows is the body
				responseBody := strings.Join(lines[idx+1:], "\n")
				_, err := w.Write([]byte(responseBody))
				require.NoError(t, err, "writing the body of the HTTP response failed")
				break
			}
		}
	}))
	mockServerBaseURL = server.URL
	return server
}

func NormalizedFullPath(url *url.URL) string {
	// TODO normalize path (remove trailing slash?)
	// TODO normalize RawQuery (order query parameters?)
	if len(url.Query()) == 0 {
		return url.EscapedPath()
	}
	return fmt.Sprintf("%s?%s", url.EscapedPath(), url.RawQuery)
}
