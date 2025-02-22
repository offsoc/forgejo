// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/builder"
)

// ReportStatusType defines the statuses a report (of abusive content) can have.
type ReportStatusType int //revive:disable-line:exported

const (
	// ReportStatusTypeOpen represents the status of open reports that were not yet handled in any way.
	ReportStatusTypeOpen ReportStatusType = iota + 1 // 1
	// ReportStatusTypeHandled represents the status of valid reports, that have been acted upon.
	ReportStatusTypeHandled // 2
	// ReportStatusTypeIgnored represents the status of ignored reports, that were closed without any action.
	ReportStatusTypeIgnored // 3
)

// AbuseCategoryType defines the categories in which a user can include the reported content.
type AbuseCategoryType int //revive:disable-line:exported

const (
	AbuseCategoryTypeSpam            AbuseCategoryType = iota + 1 // 1
	AbuseCategoryTypeMalware                                      // 2
	AbuseCategoryTypeIllegalContent                               // 3
	AbuseCategoryTypeOtherViolations                              // 4 (Other violations of platform rules)
)

// ReportedContentType defines the types of content that can be reported
// (i.e. user/organization profile, repository, issue/pull, comment).
type ReportedContentType int //revive:disable-line:exported

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

// AbuseReport represents a report of abusive content.
type AbuseReport struct {
	ID     int64            `xorm:"pk autoincr"`
	Status ReportStatusType `xorm:"NOT NULL DEFAULT 1"`
	// The ID of the user who submitted the report.
	ReporterID int64 `xorm:"NOT NULL"` // index ?!
	// Reported content type: user/organization profile, repository, issue/pull or comment.
	ContentType ReportedContentType `xorm:"INDEX NOT NULL"`
	// The ID of the reported item (based on ContentType: user, repository, issue or comment).
	ContentID int64 `xorm:"NOT NULL"`
	// The abuse category selected by the reporter.
	Category AbuseCategoryType `xorm:"INDEX NOT NULL"`
	// Remarks provided by the reporter.
	Remarks string // TODO: ReporterReparks or Reason
	// The ID of the corresponding shadow-copied content when exists; otherwise null.
	ShadowCopyID *int64             `xorm:"DEFAULT NULL"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created NOT NULL"`
}

func init() {
	// RegisterModel will create the table if does not already exist
	// or any missing columns if the table was previously created.
	// It will not drop or rename existing columns (when struct has changed).
	db.RegisterModel(new(AbuseReport))
}

// IsReported reports whether one or more reports were already submitted for contentType and contentID.
func IsReported(ctx context.Context, contentType ReportedContentType, contentID int64) bool {
	// TODO: only consider the reports with 'New' status (and adjust the function name)?!
	reported, _ := db.GetEngine(ctx).Exist(&AbuseReport{ContentType: contentType, ContentID: contentID})
	return reported
}

// IsReportedUserBy reports whether reportedUserID is already reported by doerID.
func IsReportedUserBy(ctx context.Context, doerID int64, reportedUserID int64) bool {
	return alreadyReportedBy(ctx, doerID, ReportedContentTypeUser, reportedUserID)
}

// alreadyReportedBy returns if doerID has already submitted a report for contentType and contentID.
func alreadyReportedBy(ctx context.Context, doerID int64, contentType ReportedContentType, contentID int64) bool {
	reported, _ := db.GetEngine(ctx).Exist(&AbuseReport{ReporterID: doerID, ContentType: contentType, ContentID: contentID})
	return reported
}

// ReportUser creates a new abuse report regarding the user with the provided reportedUserID.
func ReportUser(ctx context.Context, reporterID int64, reportedUserID int64, remarks string) error {
	if reporterID == reportedUserID {
		return nil
	}

	report := &AbuseReport{
		ReporterID:  reporterID,
		ContentType: ReportedContentTypeUser,
		ContentID:   reportedUserID,
		Remarks:     remarks,
	}

	return reportAbuse(ctx, report)
}

// ReportRepository creates a new abuse report regarding the repository with the provided ID.
func ReportRepository(ctx context.Context, reporterID int64, repositoryID int64, remarks string) error {
	report := &AbuseReport{
		ReporterID:  reporterID,
		ContentType: ReportedContentTypeRepository,
		ContentID:   repositoryID,
		Remarks:     remarks,
	}

	return reportAbuse(ctx, report)
}

// ReportIssue creates a new abuse report regarding the issue with the provided ID.
func ReportIssue(ctx context.Context, reporterID int64, issueID int64, remarks string) error {
	report := &AbuseReport{
		ReporterID:  reporterID,
		ContentType: ReportedContentTypeIssue,
		ContentID:   issueID,
		Remarks:     remarks,
	}

	return reportAbuse(ctx, report)
}

// ReportComment creates a new abuse report regarding the comment with the provided ID.
func ReportComment(ctx context.Context, reporterID int64, commentID int64, remarks string) error {
	report := &AbuseReport{
		ReporterID:  reporterID,
		ContentType: ReportedContentTypeComment,
		ContentID:   commentID,
		Remarks:     remarks,
	}

	return reportAbuse(ctx, report)
}

func reportAbuse(ctx context.Context, report *AbuseReport) error {
	if alreadyReportedBy(ctx, report.ReporterID, report.ContentType, report.ContentID) {
		log.Warn("Seems that user %d wanted to report again the content with type %d and ID %d; this request will be ignored.", report.ReporterID, report.ContentType, report.ContentID)
		return nil
	}

	report.Status = ReportStatusTypeOpen
	report.Category = AbuseCategoryTypeOtherViolations // TODO: replace with user's selection

	_, err := db.GetEngine(ctx).Insert(report)

	return err
}

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
