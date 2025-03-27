// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"
	"strconv"

	action_model "forgejo.org/models/actions"
	issue_model "forgejo.org/models/issues"
	package_model "forgejo.org/models/packages"
	quota_model "forgejo.org/models/quota"
	repo_model "forgejo.org/models/repo"
	api "forgejo.org/modules/structs"
)

func ToQuotaRuleInfo(rule quota_model.Rule, withName bool) api.QuotaRuleInfo {
	info := api.QuotaRuleInfo{
		Limit:    rule.Limit,
		Subjects: make([]string, len(rule.Subjects)),
	}
	for i := range len(rule.Subjects) {
		info.Subjects[i] = rule.Subjects[i].String()
	}

	if withName {
		info.Name = rule.Name
	}

	return info
}

func toQuotaInfoUsed(used *quota_model.Used) api.QuotaUsed {
	info := api.QuotaUsed{
		Size: api.QuotaUsedSize{
			Repos: api.QuotaUsedSizeRepos{
				Public:  used.Size.Repos.Public,
				Private: used.Size.Repos.Private,
			},
			Git: api.QuotaUsedSizeGit{
				LFS: used.Size.Git.LFS,
			},
			Assets: api.QuotaUsedSizeAssets{
				Attachments: api.QuotaUsedSizeAssetsAttachments{
					Issues:   used.Size.Assets.Attachments.Issues,
					Releases: used.Size.Assets.Attachments.Releases,
				},
				Artifacts: used.Size.Assets.Artifacts,
				Packages: api.QuotaUsedSizeAssetsPackages{
					All: used.Size.Assets.Packages.All,
				},
			},
		},
	}
	return info
}

func ToQuotaInfo(used *quota_model.Used, groups quota_model.GroupList, withNames bool) api.QuotaInfo {
	info := api.QuotaInfo{
		Used:   toQuotaInfoUsed(used),
		Groups: ToQuotaGroupList(groups, withNames),
	}

	return info
}

func ToQuotaGroup(group quota_model.Group, withNames bool) api.QuotaGroup {
	info := api.QuotaGroup{
		Rules: make([]api.QuotaRuleInfo, len(group.Rules)),
	}
	if withNames {
		info.Name = group.Name
	}
	for i := range len(group.Rules) {
		info.Rules[i] = ToQuotaRuleInfo(group.Rules[i], withNames)
	}

	return info
}

func ToQuotaGroupList(groups quota_model.GroupList, withNames bool) api.QuotaGroupList {
	list := make(api.QuotaGroupList, len(groups))

	for i := range len(groups) {
		list[i] = ToQuotaGroup(*groups[i], withNames)
	}

	return list
}

func ToQuotaUsedAttachmentList(ctx context.Context, attachments []*repo_model.Attachment) (*api.QuotaUsedAttachmentList, error) {
	getAttachmentContainer := func(a *repo_model.Attachment) (string, string, error) {
		if a.ReleaseID != 0 {
			release, err := repo_model.GetReleaseByID(ctx, a.ReleaseID)
			if err != nil {
				return "", "", err
			}
			if err = release.LoadAttributes(ctx); err != nil {
				return "", "", err
			}
			return release.APIURL(), release.HTMLURL(), nil
		}
		if a.CommentID != 0 {
			comment, err := issue_model.GetCommentByID(ctx, a.CommentID)
			if err != nil {
				return "", "", err
			}
			return comment.APIURL(ctx), comment.HTMLURL(ctx), nil
		}
		if a.IssueID != 0 {
			issue, err := issue_model.GetIssueByID(ctx, a.IssueID)
			if err != nil {
				return "", "", err
			}
			if err = issue.LoadRepo(ctx); err != nil {
				return "", "", err
			}
			return issue.APIURL(ctx), issue.HTMLURL(), nil
		}
		return "", "", nil
	}

	result := make(api.QuotaUsedAttachmentList, len(attachments))
	for i, a := range attachments {
		capiURL, chtmlURL, err := getAttachmentContainer(a)
		if err != nil {
			return nil, err
		}

		apiURL := capiURL + "/assets/" + strconv.FormatInt(a.ID, 10)
		result[i] = &api.QuotaUsedAttachment{
			Name:   a.Name,
			Size:   a.Size,
			APIURL: apiURL,
		}
		result[i].ContainedIn.APIURL = capiURL
		result[i].ContainedIn.HTMLURL = chtmlURL
	}

	return &result, nil
}

func ToQuotaUsedPackageList(ctx context.Context, packages []*package_model.PackageVersion) (*api.QuotaUsedPackageList, error) {
	result := make(api.QuotaUsedPackageList, len(packages))
	for i, pv := range packages {
		d, err := package_model.GetPackageDescriptor(ctx, pv)
		if err != nil {
			return nil, err
		}

		var size int64
		for _, file := range d.Files {
			size += file.Blob.Size
		}

		result[i] = &api.QuotaUsedPackage{
			Name:    d.Package.Name,
			Type:    d.Package.Type.Name(),
			Version: d.Version.Version,
			Size:    size,
			HTMLURL: d.VersionHTMLURL(),
		}
	}

	return &result, nil
}

func ToQuotaUsedArtifactList(ctx context.Context, artifacts []*action_model.ActionArtifact) (*api.QuotaUsedArtifactList, error) {
	result := make(api.QuotaUsedArtifactList, len(artifacts))
	for i, a := range artifacts {
		run, err := action_model.GetRunByID(ctx, a.RunID)
		if err != nil {
			return nil, err
		}

		result[i] = &api.QuotaUsedArtifact{
			Name:    a.ArtifactName,
			Size:    a.FileCompressedSize,
			HTMLURL: run.HTMLURL(),
		}
	}

	return &result, nil
}
