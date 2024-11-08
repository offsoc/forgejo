// bootstrap module must be the first one to be imported, it handles webpack lazy-loading and global errors
import './bootstrap.ts';

import {initRepoActivityTopAuthorsChart} from './components/RepoActivityTopAuthors.vue';
import {initScopedAccessTokenCategories} from './components/ScopedAccessTokenSelector.vue';
import {initDashboardRepoList} from './components/DashboardRepoList.vue';

import {initGlobalCopyToClipboardListener} from './features/clipboard.ts';
import {initContextPopups} from './features/contextpopup.ts';
import {initRepoGraphGit} from './features/repo-graph.ts';
import {initHeatmap} from './features/heatmap.ts';
import {initImageDiff} from './features/imagediff.ts';
import {initRepoMigration} from './features/repo-migration.ts';
import {initRepoProject} from './features/repo-projects.ts';
import {initTableSort} from './features/tablesort.ts';
import {initAutoFocusEnd} from './features/autofocus-end.ts';
import {initAdminUserListSearchForm} from './features/admin/users.ts';
import {initAdminConfigs} from './features/admin/config.ts';
import {initMarkupAnchors} from './markup/anchors.ts';
import {initNotificationCount, initNotificationsTable} from './features/notification.ts';
import {initRepoIssueContentHistory} from './features/repo-issue-content.ts';
import {initStopwatch} from './features/stopwatch.ts';
import {initFindFileInRepo} from './features/repo-findfile.ts';
import {initCommentContent, initMarkupContent} from './markup/content.ts';
import {initPdfViewer} from './render/pdf.ts';

import {initUserAuthOauth2} from './features/user-auth.ts';
import {
  initRepoIssueDue,
  initRepoIssueReferenceRepositorySearch,
  initRepoIssueTimeTracking,
  initRepoIssueWipTitle,
  initRepoPullRequestAllowMaintainerEdit,
  initRepoPullRequestReview, initRepoIssueSidebarList, initArchivedLabelHandler,
} from './features/repo-issue.ts';
import {initRepoEllipsisButton, initCommitStatuses, initCommitNotes} from './features/repo-commit.ts';
import {
  initFootLanguageMenu,
  initGlobalButtonClickOnEnter,
  initGlobalButtons,
  initGlobalCommon,
  initGlobalDropzone,
  initGlobalEnterQuickSubmit,
  initGlobalFormDirtyLeaveConfirm,
  initGlobalLinkActions,
  initHeadNavbarContentToggle,
} from './features/common-global.ts';
import {initRepoTopicBar} from './features/repo-home.ts';
import {initAdminEmails} from './features/admin/emails.ts';
import {initAdminCommon} from './features/admin/common.ts';
import {initRepoTemplateSearch} from './features/repo-template.ts';
import {initRepoCodeView} from './features/repo-code.ts';
import {initSshKeyFormParser} from './features/sshkey-helper.ts';
import {initUserSettings} from './features/user-settings.ts';
import {initRepoArchiveLinks} from './features/repo-common.ts';
import {initRepoMigrationStatusChecker} from './features/repo-migrate.ts';
import {
  initRepoSettingGitHook,
  initRepoSettingsCollaboration,
  initRepoSettingSearchTeamBox,
} from './features/repo-settings.ts';
import {initRepoDiffView} from './features/repo-diff.ts';
import {initOrgTeamSearchRepoBox} from './features/org-team.ts';
import {initUserAuthWebAuthn, initUserAuthWebAuthnRegister} from './features/user-auth-webauthn.ts';
import {initRepoRelease, initRepoReleaseNew} from './features/repo-release.ts';
import {initRepoEditor} from './features/repo-editor.ts';
import {initCompSearchUserBox} from './features/comp/SearchUserBox.ts';
import {initInstall} from './features/install.ts';
import {initCompWebHookEditor} from './features/comp/WebHookEditor.ts';
import {initRepoBranchButton} from './features/repo-branch.ts';
import {initCommonOrganization} from './features/common-organization.ts';
import {initRepoWikiForm} from './features/repo-wiki.ts';
import {initRepoCommentForm, initRepository} from './features/repo-legacy.ts';
import {initCopyContent} from './features/copycontent.ts';
import {initCaptcha} from './features/captcha.ts';
import {initRepositoryActionView} from './components/RepoActionView.vue';
import {initGlobalTooltips} from './modules/tippy.ts';
import {initGiteaFomantic} from './modules/fomantic.ts';
import {onDomReady} from './utils/dom.ts';
import {initRepoIssueList} from './features/repo-issue-list.ts';
import {initCommonIssueListQuickGoto} from './features/common-issue-list.ts';
import {initRepoContributors} from './features/contributors.ts';
import {initRepoCodeFrequency} from './features/code-frequency.ts';
import {initRepoRecentCommits} from './features/recent-commits.ts';
import {initRepoDiffCommitBranchesAndTags} from './features/repo-diff-commit.ts';
import {initDirAuto} from './modules/dirauto.ts';
import {initRepositorySearch} from './features/repo-search.ts';
import {initColorPickers} from './features/colorpicker.ts';
import {initRepoMilestoneEditor} from './features/repo-milestone.ts';

// Init Gitea's Fomantic settings
initGiteaFomantic();
initDirAuto();

onDomReady(() => {
  initGlobalCommon();

  initGlobalTooltips();
  initGlobalButtonClickOnEnter();
  initGlobalButtons();
  initGlobalCopyToClipboardListener();
  initGlobalDropzone();
  initGlobalEnterQuickSubmit();
  initGlobalFormDirtyLeaveConfirm();
  initGlobalLinkActions();

  initCommonOrganization();
  initCommonIssueListQuickGoto();

  initCompSearchUserBox();
  initCompWebHookEditor();

  initInstall();

  initHeadNavbarContentToggle();
  initFootLanguageMenu();

  initCommentContent();
  initContextPopups();
  initHeatmap();
  initImageDiff();
  initMarkupAnchors();
  initMarkupContent();
  initSshKeyFormParser();
  initStopwatch();
  initTableSort();
  initAutoFocusEnd();
  initFindFileInRepo();
  initCopyContent();

  initAdminCommon();
  initAdminEmails();
  initAdminUserListSearchForm();
  initAdminConfigs();

  initDashboardRepoList();

  initNotificationCount();
  initNotificationsTable();

  initOrgTeamSearchRepoBox();

  initRepoActivityTopAuthorsChart();
  initRepoArchiveLinks();
  initRepoBranchButton();
  initRepoCodeView();
  initRepoCommentForm();
  initRepoEllipsisButton();
  initRepoDiffCommitBranchesAndTags();
  initRepoEditor();
  initRepoGraphGit();
  initRepoIssueContentHistory();
  initRepoIssueDue();
  initRepoIssueList();
  initRepoIssueSidebarList();
  initArchivedLabelHandler();
  initRepoIssueReferenceRepositorySearch();
  initRepoIssueTimeTracking();
  initRepoIssueWipTitle();
  initRepoMigration();
  initRepoMigrationStatusChecker();
  initRepoProject();
  initRepoPullRequestAllowMaintainerEdit();
  initRepoPullRequestReview();
  initRepoRelease();
  initRepoReleaseNew();
  initRepoSettingGitHook();
  initRepoSettingSearchTeamBox();
  initRepoSettingsCollaboration();
  initRepoTemplateSearch();
  initRepoTopicBar();
  initRepoWikiForm();
  initRepository();
  initRepositoryActionView();
  initRepositorySearch();
  initRepoContributors();
  initRepoCodeFrequency();
  initRepoRecentCommits();
  initRepoMilestoneEditor();

  initCommitStatuses();
  initCommitNotes();
  initCaptcha();

  initUserAuthOauth2();
  initUserAuthWebAuthn();
  initUserAuthWebAuthnRegister();
  initUserSettings();
  initRepoDiffView();
  initPdfViewer();
  initScopedAccessTokenCategories();
  initColorPickers();
});
