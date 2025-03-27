// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package healthcheck

import (
	"context"
	"net/http"
	"os"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

type status string

const (
	// Pass healthy (acceptable aliases: "ok" to support Node's Terminus and "up" for Java's SpringBoot)
	// fail unhealthy (acceptable aliases: "error" to support Node's Terminus and "down" for Java's SpringBoot), and
	// warn healthy, with some concerns.
	//
	// ref https://datatracker.ietf.org/doc/html/draft-inadarei-api-health-check#section-3.1
	// status: (required) indicates whether the service status is acceptable
	// or not.  API publishers SHOULD use following values for the field:
	// The value of the status field is case-insensitive and is tightly
	// related with the HTTP Response code returned by the health endpoint.
	// For "pass" status, HTTP Response code in the 2xx-3xx range MUST be
	// used.  For "fail" status, HTTP Response code in the 4xx-5xx range
	// MUST be used.  In case of the "warn" status, endpoints MUST return
	// HTTP status in the 2xx-3xx range, and additional information SHOULD
	// be provided, utilizing optional fields of the Response.
	Pass status = "pass"
	Fail status = "fail"
	warn status = "warn"
)

func (s status) ToHTTPStatus() int {
	if s == Pass || s == warn {
		return http.StatusOK
	}
	return http.StatusFailedDependency
}

type checks map[string][]componentStatus

// Response is the data returned by the health endpoint, which will be marshaled to JSON format
type Response struct {
	Status      status `json:"status"`
	Description string `json:"description"`      // a human-friendly description of the service
	Checks      checks `json:"checks,omitempty"` // The Checks Object, should be omitted on installation route
}

// componentStatus presents one status of a single check object
// an object that provides detailed health statuses of additional downstream systems and endpoints
// which can affect the overall health of the main API.
type componentStatus struct {
	Status status `json:"status"`
	Time   string `json:"time"`             // the date-time, in ISO8601 format
	Output string `json:"output,omitempty"` // this field SHOULD be omitted for "pass" state.
}

// Check is the health check API handler
func Check(w http.ResponseWriter, r *http.Request) {
	rsp := Response{
		Status:      Pass,
		Description: setting.AppName,
		Checks:      make(checks),
	}

	statuses := make([]status, 0)
	if setting.InstallLock {
		statuses = append(statuses, checkDatabase(r.Context(), rsp.Checks))
		statuses = append(statuses, checkCache(rsp.Checks))
	}
	for _, s := range statuses {
		if s != Pass {
			rsp.Status = Fail
			break
		}
	}

	data, _ := json.MarshalIndent(rsp, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(rsp.Status.ToHTTPStatus())
	_, _ = w.Write(data)
}

// database checks gitea database status
func checkDatabase(ctx context.Context, checks checks) status {
	st := componentStatus{}
	if err := db.GetEngine(ctx).Ping(); err != nil {
		st.Status = Fail
		st.Time = getCheckTime()
		log.Error("database ping failed with error: %v", err)
	} else {
		st.Status = Pass
		st.Time = getCheckTime()
	}

	if setting.Database.Type.IsSQLite3() && st.Status == Pass {
		if !setting.EnableSQLite3 {
			st.Status = Fail
			st.Time = getCheckTime()
			log.Error("SQLite3 health check failed with error: %v", "this Forgejo binary is built without SQLite3 enabled")
		} else {
			if _, err := os.Stat(setting.Database.Path); err != nil {
				st.Status = Fail
				st.Time = getCheckTime()
				log.Error("SQLite3 file exists check failed with error: %v", err)
			}
		}
	}

	checks["database:ping"] = []componentStatus{st}
	return st.Status
}

// cache checks gitea cache status
func checkCache(checks checks) status {
	st := componentStatus{}
	if err := cache.GetCache().Ping(); err != nil {
		st.Status = Fail
		st.Time = getCheckTime()
		log.Error("cache ping failed with error: %v", err)
	} else {
		st.Status = Pass
		st.Time = getCheckTime()
	}
	checks["cache:ping"] = []componentStatus{st}
	return st.Status
}

func getCheckTime() string {
	return time.Now().UTC().Format(time.RFC3339)
}
