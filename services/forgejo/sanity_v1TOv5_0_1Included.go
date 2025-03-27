// SPDX-License-Identifier: MIT

package forgejo

import (
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/models/forgejo/semver"
	"forgejo.org/modules/setting"

	"github.com/hashicorp/go-version"
)

var v1TOv5_0_1IncludedStorageSections = []struct {
	section        string
	storageSection string
}{
	{"attachment", "storage.attachments"},
	{"lfs", "storage.lfs"},
	{"avatar", "storage.avatars"},
	{"repo-avatar", "storage.repo-avatars"},
	{"repo-archive", "storage.repo-archive"},
	{"packages", "storage.packages"},
	// the actions sections are not included here because they were experimental at the time
}

func v1TOv5_0_1Included(e db.Engine, dbVersion int64, cfg setting.ConfigProvider) error {
	//
	// When upgrading from Forgejo > v5 or Gitea > v1.20, no sanity check is necessary
	//
	if dbVersion > ForgejoV5DatabaseVersion {
		return nil
	}

	//
	// When upgrading from a Forgejo point version >= v5.0.1, no sanity
	// check is necessary
	//
	// When upgrading from a Gitea >= v1.20 the sanitiy checks will
	// always be done They are necessary for Gitea [v1.20.0..v1.20.2]
	// but not for [v1.20.3..] but there is no way to know which point
	// release was running prior to the upgrade.  This may require the
	// Gitea admin to update their app.ini although it is not necessary
	// but will have no other consequence.
	//
	previousServerVersion, err := semver.GetVersionWithEngine(e)
	if err != nil {
		return err
	}
	upper, err := version.NewVersion("v5.0.1")
	if err != nil {
		return err
	}

	if previousServerVersion.GreaterThan(upper) {
		return nil
	}

	//
	// Sanity checks
	//

	originalCfg, err := cfg.PrepareSaving()
	if err != nil {
		return err
	}

	messages := make([]string, 0, 10)
	for _, c := range v1TOv5_0_1IncludedStorageSections {
		section, _ := originalCfg.GetSection(c.section)
		if section == nil {
			continue
		}
		storageSection, _ := originalCfg.GetSection(c.storageSection)
		if storageSection == nil {
			continue
		}
		messages = append(messages, fmt.Sprintf("[%s] and [%s] may conflict with each other", c.section, c.storageSection))
	}

	if originalCfg.Section("storage").HasKey("PATH") {
		messages = append(messages, "[storage].PATH is set and may create storage issues")
	}

	if len(messages) > 0 {
		return fatal(fmt.Errorf("%s\nThese issues need to be manually fixed in the app.ini file at %s. Please read https://forgejo.org/2023-08-release-v1-20-3-0/ for instructions", strings.Join(messages, "\n"), cfg.GetFile()))
	}
	return nil
}
