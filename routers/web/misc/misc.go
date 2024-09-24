// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"path"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/httpcache"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

func SSHInfo(rw http.ResponseWriter, req *http.Request) {
	if !git.SupportProcReceive {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	rw.Header().Set("content-type", "text/json;charset=UTF-8")
	_, err := rw.Write([]byte(`{"type":"gitea","version":1}`))
	if err != nil {
		log.Error("fail to write result: err: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func DummyOK(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func RobotsTxt(w http.ResponseWriter, req *http.Request) {
	robotsTxt := util.FilePathJoinAbs(setting.CustomPath, "public/robots.txt")
	if ok, _ := util.IsExist(robotsTxt); !ok {
		robotsTxt = util.FilePathJoinAbs(setting.CustomPath, "robots.txt") // the legacy "robots.txt"
	}
	httpcache.SetCacheControlInHeader(w.Header(), setting.StaticCacheTime)
	http.ServeFile(w, req, robotsTxt)
}

func ManifestJson(w http.ResponseWriter, req *http.Request) {
	httpcache.SetCacheControlInHeader(w.Header(), setting.StaticCacheTime)
	w.Header().Add("content-type", "application/manifest+json;charset=UTF-8")

	manifestJson := util.FilePathJoinAbs(setting.CustomPath, "public/manifest.json")
	if ok, _ := util.IsExist(manifestJson); ok {
		http.ServeFile(w, req, manifestJson)
		return
	}

	bytes, err := setting.GetManifestJson()
	if err != nil {
		log.Error("unable to marshal manifest JSON. Error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func StaticRedirect(target string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, path.Join(setting.StaticURLPrefix, target), http.StatusMovedPermanently)
	}
}
