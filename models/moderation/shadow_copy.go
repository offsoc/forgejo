// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/builder"
)

type AbuseReportShadowCopy struct {
	ID          int64              `xorm:"pk autoincr"`
	RawValue    string             `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL"`
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

func CreateShadowCopyForRepository(ctx context.Context, commentID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeRepository, commentID, content)
}

func CreateShadowCopyForIssue(ctx context.Context, commentID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeIssue, commentID, content)
}

func CreateShadowCopyForComment(ctx context.Context, commentID int64, content string) error {
	return createShadowCopy(ctx, ReportedContentTypeComment, commentID, content)
}

func createShadowCopy(ctx context.Context, contentType ReportedContentType, contentID int64, content string) error {
	err := db.WithTx(ctx, func(ctx context.Context) error {
		sess := db.GetEngine(ctx)

		notLinkedFound, err := sess.Cols("id").Where(builder.IsNull{"shadow_copy_id"}).Exist(
			&AbuseReport{ContentType: contentType, ContentID: contentID},
		)
		if err != nil {
			return err
		} else if !notLinkedFound {
			log.Warn("Requested to create a shadow copy for reported content with type %d and ID %d but there is no such report without a shadow copy ID.", contentType, contentID)
			return nil
		}

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
			// TODO: What should happen if an item is updated multiple times (and the reports already have a shadow copy ID)?
		}).And(builder.IsNull{"shadow_copy_id"}).Cols("shadow_copy_id").Update(&AbuseReport{ShadowCopyID: &shadowCopy.ID})
		if err != nil {
			return fmt.Errorf("Could not link the shadow copy (%d) to reported content with type %d and ID %d. %w", shadowCopy.ID, contentType, contentID, err)
		}

		return nil
	})

	return err
}
