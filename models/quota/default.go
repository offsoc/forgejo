// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"forgejo.org/modules/setting"
)

func EvaluateDefault(used Used, forSubject LimitSubject) (bool, int64) {
	groups := GroupList{
		&Group{
			Name: "builtin-default-group",
			Rules: []Rule{
				{
					Name:     "builtin-default-rule",
					Limit:    setting.Quota.Default.Total,
					Subjects: LimitSubjects{LimitSubjectSizeAll},
				},
			},
		},
	}

	return groups.Evaluate(used, forSubject)
}
