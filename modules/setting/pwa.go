// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
)

type PwaConfig struct {
	Standalone bool
}

var PWA = PwaConfig{
	Standalone: false,
}

func loadPWAFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("pwa")
	if err := sec.MapTo(&PWA); err != nil {
		log.Fatal("Failed to map [pwa] settings: %v", err)
	}
}

type manifestIcon struct {
	Src   string `json:"src"`
	Type  string `json:"type"`
	Sizes string `json:"sizes"`
}

type manifestJSON struct {
	Name      string         `json:"name"`
	ShortName string         `json:"short_name"`
	StartURL  string         `json:"start_url"`
	Icons     []manifestIcon `json:"icons"`
	Display   string         `json:"display,omitempty"`
}

func GetManifestJSON() ([]byte, error) {
	manifest := manifestJSON{
		Name:      AppName,
		ShortName: AppName,
		StartURL:  AppURL,
		Icons: []manifestIcon{
			{
				Src:   AbsoluteAssetURL + "/assets/img/logo.png",
				Type:  "image/png",
				Sizes: "512x512",
			},
			{
				Src:   AbsoluteAssetURL + "/assets/img/logo.svg",
				Type:  "image/svg+xml",
				Sizes: "512x512",
			},
		},
	}

	if PWA.Standalone {
		manifest.Display = "standalone"
	}

	return json.Marshal(manifest)
}
