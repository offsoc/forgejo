// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"forgejo.org/modules/httpcache"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web/routing"
)

func storageHandler(storageSetting *setting.Storage, prefix string, objStore storage.ObjectStorage) http.HandlerFunc {
	prefix = strings.Trim(prefix, "/")
	funcInfo := routing.GetFuncInfo(storageHandler, prefix)

	if storageSetting.MinioConfig.ServeDirect {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "GET" && req.Method != "HEAD" {
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}

			if !strings.HasPrefix(req.URL.Path, "/"+prefix+"/") {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			routing.UpdateFuncInfo(req.Context(), funcInfo)

			rPath := strings.TrimPrefix(req.URL.Path, "/"+prefix+"/")
			rPath = util.PathJoinRelX(rPath)

			u, err := objStore.URL(rPath, path.Base(rPath), nil)
			if err != nil {
				if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
					log.Warn("Unable to find %s %s", prefix, rPath)
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
					return
				}
				log.Error("Error whilst getting URL for %s %s. Error: %v", prefix, rPath, err)
				http.Error(w, fmt.Sprintf("Error whilst getting URL for %s %s", prefix, rPath), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, req, u.String(), http.StatusTemporaryRedirect)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" && req.Method != "HEAD" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if !strings.HasPrefix(req.URL.Path, "/"+prefix+"/") {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		routing.UpdateFuncInfo(req.Context(), funcInfo)

		rPath := strings.TrimPrefix(req.URL.Path, "/"+prefix+"/")
		rPath = util.PathJoinRelX(rPath)
		if rPath == "" || rPath == "." {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		fi, err := objStore.Stat(rPath)
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
				log.Warn("Unable to find %s %s", prefix, rPath)
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Error("Error whilst opening %s %s. Error: %v", prefix, rPath, err)
			http.Error(w, fmt.Sprintf("Error whilst opening %s %s", prefix, rPath), http.StatusInternalServerError)
			return
		}

		fr, err := objStore.Open(rPath)
		if err != nil {
			log.Error("Error whilst opening %s %s. Error: %v", prefix, rPath, err)
			http.Error(w, fmt.Sprintf("Error whilst opening %s %s", prefix, rPath), http.StatusInternalServerError)
			return
		}
		defer fr.Close()
		httpcache.ServeContentWithCacheControl(w, req, path.Base(rPath), fi.ModTime(), fr)
	})
}
