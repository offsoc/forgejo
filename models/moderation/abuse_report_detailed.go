// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type AbuseReportDetailed struct {
	AbuseReport        `xorm:"extends"`
	ReportedTimes      int // only for overview
	ReporterName       string
	ContentReference   string
	ShadowCopyDate     timeutil.TimeStamp // only for details
	ShadowCopyRawValue string             // only for details

	ShadowCopyMap string
}

func (AbuseReportDetailed) TableName() string {
	return "abuse_report"
}

func (ard AbuseReportDetailed) ContentTypeIconName() string {
	switch ard.ContentType {
	case ReportedContentTypeUser:
		return "octicon-person"
	case ReportedContentTypeRepository:
		return "octicon-repo"
	case ReportedContentTypeIssue:
		return "octicon-issue-opened"
	case ReportedContentTypeComment:
		return "octicon-comment"
	default:
		return "octicon-question"
	}
}

func GetOpenReports(ctx context.Context) ([]*AbuseReportDetailed, error) {
	var reports []*AbuseReportDetailed
	err := db.GetEngine(ctx).SQL(`SELECT AR.*, COUNT(AR.id) AS 'reported_times', U.name AS 'reporter_name', REFS.ref AS 'content_reference'
		FROM abuse_report AR
		LEFT JOIN user U ON U.id = AR.reporter_id
		INNER JOIN (
			SELECT 1 AS 'type', id, concat('@', name) AS 'ref'
			FROM user WHERE id IN (
				SELECT content_id FROM abuse_report WHERE status = 1 AND content_type = 1
			)
			UNION
			SELECT 2 AS 'type', id, concat('/', owner_name, '/', name) AS 'ref'
			FROM repository WHERE id IN (
				SELECT content_id FROM abuse_report WHERE status = 1 AND content_type = 2
			)
			UNION
			SELECT 3 AS 'type', I.id, concat(IR.owner_name, '/', IR.name, '#', I.'index') AS 'ref'
			FROM issue I
			LEFT JOIN repository IR ON IR.id = I.repo_id
			WHERE I.id IN (
				SELECT content_id FROM abuse_report WHERE status = 1 AND content_type = 3
			)
			UNION
			SELECT 4 AS 'type', C.id, concat('/', CIR.owner_name, '/', CIR.name, '/issues/', CI.'index', '#issuecomment-', C.id) AS 'ref'
			FROM comment C
			LEFT JOIN issue CI ON CI.id = C.issue_id
			LEFT JOIN repository CIR ON CIR.id = CI.repo_id
			WHERE C.id IN (
				SELECT content_id FROM abuse_report WHERE status = 1 AND content_type = 4
			)
		) REFS ON REFS.type = AR.content_type AND REFS.id = AR.content_id
		WHERE AR.status = 1
		GROUP BY AR.content_type, AR.content_id
		ORDER BY AR.created_unix ASC`).
		Find(&reports)
	if err != nil {
		return nil, err
	}
	return reports, nil
}
