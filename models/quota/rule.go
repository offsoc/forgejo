// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"context"
	"slices"

	"code.gitea.io/gitea/models/db"
)

type Rule struct {
	Name     string        `xorm:"pk not null" json:"name,omitempty"`
	Limit    int64         `xorm:"NOT NULL" binding:"Required" json:"limit"`
	Subjects LimitSubjects `json:"subjects,omitempty"`
}

func (r *Rule) TableName() string {
	return "quota_rule"
}

func (r Rule) Acceptable(used Used) bool {
	if r.Limit == -1 {
		return true
	}

	return r.Sum(used) <= r.Limit
}

func (r Rule) Sum(used Used) int64 {
	var sum int64
	for _, subject := range r.Subjects {
		sum += used.CalculateFor(subject)
	}
	return sum
}

func (r Rule) Evaluate(used Used, forSubject LimitSubject) (bool, bool) {
	// If there's no limit, short circuit out
	if r.Limit == -1 {
		return true, true
	}

	// If the rule does not cover forSubject, bail out early
	if !slices.Contains(r.Subjects, forSubject) {
		return false, false
	}

	return r.Sum(used) <= r.Limit, true
}

func (r *Rule) Edit(ctx context.Context, limit *int64, subjects *LimitSubjects) (*Rule, error) {
	cols := []string{}

	if limit != nil {
		r.Limit = *limit
		cols = append(cols, "limit")
	}
	if subjects != nil {
		r.Subjects = *subjects
		cols = append(cols, "subjects")
	}

	_, err := db.GetEngine(ctx).Where("name = ?", r.Name).Cols(cols...).Update(r)
	return r, err
}

func GetRuleByName(ctx context.Context, name string) (*Rule, error) {
	var rule Rule
	has, err := db.GetEngine(ctx).Where("name = ?", name).Get(&rule)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return &rule, err
}

func ListRules(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	err := db.GetEngine(ctx).Find(&rules)
	return rules, err
}

func DoesRuleExist(ctx context.Context, name string) (bool, error) {
	return db.GetEngine(ctx).
		Where("name = ?", name).
		Get(&Rule{})
}

func CreateRule(ctx context.Context, name string, limit int64, subjects LimitSubjects) (*Rule, error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	exists, err := DoesRuleExist(ctx, name)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, ErrRuleAlreadyExists{Name: name}
	}

	rule := Rule{
		Name:     name,
		Limit:    limit,
		Subjects: subjects,
	}
	_, err = db.GetEngine(ctx).Insert(rule)
	if err != nil {
		return nil, err
	}

	return &rule, committer.Commit()
}

func DeleteRuleByName(ctx context.Context, name string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	_, err = db.GetEngine(ctx).Delete(GroupRuleMapping{
		RuleName: name,
	})
	if err != nil {
		return err
	}

	_, err = db.GetEngine(ctx).Delete(Rule{Name: name})
	if err != nil {
		return err
	}
	return committer.Commit()
}
