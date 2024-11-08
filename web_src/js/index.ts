// bootstrap module must be the first one to be imported, it handles webpack lazy-loading and global errors
import './bootstrap';

import {initRepoActivityTopAuthorsChart} from './components/RepoActivityTopAuthors.vue';
import {initScopedAccessTokenCategories} from './components/ScopedAccessTokenSelector.vue';
import {initDashboardRepoList} from './components/DashboardRepoList.vue';

import {initGlobalCopyToClipboardListener} from './features/clipboard';
import {initContextPopups} from './features/contextpopup';
import {initRepoGraphGit} from './features/repo-graph';
import {initHeatmap} from './features/heatmap';
import {initImageDiff} from './features/imagediff';
import {initRepoMigration} from './features/repo-migration';
import {initRepoProject} from './features/repo-projects';
import {initTableSort} from './features/tablesort';
import {initAutoFocusEnd} from './features/autofocus-end';
import {initAdminUserListSearchForm} from './features/admin/users';
import {initAdminConfigs} from './features/admin/config';
import {initMarkupAnchors} from './markup/anchors';
import {initNotificationCount, initNotificationsTable} from './features/notification';
import {initRepoIssueContentHistory} from './features/repo-issue-content';
import {initStopwatch} from './features/stopwatch';
import {initFindFileInRepo} from './features/repo-findfile';
import {initCommentContent, initMarkupContent} from './markup/content';
import {initPdfViewer} from './render/pdf';

import {initUserAuthOauth2} from './features/user-auth';
import {
  initRepoIssueDue,
  initRepoIssueReferenceRepositorySearch,
  initRepoIssueTimeTracking,
  initRepoIssueWipTitle,
  initRepoPullRequestAllowMaintainerEdit,
  initRepoPullRequestReview, initRepoIssueSidebarList, initArchivedLabelHandler,
} from './features/repo-issue';
import {initRepoEllipsisButton, initCommitStatuses} from './features/repo-commit';
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
} from './features/common-global';
import {initRepoTopicBar} from './features/repo-home';
import {initAdminEmails} from './features/admin/emails';
import {initAdminCommon} from './features/admin/common';
import {initRepoTemplateSearch} from './features/repo-template';
import {initRepoCodeView} from './features/repo-code';
import {initSshKeyFormParser} from './features/sshkey-helper';
import {initUserSettings} from './features/user-settings';
import {initRepoArchiveLinks} from './features/repo-common';
import {initRepoMigrationStatusChecker} from './features/repo-migrate';
import {
  initRepoSettingGitHook,
  initRepoSettingsCollaboration,
  initRepoSettingSearchTeamBox,
} from './features/repo-settings';
import {initRepoDiffView} from './features/repo-diff';
import {initOrgTeamSearchRepoBox} from './features/org-team';
import {initUserAuthWebAuthn, initUserAuthWebAuthnRegister} from './features/user-auth-webauthn';
import {initRepoRelease, initRepoReleaseNew} from './features/repo-release';
import {initRepoEditor} from './features/repo-editor';
import {initCompSearchUserBox} from './features/comp/SearchUserBox';
import {initInstall} from './features/install';
import {initCompWebHookEditor} from './features/comp/WebHookEditor';
import {initRepoBranchButton} from './features/repo-branch';
import {initCommonOrganization} from './features/common-organization';
import {initRepoWikiForm} from './features/repo-wiki';
import {initRepoCommentForm, initRepository} from './features/repo-legacy';
import {initCopyContent} from './features/copycontent';
import {initCaptcha} from './features/captcha';
import {initRepositoryActionView} from './components/RepoActionView.vue';
import {initGlobalTooltips} from './modules/tippy';
import {initGiteaFomantic} from './modules/fomantic';
import {onDomReady} from './utils/dom';
import {initRepoIssueList} from './features/repo-issue-list';
import {initCommonIssueListQuickGoto} from './features/common-issue-list';
import {initRepoContributors} from './features/contributors';
import {initRepoCodeFrequency} from './features/code-frequency';
import {initRepoRecentCommits} from './features/recent-commits';
import {initRepoDiffCommitBranchesAndTags} from './features/repo-diff-commit';
import {initDirAuto} from './modules/dirauto';
import {initRepositorySearch} from './features/repo-search';
import {initColorPickers} from './features/colorpicker';
import {initRepoMilestoneEditor} from './features/repo-milestone';

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
