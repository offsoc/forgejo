// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

// ReportStatusType defines the statuses a report (of abusive content) can have.
type ReportStatusType int

const (
	// ReportStatusTypeOpen represents the status of open reports that were not yet handled in any way.
	ReportStatusTypeOpen ReportStatusType = iota + 1 // 1
	// ReportStatusTypeHandled represents the status of valid reports, that have been acted upon.
	ReportStatusTypeHandled // 2
	// ReportStatusTypeIgnored represents the status of ignored reports, that were closed without any action.
	ReportStatusTypeIgnored // 3
)

type (
	// AbuseCategoryType defines the categories in which a user can include the reported content.
	AbuseCategoryType int

	// AbuseCategoryItem defines a pair of value and it's corresponding translation key
	// (used to add options within the dropdown shown when new reports are submitted).
	AbuseCategoryItem struct {
		Value          AbuseCategoryType
		TranslationKey string
	}
)

const (
	AbuseCategoryTypeOther          AbuseCategoryType = iota + 1 // 1 (Other violations of platform rules)
	AbuseCategoryTypeSpam                                        // 2
	AbuseCategoryTypeMalware                                     // 3
	AbuseCategoryTypeIllegalContent                              // 4
)

// GetAbuseCategoriesList returns a list of pairs with the available abuse category types
// and their corresponding translation keys
func GetAbuseCategoriesList() []AbuseCategoryItem {
	return []AbuseCategoryItem{
		{AbuseCategoryTypeSpam, "moderation.abuse_category.spam"},
		{AbuseCategoryTypeMalware, "moderation.abuse_category.malware"},
		{AbuseCategoryTypeIllegalContent, "moderation.abuse_category.illegal_content"},
		{AbuseCategoryTypeOther, "moderation.abuse_category.other_violations"},
	}
}

// ReportedContentType defines the types of content that can be reported
// (i.e. user/organization profile, repository, issue/pull, comment).
type ReportedContentType int

const (
	// ReportedContentTypeUser should be used when reporting abusive users or organizations.
	ReportedContentTypeUser ReportedContentType = iota + 1 // 1

	// ReportedContentTypeRepository should be used when reporting a repository with abusive content.
	ReportedContentTypeRepository // 2

	// ReportedContentTypeIssue should be used when reporting an issue or pull request with abusive content.
	ReportedContentTypeIssue // 3

	// ReportedContentTypeComment should be used when reporting a comment with abusive content.
	ReportedContentTypeComment // 4
)

var allReportedContentTypes = []ReportedContentType{
	ReportedContentTypeUser,
	ReportedContentTypeRepository,
	ReportedContentTypeIssue,
	ReportedContentTypeComment,
}

func (t ReportedContentType) IsValid() bool {
	return slices.Contains(allReportedContentTypes, t)
}

// AbuseReport represents a report of abusive content.
type AbuseReport struct {
	ID     int64            `xorm:"pk autoincr"`
	Status ReportStatusType `xorm:"INDEX NOT NULL DEFAULT 1"`
	// The ID of the user who submitted the report.
	ReporterID int64 `xorm:"NOT NULL"`
	// Reported content type: user/organization profile, repository, issue/pull or comment.
	ContentType ReportedContentType `xorm:"INDEX NOT NULL"`
	// The ID of the reported item (based on ContentType: user, repository, issue or comment).
	ContentID int64 `xorm:"NOT NULL"`
	// The abuse category selected by the reporter.
	Category AbuseCategoryType `xorm:"INDEX NOT NULL"`
	// Remarks provided by the reporter.
	Remarks string
	// The ID of the corresponding shadow-copied content when exists; otherwise null.
	ShadowCopyID sql.NullInt64      `xorm:"DEFAULT NULL"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created NOT NULL"`
}

var ErrSelfReporting = errors.New("reporting yourself is not allowed")

func init() {
	// RegisterModel will create the table if does not already exist
	// or any missing columns if the table was previously created.
	// It will not drop or rename existing columns (when struct has changed).
	db.RegisterModel(new(AbuseReport))
}

// IsShadowCopyNeeded reports whether one or more reports were already submitted
// for contentType and contentID and not yet linked to a shadow copy (regardless their status).
func IsShadowCopyNeeded(ctx context.Context, contentType ReportedContentType, contentID int64) (bool, error) {
	return db.GetEngine(ctx).Cols("id").Where(builder.IsNull{"shadow_copy_id"}).Exist(
		&AbuseReport{ContentType: contentType, ContentID: contentID},
	)
}

// AlreadyReportedByAndOpen returns if doerID has already submitted a report for contentType and contentID that is still Open.
func AlreadyReportedByAndOpen(ctx context.Context, doerID int64, contentType ReportedContentType, contentID int64) bool {
	reported, _ := db.GetEngine(ctx).Exist(&AbuseReport{
		Status:      ReportStatusTypeOpen,
		ReporterID:  doerID,
		ContentType: contentType,
		ContentID:   contentID,
	})
	return reported
}

// ReportAbuse creates a new abuse report in the DB with 'Open' status.
// If the reported content is the user profile of the reporter ErrSelfReporting is returned.
// If there is already an open report submitted by the same user for the same content,
// the request will be ignored without returning an error (and a warning will be logged).
func ReportAbuse(ctx context.Context, report *AbuseReport) error {
	if report.ContentType == ReportedContentTypeUser && report.ReporterID == report.ContentID {
		return ErrSelfReporting
	}

	if AlreadyReportedByAndOpen(ctx, report.ReporterID, report.ContentType, report.ContentID) {
		log.Warn("Seems that user %d wanted to report again the content with type %d and ID %d; this request will be ignored.", report.ReporterID, report.ContentType, report.ContentID)
		return nil
	}

	report.Status = ReportStatusTypeOpen
	_, err := db.GetEngine(ctx).Insert(report)

	return err
}

/*
// MarkAsHandled will change the status to 'Handled' for all reports linked to the same item (user, repository, issue or comment).
func MarkAsHandled(ctx context.Context, contentType ReportedContentType, contentID int64) error {
	return updateStatus(ctx, contentType, contentID, ReportStatusTypeHandled)
}

// MarkAsIgnored will change the status to 'Ignored' for all reports linked to the same item (user, repository, issue or comment).
func MarkAsIgnored(ctx context.Context, contentType ReportedContentType, contentID int64) error {
	return updateStatus(ctx, contentType, contentID, ReportStatusTypeIgnored)
}

// updateStatus will set the provided status for any reports linked to the item with the given type and ID.
func updateStatus(ctx context.Context, contentType ReportedContentType, contentID int64, status ReportStatusType) error {
	_, err := db.GetEngine(ctx).Where(builder.Eq{
		"content_type": contentType,
		"content_id":   contentID,
	}).Cols("status").Update(&AbuseReport{Status: status})

	return err
}
*/
