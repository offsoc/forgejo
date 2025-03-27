// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"

	"github.com/go-enry/go-enry/v2"
)

// LanguageStat describes language statistics of a repository
type LanguageStat struct {
	ID          int64 `xorm:"pk autoincr"`
	RepoID      int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CommitID    string
	IsPrimary   bool
	Language    string             `xorm:"VARCHAR(50) UNIQUE(s) INDEX NOT NULL"`
	Percentage  float32            `xorm:"-"`
	Size        int64              `xorm:"NOT NULL DEFAULT 0"`
	Color       string             `xorm:"-"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX CREATED"`
}

func init() {
	db.RegisterModel(new(LanguageStat))
}

// LanguageStatList defines a list of language statistics
type LanguageStatList []*LanguageStat

// LoadAttributes loads attributes
func (stats LanguageStatList) LoadAttributes() {
	for i := range stats {
		stats[i].Color = enry.GetColor(stats[i].Language)
	}
}

func (stats LanguageStatList) getLanguagePercentages() map[string]float32 {
	langPerc := make(map[string]float32)
	var otherPerc float32
	var total int64

	for _, stat := range stats {
		total += stat.Size
	}
	if total > 0 {
		for _, stat := range stats {
			perc := float32(float64(stat.Size) / float64(total) * 100)
			if perc <= 0.1 {
				otherPerc += perc
				continue
			}
			langPerc[stat.Language] = perc
		}
	}
	if otherPerc > 0 {
		langPerc["other"] = otherPerc
	}
	roundByLargestRemainder(langPerc, 100)
	return langPerc
}

// Use the quota method to round the percentages to one decimal place while
// keeping the sum of the percentages at 100%.
func roundByLargestRemainder(percs map[string]float32, target float32) {
	// Tracks the difference between the sum of percentage and 100%.
	leftToDistribute := int(target * 10)

	type key struct {
		language  string
		remainder float64
	}
	keys := make([]key, 0, len(percs))

	for k, v := range percs {
		floored, frac := math.Modf(float64(v * 10))
		percs[k] = float32(floored)
		leftToDistribute -= int(floored)
		keys = append(keys, key{language: k, remainder: frac})
	}

	// Sort the fractional part in an ascending order.
	slices.SortFunc(keys, func(b, a key) int {
		return cmp.Compare(a.remainder, b.remainder)
	})

	// As long as the sum of 100% is not reached, add 0.1% percentage.
	for _, k := range keys {
		if leftToDistribute > 0 {
			percs[k.language]++
			leftToDistribute--
		}
		percs[k.language] /= 10
	}
}

// GetLanguageStats returns the language statistics for a repository
func GetLanguageStats(ctx context.Context, repo *Repository) (LanguageStatList, error) {
	stats := make(LanguageStatList, 0, 6)
	if err := db.GetEngine(ctx).Where("`repo_id` = ?", repo.ID).Desc("`size`").Find(&stats); err != nil {
		return nil, err
	}
	return stats, nil
}

// GetTopLanguageStats returns the top language statistics for a repository
func GetTopLanguageStats(ctx context.Context, repo *Repository, limit int) (LanguageStatList, error) {
	stats, err := GetLanguageStats(ctx, repo)
	if err != nil {
		return nil, err
	}
	perc := stats.getLanguagePercentages()
	topstats := make(LanguageStatList, 0, limit)
	var other float32
	for i := range stats {
		if _, ok := perc[stats[i].Language]; !ok {
			continue
		}
		if stats[i].Language == "other" || len(topstats) >= limit {
			other += perc[stats[i].Language]
			continue
		}
		stats[i].Percentage = perc[stats[i].Language]
		topstats = append(topstats, stats[i])
	}
	if other > 0 {
		topstats = append(topstats, &LanguageStat{
			RepoID:     repo.ID,
			Language:   "other",
			Color:      "#cccccc",
			Percentage: float32(math.Round(float64(other)*10) / 10),
		})
	}
	topstats.LoadAttributes()
	return topstats, nil
}

// UpdateLanguageStats updates the language statistics for repository
func UpdateLanguageStats(ctx context.Context, repo *Repository, commitID string, stats map[string]int64) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	oldstats, err := GetLanguageStats(ctx, repo)
	if err != nil {
		return err
	}
	var topLang string
	var s int64
	for lang, size := range stats {
		if size > s {
			s = size
			topLang = strings.ToLower(lang)
		}
	}

	for lang, size := range stats {
		upd := false
		llang := strings.ToLower(lang)
		for _, s := range oldstats {
			// Update already existing language
			if strings.ToLower(s.Language) == llang {
				s.CommitID = commitID
				s.IsPrimary = llang == topLang
				s.Size = size
				if _, err := sess.ID(s.ID).Cols("`commit_id`", "`size`", "`is_primary`").Update(s); err != nil {
					return err
				}
				upd = true
				break
			}
		}
		// Insert new language
		if !upd {
			if err := db.Insert(ctx, &LanguageStat{
				RepoID:    repo.ID,
				CommitID:  commitID,
				IsPrimary: llang == topLang,
				Language:  lang,
				Size:      size,
			}); err != nil {
				return err
			}
		}
	}
	// Delete old languages
	statsToDelete := make([]int64, 0, len(oldstats))
	for _, s := range oldstats {
		if s.CommitID != commitID {
			statsToDelete = append(statsToDelete, s.ID)
		}
	}
	if len(statsToDelete) > 0 {
		if _, err := sess.In("`id`", statsToDelete).Delete(&LanguageStat{}); err != nil {
			return err
		}
	}

	// Update indexer status
	if err = UpdateIndexerStatus(ctx, repo, RepoIndexerTypeStats, commitID); err != nil {
		return err
	}

	return committer.Commit()
}

// CopyLanguageStat Copy originalRepo language stat information to destRepo (use for forked repo)
func CopyLanguageStat(ctx context.Context, originalRepo, destRepo *Repository) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	RepoLang := make(LanguageStatList, 0, 6)
	if err := db.GetEngine(ctx).Where("`repo_id` = ?", originalRepo.ID).Desc("`size`").Find(&RepoLang); err != nil {
		return err
	}
	if len(RepoLang) > 0 {
		for i := range RepoLang {
			RepoLang[i].ID = 0
			RepoLang[i].RepoID = destRepo.ID
			RepoLang[i].CreatedUnix = timeutil.TimeStampNow()
		}
		// update destRepo's indexer status
		tmpCommitID := RepoLang[0].CommitID
		if err := UpdateIndexerStatus(ctx, destRepo, RepoIndexerTypeStats, tmpCommitID); err != nil {
			return err
		}
		if err := db.Insert(ctx, &RepoLang); err != nil {
			return err
		}
	}
	return committer.Commit()
}
