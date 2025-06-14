// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/shared/types"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/translation"
	"forgejo.org/modules/util"

	runnerv1 "code.gitea.io/actions-proto-go/runner/v1"
	"xorm.io/builder"
)

// ActionRunner represents runner machines
//
// It can be:
//  1. global runner, OwnerID is 0 and RepoID is 0
//  2. org/user level runner, OwnerID is org/user ID and RepoID is 0
//  3. repo level runner, OwnerID is 0 and RepoID is repo ID
//
// Please note that it's not acceptable to have both OwnerID and RepoID to be non-zero,
// or it will be complicated to find runners belonging to a specific owner.
// For example, conditions like `OwnerID = 1` will also return runner {OwnerID: 1, RepoID: 1},
// but it's a repo level runner, not an org/user level runner.
// To avoid this, make it clear with {OwnerID: 0, RepoID: 1} for repo level runners.
type ActionRunner struct {
	ID          int64
	UUID        string                 `xorm:"CHAR(36) UNIQUE"`
	Name        string                 `xorm:"VARCHAR(255)"`
	Version     string                 `xorm:"VARCHAR(64)"`
	OwnerID     int64                  `xorm:"index"`
	Owner       *user_model.User       `xorm:"-"`
	RepoID      int64                  `xorm:"index"`
	Repo        *repo_model.Repository `xorm:"-"`
	Description string                 `xorm:"TEXT"`
	Base        int                    // 0 native 1 docker 2 virtual machine
	RepoRange   string                 // glob match which repositories could use this runner

	Token     string `xorm:"-"`
	TokenHash string `xorm:"UNIQUE"` // sha256 of token
	TokenSalt string
	// TokenLastEight string `xorm:"token_last_eight"` // it's unnecessary because we don't find runners by token

	LastOnline timeutil.TimeStamp `xorm:"index"`
	LastActive timeutil.TimeStamp `xorm:"index"`

	// Store labels defined in state file (default: .runner file) of `act_runner`
	AgentLabels []string `xorm:"TEXT"`

	Created timeutil.TimeStamp `xorm:"created"`
	Updated timeutil.TimeStamp `xorm:"updated"`
	Deleted timeutil.TimeStamp `xorm:"deleted"`
}

const (
	RunnerOfflineTime = time.Minute
	RunnerIdleTime    = 10 * time.Second
)

// BelongsToOwnerName before calling, should guarantee that all attributes are loaded
func (r *ActionRunner) BelongsToOwnerName() string {
	if r.RepoID != 0 {
		return r.Repo.FullName()
	}
	if r.OwnerID != 0 {
		return r.Owner.Name
	}
	return ""
}

func (r *ActionRunner) BelongsToOwnerType() types.OwnerType {
	if r.RepoID != 0 {
		return types.OwnerTypeRepository
	}
	if r.OwnerID != 0 {
		switch r.Owner.Type {
		case user_model.UserTypeOrganization:
			return types.OwnerTypeOrganization
		case user_model.UserTypeIndividual:
			return types.OwnerTypeIndividual
		}
	}
	return types.OwnerTypeSystemGlobal
}

// if the logic here changed, you should also modify FindRunnerOptions.ToCond
func (r *ActionRunner) Status() runnerv1.RunnerStatus {
	if time.Since(r.LastOnline.AsTime()) > RunnerOfflineTime {
		return runnerv1.RunnerStatus_RUNNER_STATUS_OFFLINE
	}
	if time.Since(r.LastActive.AsTime()) > RunnerIdleTime {
		return runnerv1.RunnerStatus_RUNNER_STATUS_IDLE
	}
	return runnerv1.RunnerStatus_RUNNER_STATUS_ACTIVE
}

func (r *ActionRunner) StatusName() string {
	return strings.ToLower(strings.TrimPrefix(r.Status().String(), "RUNNER_STATUS_"))
}

func (r *ActionRunner) StatusLocaleName(lang translation.Locale) string {
	return lang.TrString("actions.runners.status." + r.StatusName())
}

func (r *ActionRunner) IsOnline() bool {
	status := r.Status()
	if status == runnerv1.RunnerStatus_RUNNER_STATUS_IDLE || status == runnerv1.RunnerStatus_RUNNER_STATUS_ACTIVE {
		return true
	}
	return false
}

// Editable checks if the runner is editable by the user
func (r *ActionRunner) Editable(ownerID, repoID int64) bool {
	if ownerID == 0 && repoID == 0 {
		return true
	}
	if ownerID > 0 && r.OwnerID == ownerID {
		return true
	}
	return repoID > 0 && r.RepoID == repoID
}

// LoadAttributes loads the attributes of the runner
func (r *ActionRunner) LoadAttributes(ctx context.Context) error {
	if r.OwnerID > 0 {
		var user user_model.User
		has, err := db.GetEngine(ctx).ID(r.OwnerID).Get(&user)
		if err != nil {
			return err
		}
		if has {
			r.Owner = &user
		}
	}
	if r.RepoID > 0 {
		var repo repo_model.Repository
		has, err := db.GetEngine(ctx).ID(r.RepoID).Get(&repo)
		if err != nil {
			return err
		}
		if has {
			r.Repo = &repo
		}
	}
	return nil
}

func (r *ActionRunner) GenerateToken() (err error) {
	r.Token, r.TokenSalt, r.TokenHash, _, err = generateSaltedToken()
	return err
}

// UpdateSecret updates the hash based on the specified token. It does not
// ensure that the runner's UUID matches the first 16 bytes of the token.
func (r *ActionRunner) UpdateSecret(token string) error {
	salt := hex.EncodeToString(util.CryptoRandomBytes(16))

	r.Token = token
	r.TokenSalt = salt
	r.TokenHash = auth_model.HashToken(token, salt)
	return nil
}

func init() {
	db.RegisterModel(&ActionRunner{})
}

type FindRunnerOptions struct {
	db.ListOptions
	RepoID        int64
	OwnerID       int64 // it will be ignored if RepoID is set
	Sort          string
	Filter        string
	IsOnline      optional.Option[bool]
	WithAvailable bool // not only runners belong to, but also runners can be used
}

func (opts FindRunnerOptions) ToConds() builder.Cond {
	cond := builder.NewCond()

	if opts.RepoID > 0 {
		c := builder.NewCond().And(builder.Eq{"repo_id": opts.RepoID})
		if opts.WithAvailable {
			c = c.Or(builder.Eq{"owner_id": builder.Select("owner_id").From("repository").Where(builder.Eq{"id": opts.RepoID})})
			c = c.Or(builder.Eq{"repo_id": 0, "owner_id": 0})
		}
		cond = cond.And(c)
	} else if opts.OwnerID > 0 { // OwnerID is ignored if RepoID is set
		c := builder.NewCond().And(builder.Eq{"owner_id": opts.OwnerID})
		if opts.WithAvailable {
			c = c.Or(builder.Eq{"repo_id": 0, "owner_id": 0})
		}
		cond = cond.And(c)
	}

	if opts.Filter != "" {
		cond = cond.And(builder.Like{"name", opts.Filter})
	}

	if opts.IsOnline.Has() {
		if opts.IsOnline.Value() {
			cond = cond.And(builder.Gt{"last_online": time.Now().Add(-RunnerOfflineTime).Unix()})
		} else {
			cond = cond.And(builder.Lte{"last_online": time.Now().Add(-RunnerOfflineTime).Unix()})
		}
	}
	return cond
}

func (opts FindRunnerOptions) ToOrders() string {
	switch opts.Sort {
	case "online":
		return "last_online DESC"
	case "offline":
		return "last_online ASC"
	case "alphabetically":
		return "name ASC"
	case "reversealphabetically":
		return "name DESC"
	case "newest":
		return "id DESC"
	case "oldest":
		return "id ASC"
	}
	return "last_online DESC"
}

// GetRunnerByUUID returns a runner via uuid
func GetRunnerByUUID(ctx context.Context, uuid string) (*ActionRunner, error) {
	var runner ActionRunner
	has, err := db.GetEngine(ctx).Where("uuid=?", uuid).Get(&runner)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("runner with uuid %s: %w", uuid, util.ErrNotExist)
	}
	return &runner, nil
}

// GetRunnerByID returns a runner via id
func GetRunnerByID(ctx context.Context, id int64) (*ActionRunner, error) {
	var runner ActionRunner
	has, err := db.GetEngine(ctx).Where("id=?", id).Get(&runner)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("runner with id %d: %w", id, util.ErrNotExist)
	}
	return &runner, nil
}

// UpdateRunner updates runner's information.
func UpdateRunner(ctx context.Context, r *ActionRunner, cols ...string) error {
	e := db.GetEngine(ctx)
	r.Name, _ = util.SplitStringAtByteN(r.Name, 255)
	var err error
	if len(cols) == 0 {
		_, err = e.ID(r.ID).AllCols().Update(r)
	} else {
		_, err = e.ID(r.ID).Cols(cols...).Update(r)
	}
	return err
}

// DeleteRunner deletes a runner by given ID.
func DeleteRunner(ctx context.Context, r *ActionRunner) error {
	// Replace the UUID, which was either based on the secret's first 16 bytes or an UUIDv4,
	// with a sequence of 8 0xff bytes followed by the little-endian version of the record's
	// identifier. This will prevent the deleted record's identifier from colliding with any
	// new record.
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(r.ID))
	r.UUID = fmt.Sprintf("ffffffff-ffff-ffff-%.2x%.2x-%.2x%.2x%.2x%.2x%.2x%.2x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7])

	err := UpdateRunner(ctx, r, "UUID")
	if err != nil {
		return err
	}

	_, err = db.DeleteByID[ActionRunner](ctx, r.ID)
	return err
}

// CreateRunner creates new runner.
func CreateRunner(ctx context.Context, t *ActionRunner) error {
	if t.OwnerID != 0 && t.RepoID != 0 {
		// It's trying to create a runner that belongs to a repository, but OwnerID has been set accidentally.
		// Remove OwnerID to avoid confusion; it's not worth returning an error here.
		t.OwnerID = 0
	}
	t.Name, _ = util.SplitStringAtByteN(t.Name, 255)
	return db.Insert(ctx, t)
}

func CountRunnersWithoutBelongingOwner(ctx context.Context) (int64, error) {
	// Only affect action runners were a owner ID is set, as actions runners
	// could also be created on a repository.
	return db.GetEngine(ctx).Table("action_runner").
		Join("LEFT", "`user`", "`action_runner`.owner_id = `user`.id").
		Where("`action_runner`.owner_id != ?", 0).
		And(builder.IsNull{"`user`.id"}).
		Count(new(ActionRunner))
}

func FixRunnersWithoutBelongingOwner(ctx context.Context) (int64, error) {
	subQuery := builder.Select("`action_runner`.id").
		From("`action_runner`").
		Join("LEFT", "`user`", "`action_runner`.owner_id = `user`.id").
		Where(builder.Neq{"`action_runner`.owner_id": 0}).
		And(builder.IsNull{"`user`.id"})
	b := builder.Delete(builder.In("id", subQuery)).From("`action_runner`")
	res, err := db.GetEngine(ctx).Exec(b)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func CountRunnersWithoutBelongingRepo(ctx context.Context) (int64, error) {
	return db.GetEngine(ctx).Table("action_runner").
		Join("LEFT", "`repository`", "`action_runner`.repo_id = `repository`.id").
		Where("`action_runner`.repo_id != ?", 0).
		And(builder.IsNull{"`repository`.id"}).
		Count(new(ActionRunner))
}

func FixRunnersWithoutBelongingRepo(ctx context.Context) (int64, error) {
	subQuery := builder.Select("`action_runner`.id").
		From("`action_runner`").
		Join("LEFT", "`repository`", "`action_runner`.repo_id = `repository`.id").
		Where(builder.Neq{"`action_runner`.repo_id": 0}).
		And(builder.IsNull{"`repository`.id"})
	b := builder.Delete(builder.In("id", subQuery)).From("`action_runner`")
	res, err := db.GetEngine(ctx).Exec(b)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func DeleteOfflineRunners(ctx context.Context, olderThan timeutil.TimeStamp, globalOnly bool) error {
	log.Info("Doing: DeleteOfflineRunners")

	if olderThan.AsTime().After(timeutil.TimeStampNow().AddDuration(-RunnerOfflineTime).AsTime()) {
		return fmt.Errorf("invalid `cron.cleanup_offline_runners.older_than`value: must be at least %q", RunnerOfflineTime)
	}

	cond := builder.Or(
		// never online
		builder.And(builder.Eq{"last_online": 0}, builder.Lt{"created": olderThan}),
		// was online but offline
		builder.And(builder.Gt{"last_online": 0}, builder.Lt{"last_online": olderThan}),
	)

	if globalOnly {
		cond = builder.And(cond, builder.Eq{"owner_id": 0}, builder.Eq{"repo_id": 0})
	}

	if err := db.Iterate(
		ctx,
		cond,
		func(ctx context.Context, r *ActionRunner) error {
			if err := DeleteRunner(ctx, r); err != nil {
				return fmt.Errorf("DeleteOfflineRunners: %w", err)
			}
			lastOnline := r.LastOnline.AsTime()
			olderThanTime := olderThan.AsTime()
			if !lastOnline.IsZero() && lastOnline.Before(olderThanTime) {
				log.Info(
					"Deleted runner [ID: %d, Name: %s], last online %s ago",
					r.ID, r.Name, olderThanTime.Sub(lastOnline).String(),
				)
			} else {
				log.Info(
					"Deleted runner [ID: %d, Name: %s], unused since %s ago",
					r.ID, r.Name, olderThanTime.Sub(r.Created.AsTime()).String(),
				)
			}

			return nil
		},
	); err != nil {
		return err
	}

	log.Info("Finished: DeleteOfflineRunners")

	return nil
}
