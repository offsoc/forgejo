// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"context"
	"math"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"

	"xorm.io/builder"
)

type (
	GroupList []*Group
	Group     struct {
		// Name of the quota group
		Name  string `json:"name" xorm:"pk NOT NULL" binding:"Required"`
		Rules []Rule `json:"rules" xorm:"-"`
	}
)

type GroupRuleMapping struct {
	ID        int64  `xorm:"pk autoincr" json:"-"`
	GroupName string `xorm:"index unique(qgrm_gr) not null" json:"group_name"`
	RuleName  string `xorm:"unique(qgrm_gr) not null" json:"rule_name"`
}

type Kind int

const (
	KindUser Kind = iota
)

type GroupMapping struct {
	ID        int64  `xorm:"pk autoincr"`
	Kind      Kind   `xorm:"unique(qgm_kmg) not null"`
	MappedID  int64  `xorm:"unique(qgm_kmg) not null"`
	GroupName string `xorm:"index unique(qgm_kmg) not null"`
}

func (g *Group) TableName() string {
	return "quota_group"
}

func (grm *GroupRuleMapping) TableName() string {
	return "quota_group_rule_mapping"
}

func (ugm *GroupMapping) TableName() string {
	return "quota_group_mapping"
}

func (g *Group) LoadRules(ctx context.Context) error {
	return db.GetEngine(ctx).Select("`quota_rule`.*").
		Table("quota_rule").
		Join("INNER", "`quota_group_rule_mapping`", "`quota_group_rule_mapping`.rule_name = `quota_rule`.name").
		Where("`quota_group_rule_mapping`.group_name = ?", g.Name).
		Find(&g.Rules)
}

func (g *Group) isUserInGroup(ctx context.Context, userID int64) (bool, error) {
	return db.GetEngine(ctx).
		Where("kind = ? AND mapped_id = ? AND group_name = ?", KindUser, userID, g.Name).
		Get(&GroupMapping{})
}

func (g *Group) AddUserByID(ctx context.Context, userID int64) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	exists, err := g.isUserInGroup(ctx, userID)
	if err != nil {
		return err
	} else if exists {
		return ErrUserAlreadyInGroup{GroupName: g.Name, UserID: userID}
	}

	_, err = db.GetEngine(ctx).Insert(&GroupMapping{
		Kind:      KindUser,
		MappedID:  userID,
		GroupName: g.Name,
	})
	if err != nil {
		return err
	}
	return committer.Commit()
}

func (g *Group) RemoveUserByID(ctx context.Context, userID int64) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	exists, err := g.isUserInGroup(ctx, userID)
	if err != nil {
		return err
	} else if !exists {
		return ErrUserNotInGroup{GroupName: g.Name, UserID: userID}
	}

	_, err = db.GetEngine(ctx).Delete(&GroupMapping{
		Kind:      KindUser,
		MappedID:  userID,
		GroupName: g.Name,
	})
	if err != nil {
		return err
	}
	return committer.Commit()
}

func (g *Group) isRuleInGroup(ctx context.Context, ruleName string) (bool, error) {
	return db.GetEngine(ctx).
		Where("group_name = ? AND rule_name = ?", g.Name, ruleName).
		Get(&GroupRuleMapping{})
}

func (g *Group) AddRuleByName(ctx context.Context, ruleName string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	exists, err := DoesRuleExist(ctx, ruleName)
	if err != nil {
		return err
	} else if !exists {
		return ErrRuleNotFound{Name: ruleName}
	}

	has, err := g.isRuleInGroup(ctx, ruleName)
	if err != nil {
		return err
	} else if has {
		return ErrRuleAlreadyInGroup{GroupName: g.Name, RuleName: ruleName}
	}

	_, err = db.GetEngine(ctx).Insert(&GroupRuleMapping{
		GroupName: g.Name,
		RuleName:  ruleName,
	})
	if err != nil {
		return err
	}
	return committer.Commit()
}

func (g *Group) RemoveRuleByName(ctx context.Context, ruleName string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	exists, err := g.isRuleInGroup(ctx, ruleName)
	if err != nil {
		return err
	} else if !exists {
		return ErrRuleNotInGroup{GroupName: g.Name, RuleName: ruleName}
	}

	_, err = db.GetEngine(ctx).Delete(&GroupRuleMapping{
		GroupName: g.Name,
		RuleName:  ruleName,
	})
	if err != nil {
		return err
	}
	return committer.Commit()
}

var affectsMap = map[LimitSubject]LimitSubjects{
	LimitSubjectSizeAll: {
		LimitSubjectSizeReposAll,
		LimitSubjectSizeGitLFS,
		LimitSubjectSizeAssetsAll,
	},
	LimitSubjectSizeReposAll: {
		LimitSubjectSizeReposPublic,
		LimitSubjectSizeReposPrivate,
	},
	LimitSubjectSizeAssetsAll: {
		LimitSubjectSizeAssetsAttachmentsAll,
		LimitSubjectSizeAssetsArtifacts,
		LimitSubjectSizeAssetsPackagesAll,
	},
	LimitSubjectSizeAssetsAttachmentsAll: {
		LimitSubjectSizeAssetsAttachmentsIssues,
		LimitSubjectSizeAssetsAttachmentsReleases,
	},
}

// Evaluate returns whether the size used is acceptable for the topic if a rule
// was found, and returns the smallest limit of all applicable rules or the
// first limit found to be unacceptable for the size used.
func (g *Group) Evaluate(used Used, forSubject LimitSubject) (bool, bool, int64) {
	var found bool
	foundLimit := int64(math.MaxInt64)
	for _, rule := range g.Rules {
		ok, has := rule.Evaluate(used, forSubject)
		if has {
			if !ok {
				return false, true, rule.Limit
			}
			found = true
			foundLimit = min(foundLimit, rule.Limit)
		}
	}

	if !found {
		// If Evaluation for forSubject did not succeed, try evaluating against
		// subjects below

		for _, subject := range affectsMap[forSubject] {
			ok, has, limit := g.Evaluate(used, subject)
			if has {
				if !ok {
					return false, true, limit
				}
				found = true
				foundLimit = min(foundLimit, limit)
			}
		}
	}

	return true, found, foundLimit
}

// Evaluate returns if the used size is acceptable for the subject and the
// lowest limit that is acceptable for the subject.
func (gl *GroupList) Evaluate(used Used, forSubject LimitSubject) (bool, int64) {
	// If there are no groups, use the configured defaults:
	if gl == nil || len(*gl) == 0 {
		return EvaluateDefault(used, forSubject)
	}

	for _, group := range *gl {
		ok, has, limit := group.Evaluate(used, forSubject)
		if has && ok {
			return true, limit
		}
	}
	return false, 0
}

func GetGroupByName(ctx context.Context, name string) (*Group, error) {
	var group Group
	has, err := db.GetEngine(ctx).Where("name = ?", name).Get(&group)
	if has {
		if err = group.LoadRules(ctx); err != nil {
			return nil, err
		}
		return &group, nil
	}
	return nil, err
}

func ListGroups(ctx context.Context) (GroupList, error) {
	var groups GroupList
	err := db.GetEngine(ctx).Find(&groups)
	return groups, err
}

func doesGroupExist(ctx context.Context, name string) (bool, error) {
	return db.GetEngine(ctx).Where("name = ?", name).Get(&Group{})
}

func CreateGroup(ctx context.Context, name string) (*Group, error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	exists, err := doesGroupExist(ctx, name)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, ErrGroupAlreadyExists{Name: name}
	}

	group := Group{Name: name}
	_, err = db.GetEngine(ctx).Insert(group)
	if err != nil {
		return nil, err
	}
	return &group, committer.Commit()
}

func ListUsersInGroup(ctx context.Context, name string) ([]*user_model.User, error) {
	group, err := GetGroupByName(ctx, name)
	if err != nil {
		return nil, err
	}

	var users []*user_model.User
	err = db.GetEngine(ctx).Select("`user`.*").
		Table("user").
		Join("INNER", "`quota_group_mapping`", "`quota_group_mapping`.mapped_id = `user`.id").
		Where("`quota_group_mapping`.kind = ? AND `quota_group_mapping`.group_name = ?", KindUser, group.Name).
		Find(&users)
	return users, err
}

func DeleteGroupByName(ctx context.Context, name string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	_, err = db.GetEngine(ctx).Delete(GroupMapping{
		GroupName: name,
	})
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).Delete(GroupRuleMapping{
		GroupName: name,
	})
	if err != nil {
		return err
	}

	_, err = db.GetEngine(ctx).Delete(Group{Name: name})
	if err != nil {
		return err
	}
	return committer.Commit()
}

func SetUserGroups(ctx context.Context, userID int64, groups *[]string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	// First: remove the user from any groups
	_, err = db.GetEngine(ctx).Where("kind = ? AND mapped_id = ?", KindUser, userID).Delete(GroupMapping{})
	if err != nil {
		return err
	}

	if groups == nil {
		return nil
	}

	// Then add the user to each group listed
	for _, groupName := range *groups {
		group, err := GetGroupByName(ctx, groupName)
		if err != nil {
			return err
		}
		if group == nil {
			return ErrGroupNotFound{Name: groupName}
		}
		err = group.AddUserByID(ctx, userID)
		if err != nil {
			return err
		}
	}

	return committer.Commit()
}

func GetGroupsForUser(ctx context.Context, userID int64) (GroupList, error) {
	var groups GroupList
	err := db.GetEngine(ctx).
		Where(builder.In("name",
			builder.Select("group_name").
				From("quota_group_mapping").
				Where(builder.And(
					builder.Eq{"kind": KindUser},
					builder.Eq{"mapped_id": userID}),
				))).
		Find(&groups)
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		err = db.GetEngine(ctx).Where(builder.In("name", setting.Quota.DefaultGroups)).Find(&groups)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 {
			return nil, nil
		}
	}

	for _, group := range groups {
		err = group.LoadRules(ctx)
		if err != nil {
			return nil, err
		}
	}

	return groups, nil
}
