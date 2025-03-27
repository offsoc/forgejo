// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota_test

import (
	"testing"

	quota_model "forgejo.org/models/quota"

	"github.com/stretchr/testify/assert"
)

func makeFullyUsed() quota_model.Used {
	return quota_model.Used{
		Size: quota_model.UsedSize{
			Repos: quota_model.UsedSizeRepos{
				Public:  1024,
				Private: 1024,
			},
			Git: quota_model.UsedSizeGit{
				LFS: 1024,
			},
			Assets: quota_model.UsedSizeAssets{
				Attachments: quota_model.UsedSizeAssetsAttachments{
					Issues:   1024,
					Releases: 1024,
				},
				Artifacts: 1024,
				Packages: quota_model.UsedSizeAssetsPackages{
					All: 1024,
				},
			},
		},
	}
}

func makePartiallyUsed() quota_model.Used {
	return quota_model.Used{
		Size: quota_model.UsedSize{
			Repos: quota_model.UsedSizeRepos{
				Public: 1024,
			},
			Assets: quota_model.UsedSizeAssets{
				Attachments: quota_model.UsedSizeAssetsAttachments{
					Releases: 1024,
				},
			},
		},
	}
}

func setUsed(used quota_model.Used, subject quota_model.LimitSubject, value int64) *quota_model.Used {
	switch subject {
	case quota_model.LimitSubjectSizeReposPublic:
		used.Size.Repos.Public = value
		return &used
	case quota_model.LimitSubjectSizeReposPrivate:
		used.Size.Repos.Private = value
		return &used
	case quota_model.LimitSubjectSizeGitLFS:
		used.Size.Git.LFS = value
		return &used
	case quota_model.LimitSubjectSizeAssetsAttachmentsIssues:
		used.Size.Assets.Attachments.Issues = value
		return &used
	case quota_model.LimitSubjectSizeAssetsAttachmentsReleases:
		used.Size.Assets.Attachments.Releases = value
		return &used
	case quota_model.LimitSubjectSizeAssetsArtifacts:
		used.Size.Assets.Artifacts = value
		return &used
	case quota_model.LimitSubjectSizeAssetsPackagesAll:
		used.Size.Assets.Packages.All = value
		return &used
	case quota_model.LimitSubjectSizeWiki:
	}

	return nil
}

func assertEvaluation(t *testing.T, rule quota_model.Rule, used quota_model.Used, subject quota_model.LimitSubject, expected bool) {
	t.Helper()

	t.Run(subject.String(), func(t *testing.T) {
		ok, has := rule.Evaluate(used, subject)
		assert.True(t, has)
		assert.Equal(t, expected, ok)
	})
}

func TestQuotaRuleNoEvaluation(t *testing.T) {
	rule := quota_model.Rule{
		Limit: 1024,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeAssetsAttachmentsAll,
		},
	}
	used := quota_model.Used{}
	used.Size.Repos.Public = 4096

	_, has := rule.Evaluate(used, quota_model.LimitSubjectSizeReposAll)

	// We have a rule for "size:assets:attachments:all", and query for
	// "size:repos:all". We don't cover that subject, so the evaluation returns
	// with no rules found.
	assert.False(t, has)
}

func TestQuotaRuleDirectEvaluation(t *testing.T) {
	// This function is meant to test direct rule evaluation: cases where we set
	// a rule for a subject, and we evaluate against the same subject.

	runTest := func(t *testing.T, subject quota_model.LimitSubject, limit, used int64, expected bool) {
		t.Helper()

		rule := quota_model.Rule{
			Limit: limit,
			Subjects: quota_model.LimitSubjects{
				subject,
			},
		}
		usedObj := setUsed(quota_model.Used{}, subject, used)
		if usedObj == nil {
			return
		}

		assertEvaluation(t, rule, *usedObj, subject, expected)
	}

	t.Run("limit:0", func(t *testing.T) {
		// With limit:0, nothing used is fine.
		t.Run("used:0", func(t *testing.T) {
			for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
				runTest(t, subject, 0, 0, true)
			}
		})
		// With limit:0, any usage will fail evaluation
		t.Run("used:512", func(t *testing.T) {
			for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
				runTest(t, subject, 0, 512, false)
			}
		})
	})

	t.Run("limit:unlimited", func(t *testing.T) {
		// With no limits, any usage will succeed evaluation
		t.Run("used:512", func(t *testing.T) {
			for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
				runTest(t, subject, -1, 512, true)
			}
		})
	})

	t.Run("limit:1024", func(t *testing.T) {
		// With a set limit, usage below the limit succeeds
		t.Run("used:512", func(t *testing.T) {
			for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
				runTest(t, subject, 1024, 512, true)
			}
		})

		// With a set limit, usage above the limit fails
		t.Run("used:2048", func(t *testing.T) {
			for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
				runTest(t, subject, 1024, 2048, false)
			}
		})
	})
}

func TestQuotaRuleCombined(t *testing.T) {
	rule := quota_model.Rule{
		Limit: 1024,
		Subjects: quota_model.LimitSubjects{
			quota_model.LimitSubjectSizeGitLFS,
			quota_model.LimitSubjectSizeAssetsAttachmentsReleases,
			quota_model.LimitSubjectSizeAssetsPackagesAll,
		},
	}
	used := quota_model.Used{
		Size: quota_model.UsedSize{
			Repos: quota_model.UsedSizeRepos{
				Public: 4096,
			},
			Git: quota_model.UsedSizeGit{
				LFS: 256,
			},
			Assets: quota_model.UsedSizeAssets{
				Attachments: quota_model.UsedSizeAssetsAttachments{
					Issues:   2048,
					Releases: 256,
				},
				Packages: quota_model.UsedSizeAssetsPackages{
					All: 2560,
				},
			},
		},
	}

	expectationMap := map[quota_model.LimitSubject]bool{
		quota_model.LimitSubjectSizeGitLFS:                    false,
		quota_model.LimitSubjectSizeAssetsAttachmentsReleases: false,
		quota_model.LimitSubjectSizeAssetsPackagesAll:         false,
	}

	for subject := quota_model.LimitSubjectFirst; subject <= quota_model.LimitSubjectLast; subject++ {
		t.Run(subject.String(), func(t *testing.T) {
			evalOk, evalHas := rule.Evaluate(used, subject)
			expected, expectedHas := expectationMap[subject]

			assert.Equal(t, expectedHas, evalHas)
			if expectedHas {
				assert.Equal(t, expected, evalOk)
			}
		})
	}
}

func TestQuotaRuleSizeAll(t *testing.T) {
	runTests := func(t *testing.T, rule quota_model.Rule, expected bool) {
		t.Helper()

		subject := quota_model.LimitSubjectSizeAll

		t.Run("used:0", func(t *testing.T) {
			used := quota_model.Used{}

			assertEvaluation(t, rule, used, subject, true)
		})

		t.Run("used:some-each", func(t *testing.T) {
			used := makeFullyUsed()

			assertEvaluation(t, rule, used, subject, expected)
		})

		t.Run("used:some", func(t *testing.T) {
			used := makePartiallyUsed()

			assertEvaluation(t, rule, used, subject, expected)
		})
	}

	// With all limits set to 0, evaluation always fails if usage > 0
	t.Run("rule:0", func(t *testing.T) {
		rule := quota_model.Rule{
			Limit: 0,
			Subjects: quota_model.LimitSubjects{
				quota_model.LimitSubjectSizeAll,
			},
		}

		runTests(t, rule, false)
	})

	// With no limits, evaluation always succeeds
	t.Run("rule:unlimited", func(t *testing.T) {
		rule := quota_model.Rule{
			Limit: -1,
			Subjects: quota_model.LimitSubjects{
				quota_model.LimitSubjectSizeAll,
			},
		}

		runTests(t, rule, true)
	})

	// With a specific, very generous limit, evaluation succeeds if the limit isn't exhausted
	t.Run("rule:generous", func(t *testing.T) {
		rule := quota_model.Rule{
			Limit: 102400,
			Subjects: quota_model.LimitSubjects{
				quota_model.LimitSubjectSizeAll,
			},
		}

		runTests(t, rule, true)

		t.Run("limit exhaustion", func(t *testing.T) {
			used := quota_model.Used{
				Size: quota_model.UsedSize{
					Repos: quota_model.UsedSizeRepos{
						Public: 204800,
					},
				},
			}

			assertEvaluation(t, rule, used, quota_model.LimitSubjectSizeAll, false)
		})
	})

	// With a specific, small limit, evaluation fails
	t.Run("rule:limited", func(t *testing.T) {
		rule := quota_model.Rule{
			Limit: 512,
			Subjects: quota_model.LimitSubjects{
				quota_model.LimitSubjectSizeAll,
			},
		}

		runTests(t, rule, false)
	})
}
