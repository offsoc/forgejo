// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// extracted from `/build/lint-locale.go`, `/build/lint-locale-usage.go`

package localeiter

import (
	"encoding/json" //nolint:depguard
	"fmt"

	"forgejo.org/modules/setting"
)

func IterateMessagesContent(localeContent []byte, onMsgid func(string, string) error) error {
	cfg, err := setting.NewConfigProviderForLocale(localeContent)
	if err != nil {
		return err
	}

	for _, section := range cfg.Sections() {
		for _, key := range section.Keys() {
			var trKey string
			// see https://codeberg.org/forgejo/discussions/issues/104
			//     https://github.com/WeblateOrg/weblate/issues/10831
			// for an explanation of why "common" is an alternative
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

func iterateMessagesNextInner(onMsgid func(string, string, string) error, data map[string]any, trKey string) error {
	for key, value := range data {
		fullKey := key
		if trKey != "" {
			fullKey = trKey + "." + key
		}
		switch value := value.(type) {
		case string:
			// Check whether we are adding a plural form to the parent object, or a new nested JSON object.
			realKey := trKey
			pluralSuffix := ""

			switch key {
			case "zero", "one", "two", "few", "many":
				pluralSuffix = key
			case "other":
				// do nothing
			default:
				realKey = fullKey
			}

			if err := onMsgid(realKey, pluralSuffix, value); err != nil {
				return err
			}

		case map[string]any:
			if err := iterateMessagesNextInner(onMsgid, value, fullKey); err != nil {
				return err
			}

		case nil:
			// do nothing

		default:
			return fmt.Errorf("Unexpected JSON type: %s - %T", fullKey, value)
		}
	}

	return nil
}

func IterateMessagesNextContent(localeContent []byte, onMsgid func(string, string, string) error) error {
	var localeData map[string]any
	if err := json.Unmarshal(localeContent, &localeData); err != nil {
		return err
	}
	return iterateMessagesNextInner(onMsgid, localeData, "")
}
