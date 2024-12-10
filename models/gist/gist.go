// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/google/uuid"
)

type GistVisibility int8 //revive:disable-line:exported

const (
	GistVisibilityPublic  GistVisibility = iota + 1 // 1， GistVisibilityPublic the gist can be seen be anyone
	GistVisibilityHidden                            // 2， ArtifactStatusUploadConfirmed the gist can be seen by anyone but don't appear in the search
	GistVisibilityPrivate                           // 3， ArtifactStatusUploadError teh gist can only been seen by the owner
)

func (visibility GistVisibility) String() string {
	switch visibility {
	case GistVisibilityPublic:
		return "public"
	case GistVisibilityHidden:
		return "hidden"
	case GistVisibilityPrivate:
		return "private"
	default:
		return "unknown"
	}
}

func GistVisibilityFromName(name string) (GistVisibility, error) { //revive:disable-line:exported
	switch strings.ToLower(name) {
	case "public":
		return GistVisibilityPublic, nil
	case "hidden":
		return GistVisibilityHidden, nil
	case "private":
		return GistVisibilityPrivate, nil
	default:
		return 0, fmt.Errorf("%s is not a valid gist visibiklity name", name)
	}
}

// ErrGistNotExist represents a "GistNotExist" kind of error.
type ErrGistNotExist struct {
	UUID string
}

// IsErrGistNotExist checks if an error is a ErrGistNotExist.
func IsErrGistNotExist(err error) bool {
	_, ok := err.(ErrGistNotExist)
	return ok
}

func (err ErrGistNotExist) Error() string {
	return fmt.Sprintf("gist does not exists [uuid: %s]", err.UUID)
}

type Gist struct {
	ID          int64            `xorm:"pk autoincr"`
	OwnerID     int64            `xorm:"index"`
	Owner       *user_model.User `xorm:"-"`
	UUID        string           `xorm:"UNIQUE"`
	Name        string
	Description string `xorm:"TEXT"`
	Visibility  GistVisibility
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Gist))
}

// generateUUID generates a random UUID for a Gist
func generateUUID() string {
	uuidParts := strings.Split(uuid.New().String(), "-")
	return strings.ToLower(uuidParts[0])
}

// Create creates a new Gist
func Create(ctx context.Context, gist *Gist) error {
	gist.UUID = generateUUID()
	_, err := db.GetEngine(ctx).Insert(gist)
	return err
}

// GetGistByUUID finds the Gist with the given UUID
func GetGistByUUID(ctx context.Context, uuid string) (*Gist, error) {
	gist := new(Gist)
	has, err := db.GetEngine(ctx).Where("uuid = ?", strings.ToLower(uuid)).Get(gist)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrGistNotExist{UUID: uuid}
	}

	return gist, nil
}

// CountOwnerGists retruns how many Gists a User Owns
// Note: This function does not check if the caller has permission to view the Gists
func CountOwnerGists(ctx context.Context, ownerID int64) (int64, error) {
	return db.GetEngine(ctx).Where("owner_id = ?", ownerID).Count(new(Gist))
}

// GetRepoPath returns the Path to the Gist Repo
func (gist *Gist) GetRepoPath() string {
	return filepath.Join(setting.Gist.RootPath, fmt.Sprintf("%s.git", gist.UUID))
}

// Link returns the Link to the Repo
func (gist *Gist) Link() string {
	return fmt.Sprintf("/gists/%s", url.PathEscape(gist.UUID))
}

// HTMLURL returns the gist HTML URL
func (gist *Gist) HTMLURL() string {
	return fmt.Sprintf("%sgists/%s", setting.AppURL, url.PathEscape(gist.UUID))
}

// LoadOwner loads the owner field
func (gist *Gist) LoadOwner(ctx context.Context) error {
	owner, err := user_model.GetUserByID(ctx, gist.OwnerID)
	if err != nil {
		return err
	}

	gist.Owner = owner

	return nil
}

// Update cols updates the given columns
func (gist *Gist) UpdateCols(ctx context.Context, cols ...string) error {
	_, err := db.GetEngine(ctx).ID(gist.ID).Cols(cols...).Update(gist)
	return err
}

// IsOwner checks if the given User is the Owner of the Repo
func (gist *Gist) IsOwner(user *user_model.User) bool {
	if user == nil {
		return false
	}

	return gist.OwnerID == user.ID
}

// HasAccess checks if the given User has access to the Gist
func (gist *Gist) HasAccess(user *user_model.User) bool {
	if gist.Visibility != GistVisibilityPrivate {
		return true
	}

	return gist.IsOwner(user)
}
