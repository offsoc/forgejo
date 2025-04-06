// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package updatechecker

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"forgejo.org/modules/json"
	"forgejo.org/modules/proxy"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/system"

	"github.com/hashicorp/go-version"
)

// CheckerState stores the remote version from the JSON endpoint
type CheckerState struct {
	LatestVersion string
}

// Name returns the name of the state item for update checker
func (r *CheckerState) Name() string {
	return "update-checker"
}

// GiteaUpdateChecker returns error when new version of Gitea is available
func GiteaUpdateChecker(httpEndpoint, domainEndpoint string) error {
	var version string
	var err error
	if domainEndpoint != "" {
		version, err = getVersionDNS(domainEndpoint)
	} else {
		version, err = getVersionHTTP(httpEndpoint)
	}

	if err != nil {
		return err
	}

	return UpdateRemoteVersion(context.Background(), version)
}

// getVersionDNS will request the TXT records for the domain. If a record starts
// with "forgejo_versions=" everything after that will be used as the latest
// version available.
func getVersionDNS(domainEndpoint string) (version string, err error) {
	records, err := net.LookupTXT(domainEndpoint)
	if err != nil {
		return "", err
	}

	if len(records) == 0 {
		return "", errors.New("no TXT records were found")
	}

	for _, record := range records {
		if strings.HasPrefix(record, "forgejo_versions=") {
			// Get all supported versions, separated by a comma.
			supportedVersions := strings.Split(strings.TrimPrefix(record, "forgejo_versions="), ",")
			// For now always return the latest supported version.
			return supportedVersions[len(supportedVersions)-1], nil
		}
	}

	return "", errors.New("there is no TXT record with a valid value")
}

// getVersionHTTP will make an HTTP request to the endpoint, and the returned
// content is JSON. The "latest.version" path's value will be used as the latest
// version available.
func getVersionHTTP(httpEndpoint string) (version string, err error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: proxy.Proxy(),
		},
	}

	req, err := http.NewRequest("GET", httpEndpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type respType struct {
		Latest struct {
			Version string `json:"version"`
		} `json:"latest"`
	}
	respData := respType{}
	err = json.Unmarshal(body, &respData)
	if err != nil {
		return "", err
	}
	return respData.Latest.Version, nil
}

// UpdateRemoteVersion updates the latest available version of Gitea
func UpdateRemoteVersion(ctx context.Context, version string) (err error) {
	return system.AppState.Set(ctx, &CheckerState{LatestVersion: version})
}

// GetRemoteVersion returns the current remote version (or currently installed version if fail to fetch from DB)
func GetRemoteVersion(ctx context.Context) string {
	item := new(CheckerState)
	if err := system.AppState.Get(ctx, item); err != nil {
		return ""
	}
	return item.LatestVersion
}

// GetNeedUpdate returns true whether a newer version of Gitea is available
func GetNeedUpdate(ctx context.Context) bool {
	curVer, err := version.NewVersion(setting.AppVer)
	if err != nil {
		// return false to fail silently
		return false
	}
	remoteVerStr := GetRemoteVersion(ctx)
	if remoteVerStr == "" {
		// no remote version is known
		return false
	}
	remoteVer, err := version.NewVersion(remoteVerStr)
	if err != nil {
		// return false to fail silently
		return false
	}
	return curVer.LessThan(remoteVer)
}
