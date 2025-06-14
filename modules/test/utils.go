// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"forgejo.org/modules/json"
)

// RedirectURL returns the redirect URL of a http response.
// It also works for JSONRedirect: `{"redirect": "..."}`
func RedirectURL(resp http.ResponseWriter) string {
	loc := resp.Header().Get("Location")
	if loc != "" {
		return loc
	}
	if r, ok := resp.(*httptest.ResponseRecorder); ok {
		m := map[string]any{}
		err := json.Unmarshal(r.Body.Bytes(), &m)
		if err == nil {
			if loc, ok := m["redirect"].(string); ok {
				return loc
			}
		}
	}
	return ""
}

func IsNormalPageCompleted(s string) bool {
	return strings.Contains(s, `<footer class="page-footer"`) && strings.Contains(s, `</html>`)
}

// use for global variables only
func MockVariableValue[T any](p *T, v T) (reset func()) {
	old := *p
	*p = v
	return func() { *p = old }
}

// use for global variables only
func MockProtect[T any](p *T) (reset func()) {
	old := *p
	return func() { *p = old }
}

// When this is called, sleep until the unix time was increased by one.
func SleepTillNextSecond() {
	time.Sleep(time.Second - time.Since(time.Now().Truncate(time.Second)))
}
