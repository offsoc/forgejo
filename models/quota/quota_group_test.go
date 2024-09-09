// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota_test

import (
	"testing"

	quota_model "code.gitea.io/gitea/models/quota"

	"github.com/stretchr/testify/assert"
)

func TestQuotaGroupAllRulesMustPass(t *testing.T) {
	unlimitedRule := quota_model.Rule{
		Limit: -1,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}
	denyRule := quota_model.Rule{
		Limit: 0,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}
	group := quota_model.Group{
		Rules: []quota_model.Rule{
			unlimitedRule,
			denyRule,
		},
	}

	used := quota_model.Used{}
	used.Size.Repos.Public = 1024

	// Within a group, *all* rules must pass. Thus, if we have a deny-all rule,
	// and an unlimited rule, that will always fail.
	ok, has := group.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.True(t, has)
	assert.False(t, ok)
}

func TestQuotaGroupRuleScenario1(t *testing.T) {
	group := quota_model.Group{
		Rules: []quota_model.Rule{
			{
				Limit: 1024,
				Subjects: quota_model.LimitSubjects{
					quota_model.LimitSubjectSizeAssetsAttachmentsReleases,
					quota_model.LimitSubjectSizeGitLFS,
					quota_model.LimitSubjectSizeAssetsPackagesAll,
				},
			},
			{
				Limit: 0,
				Subjects: quota_model.LimitSubjects{
					quota_model.LimitSubjectSizeGitLFS,
				},
			},
		},
	}

	used := quota_model.Used{}
	used.Size.Assets.Attachments.Releases = 512
	used.Size.Assets.Packages.All = 256
	used.Size.Git.LFS = 16

	ok, has := group.Evaluate(used, quota_model.LimitSubjectSizeAssetsAttachmentsReleases)
	assert.True(t, has, "size:assets:attachments:releases is covered")
	assert.True(t, ok, "size:assets:attachments:releases passes")

	ok, has = group.Evaluate(used, quota_model.LimitSubjectSizeAssetsPackagesAll)
	assert.True(t, has, "size:assets:packages:all is covered")
	assert.True(t, ok, "size:assets:packages:all passes")

	ok, has = group.Evaluate(used, quota_model.LimitSubjectSizeGitLFS)
	assert.True(t, has, "size:git:lfs is covered")
	assert.False(t, ok, "size:git:lfs fails")

	ok, has = group.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.True(t, has, "size:all is covered")
	assert.False(t, ok, "size:all fails")
}

func TestQuotaGroupRuleCombination(t *testing.T) {
	repoRule := quota_model.Rule{
		Limit: 4096,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeReposAll,
		},
	}
	packagesRule := quota_model.Rule{
		Limit: 0,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAssetsPackagesAll,
		},
	}

	used := quota_model.Used{}
	used.Size.Repos.Public = 1024
	used.Size.Assets.Packages.All = 1024

	group := quota_model.Group{
		Rules: []quota_model.Rule{
			repoRule,
			packagesRule,
		},
	}

	// Git LFS isn't covered by any rule
	_, has := group.Evaluate(used, quota_model.LimitSubjectSizeGitLFS)
	assert.False(t, has)

	// repos:all is covered, and is passing
	ok, has := group.Evaluate(used, quota_model.LimitSubjectSizeReposAll)
	assert.True(t, has)
	assert.True(t, ok)

	// packages:all is covered, and is failing
	ok, has = group.Evaluate(used, quota_model.LimitSubjectSizeAssetsPackagesAll)
	assert.True(t, has)
	assert.False(t, ok)

	// size:all is covered, and is failing (due to packages:all being over quota)
	ok, has = group.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.True(t, has, "size:all should be covered")
	assert.False(t, ok, "size:all should fail")
}

func TestQuotaGroupListsRequireOnlyOnePassing(t *testing.T) {
	unlimitedRule := quota_model.Rule{
		Limit: -1,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}
	denyRule := quota_model.Rule{
		Limit: 0,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}

	denyGroup := quota_model.Group{
		Rules: []quota_model.Rule{
			denyRule,
		},
	}
	unlimitedGroup := quota_model.Group{
		Rules: []quota_model.Rule{
			unlimitedRule,
		},
	}

	groups := quota_model.GroupList{&denyGroup, &unlimitedGroup}

	used := quota_model.Used{}
	used.Size.Repos.Public = 1024

	// In a group list, if any group passes, the entire evaluation passes.
	ok := groups.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.True(t, ok)
}

func TestQuotaGroupListAllFailing(t *testing.T) {
	denyRule := quota_model.Rule{
		Limit: 0,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}
	limitedRule := quota_model.Rule{
		Limit: 1024,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAll,
		},
	}

	denyGroup := quota_model.Group{
		Rules: []quota_model.Rule{
			denyRule,
		},
	}
	limitedGroup := quota_model.Group{
		Rules: []quota_model.Rule{
			limitedRule,
		},
	}

	groups := quota_model.GroupList{&denyGroup, &limitedGroup}

	used := quota_model.Used{}
	used.Size.Repos.Public = 2048

	ok := groups.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.False(t, ok)
}

func TestQuotaGroupListEmpty(t *testing.T) {
	groups := quota_model.GroupList{}

	used := quota_model.Used{}
	used.Size.Repos.Public = 2048

	ok := groups.Evaluate(used, quota_model.LimitSubjectSizeAll)
	assert.True(t, ok)
}
