// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package setting

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	"code.gitea.io/gitea/modules/generate"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"

	"github.com/unknwon/com"
	ini "gopkg.in/ini.v1"
)

// LFS represents the configuration for Git LFS
var LFS = struct {
	StartServer     bool          `ini:"LFS_START_SERVER"`
	JWTSecretBase64 string        `ini:"LFS_JWT_SECRET"`
	JWTSecretBytes  []byte        `ini:"-"`
	HTTPAuthExpiry  time.Duration `ini:"LFS_HTTP_AUTH_EXPIRY"`
	MaxFileSize     int64         `ini:"LFS_MAX_FILE_SIZE"`
	LocksPagingNum  int           `ini:"LFS_LOCKS_PAGING_NUM"`

	Storage
}{}

func newLFSService() {
	sec := Cfg.Section("server")
	if err := sec.MapTo(&LFS); err != nil {
		log.Fatal("Failed to map LFS settings: %v", err)
	}

	lfsSec := Cfg.Section("lfs")
	LFS.Storage.Type = lfsSec.Key("STORAGE_TYPE").MustString("")
	if LFS.Storage.Type == "" {
		LFS.Storage.Type = "default"
	}

	if LFS.Storage.Type != LocalStorageType && LFS.Storage.Type != MinioStorageType {
		storage, ok := storages[LFS.Storage.Type]
		if !ok {
			log.Fatal("Failed to get lfs storage type: %s", LFS.Storage.Type)
		}
		LFS.Storage = storage

		for _, key := range storage.Section.Keys() {
			if !lfsSec.HasKey(key.Name()) {
				_, _ = lfsSec.NewKey(key.Name(), key.Value())
			}
		}
		LFS.Storage.Section = lfsSec
	}

	// Override
	LFS.ServeDirect = lfsSec.Key("SERVE_DIRECT").MustBool(LFS.ServeDirect)
	LFS.Storage.Path = sec.Key("LFS_CONTENT_PATH").MustString(filepath.Join(AppDataPath, "lfs"))
	LFS.Storage.Path = lfsSec.Key("PATH").MustString(LFS.Storage.Path)
	if !filepath.IsAbs(LFS.Storage.Path) {
		lfsSec.Key("PATH").SetValue(filepath.Join(AppWorkPath, LFS.Storage.Path))
	}
	lfsSec.Key("MINIO_BASE_PATH").MustString("lfs/")

	if LFS.LocksPagingNum == 0 {
		LFS.LocksPagingNum = 50
	}

	LFS.HTTPAuthExpiry = sec.Key("LFS_HTTP_AUTH_EXPIRY").MustDuration(20 * time.Minute)

	if LFS.StartServer {
		LFS.JWTSecretBytes = make([]byte, 32)
		n, err := base64.RawURLEncoding.Decode(LFS.JWTSecretBytes, []byte(LFS.JWTSecretBase64))

		if err != nil || n != 32 {
			LFS.JWTSecretBase64, err = generate.NewJwtSecret()
			if err != nil {
				log.Fatal("Error generating JWT Secret for custom config: %v", err)
				return
			}

			// Save secret
			cfg := ini.Empty()
			if com.IsFile(CustomConf) {
				// Keeps custom settings if there is already something.
				if err := cfg.Append(CustomConf); err != nil {
					log.Error("Failed to load custom conf '%s': %v", CustomConf, err)
				}
			}

			cfg.Section("server").Key("LFS_JWT_SECRET").SetValue(LFS.JWTSecretBase64)

			if err := os.MkdirAll(filepath.Dir(CustomConf), os.ModePerm); err != nil {
				log.Fatal("Failed to create '%s': %v", CustomConf, err)
			}
			if err := cfg.SaveTo(CustomConf); err != nil {
				log.Fatal("Error saving generated JWT Secret to custom config: %v", err)
				return
			}
		}
	}
}

// CheckLFSVersion will check lfs version, if not satisfied, then disable it.
func CheckLFSVersion() {
	if LFS.StartServer {
		//Disable LFS client hooks if installed for the current OS user
		//Needs at least git v2.1.2

		err := git.LoadGitVersion()
		if err != nil {
			log.Fatal("Error retrieving git version: %v", err)
		}

		if git.CheckGitVersionConstraint(">= 2.1.2") != nil {
			LFS.StartServer = false
			log.Error("LFS server support needs at least Git v2.1.2")
		} else {
			git.GlobalCommandArgs = append(git.GlobalCommandArgs, "-c", "filter.lfs.required=",
				"-c", "filter.lfs.smudge=", "-c", "filter.lfs.clean=")
		}
	}
}
