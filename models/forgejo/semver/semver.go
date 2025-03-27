// SPDX-License-Identifier: MIT

package semver

import (
	"context"

	"forgejo.org/models/db"

	"github.com/hashicorp/go-version"
)

func init() {
	db.RegisterModel(new(ForgejoSemVer))
}

var DefaultVersionString = "1.0.0"

type ForgejoSemVer struct {
	Version string
}

func GetVersion(ctx context.Context) (*version.Version, error) {
	return GetVersionWithEngine(db.GetEngine(ctx))
}

func GetVersionWithEngine(e db.Engine) (*version.Version, error) {
	versionString := DefaultVersionString

	exists, err := e.IsTableExist("forgejo_sem_ver")
	if err != nil {
		return nil, err
	}
	if exists {
		var semver ForgejoSemVer
		has, err := e.Get(&semver)
		if err != nil {
			return nil, err
		} else if has {
			versionString = semver.Version
		}
	}

	v, err := version.NewVersion(versionString)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func SetVersionString(ctx context.Context, versionString string) error {
	return SetVersionStringWithEngine(db.GetEngine(ctx), versionString)
}

func SetVersionStringWithEngine(e db.Engine, versionString string) error {
	v, err := version.NewVersion(versionString)
	if err != nil {
		return err
	}
	return SetVersionWithEngine(e, v)
}

func SetVersion(ctx context.Context, v *version.Version) error {
	return SetVersionWithEngine(db.GetEngine(ctx), v)
}

func SetVersionWithEngine(e db.Engine, v *version.Version) error {
	var semver ForgejoSemVer
	has, err := e.Get(&semver)
	if err != nil {
		return err
	}

	if !has {
		_, err = e.Exec("insert into forgejo_sem_ver values (?)", v.String())
	} else {
		_, err = e.Exec("update forgejo_sem_ver set version = ?", v.String())
	}
	return err
}
