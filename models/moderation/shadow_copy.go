// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"context"
	"database/sql"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

type AbuseReportShadowCopy struct {
	ID          int64              `xorm:"pk autoincr"`
	RawValue    string             `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL"`
}

// Returns the ID encapsulated in a sql.NullInt64 struct.
func (sc AbuseReportShadowCopy) NullableID() sql.NullInt64 {
	return sql.NullInt64{Int64: sc.ID, Valid: sc.ID > 0}
}

func init() {
	// RegisterModel will create the table if does not already exist
	// or any missing columns if the table was previously created.
	// It will not drop or rename existing columns (when struct has changed).
	db.RegisterModel(new(AbuseReportShadowCopy))
}

func CreateShadowCopyForUser(ctx context.Context, userID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeUser, userID, content)
}

func CreateShadowCopyForRepository(ctx context.Context, repoID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeRepository, repoID, content)
}

func CreateShadowCopyForIssue(ctx context.Context, issueID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeIssue, issueID, content)
}

func CreateShadowCopyForComment(ctx context.Context, commentID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeComment, commentID, content)
}

func createShadowCopy(ctx context.Context, contentType ReportedContentType, contentID int64, content string) error {
	err := db.WithTx(ctx, func(ctx context.Context) error {
		sess := db.GetEngine(ctx)

		shadowCopy := &AbuseReportShadowCopy{RawValue: content}
		affected, err := sess.Insert(shadowCopy)
		if err != nil {
			return err
		} else if affected == 0 {
			log.Warn("Something went wrong while trying to create the shadow copy for reported content with type %d and ID %d.", contentType, contentID)
		}

		_, err = sess.Where(builder.Eq{
			"content_type": contentType,
			"content_id":   contentID,
		}).And(builder.IsNull{"shadow_copy_id"}).Update(&AbuseReport{ShadowCopyID: shadowCopy.NullableID()})
		if err != nil {
			return fmt.Errorf("could not link the shadow copy (%d) to reported content with type %d and ID %d - %w", shadowCopy.ID, contentType, contentID, err)
		}

		return nil
	})

	return err
}
