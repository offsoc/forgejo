// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// extracted from `/build/lint-locale.go`, `/build/lint-locale-usage.go`

package locale

import (
	"encoding/json" //nolint:depguard
	"fmt"

	"gopkg.in/ini.v1" //nolint:depguard
)

func IterateMessagesContent(localeContent []byte, onMsgid func(string, string) error) error {
	// Same configuration as Forgejo uses.
	cfg := ini.Empty(ini.LoadOptions{
		IgnoreContinuation: true,
	})
	cfg.NameMapper = ini.SnackCase

	if err := cfg.Append(localeContent); err != nil {
		return err
	}

	for _, section := range cfg.Sections() {
		for _, key := range section.Keys() {
			var trKey string
			if section.Name() == "" || section.Name() == "DEFAULT" || section.Name() == "common" {
				trKey = key.Name()
			} else {
				trKey = section.Name() + "." + key.Name()
			}
			if err := onMsgid(trKey, key.Value()); err != nil {
				return err
			}
		}
	}

	return nil
}

func iterateMessagesNextInner(onMsgid func(string, string) error, data map[string]any, trKey ...string) error {
	for key, value := range data {
		currentKey := key
		if len(trKey) == 1 {
			currentKey = trKey[0] + "." + key
		}

		switch value := value.(type) {
		case string:
			if err := onMsgid(currentKey, value); err != nil {
				return err
			}
		case map[string]any:
			if err := iterateMessagesNextInner(onMsgid, value, currentKey); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected type: %s - %T", currentKey, value)
		}
	}

	return nil
}

func IterateMessagesNextContent(localeContent []byte, onMsgid func(string, string) error) error {
	var localeData map[string]any
	if err := json.Unmarshal(localeContent, &localeData); err != nil {
		return err
	}
	return iterateMessagesNextInner(onMsgid, localeData)
}
