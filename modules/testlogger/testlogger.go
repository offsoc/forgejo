// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testlogger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/util"
)

var (
	prefix    string
	SlowTest  = 10 * time.Second
	SlowFlush = 5 * time.Second
)

var WriterCloser = &testLoggerWriterCloser{}

type testLoggerWriterCloser struct {
	sync.RWMutex
	t    []testing.TB
	errs []error // stack of error, parallel to the stack of testing.TB
	err  error   // fallback if the stack is empty
}

func (w *testLoggerWriterCloser) pushT(t testing.TB) {
	w.Lock()
	w.t = append(w.t, t)
	w.errs = append(w.errs, nil)
	w.Unlock()
}

func (w *testLoggerWriterCloser) Log(level log.Level, msg string) {
	msg = strings.TrimSpace(msg)

	w.printMsg(msg)
	if level >= log.ERROR {
		w.recordError(msg)
	}
}

// list of error message which will not fail the test
// ideally this list should be empty, however ensuring that it does not grow
// is already a good first step.
var ignoredErrorMessage = []string{
	// only seen on mysql tests https://codeberg.org/forgejo/forgejo/pulls/2657#issuecomment-1693055
	`table columns using inconsistent collation, they should use "utf8mb4_0900_ai_ci". Please go to admin panel Self Check page`,

	// TestPullWIPConvertSidebar
	`:PullRequestPushCommits() [E] comment.LoadIssue: issue does not exist [id:`,

	// TestAPIDeleteReleaseByTagName
	// action notification were a commit cannot be computed (because the commit got deleted)
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/release-tag, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/release-tag, rel_path: ]`,

	// TestAPIRepoTags
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/gitea/22, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/gitea/22, rel_path: ]`,

	// TestAPIDeleteTagByName
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/delete-tag, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/delete-tag, rel_path: ]`,

	// TestAPIGenerateRepo
	`Notify() [E] an error occurred while executing the CreateRepository actions method: gitRepo.GetCommit: object does not exist [id: , rel_path: ]`,

	// TestAPIPullUpdateByRebase
	`:testPR() [E] Unable to GetPullRequestByID[`,
	`:PullRequestSynchronized() [E] LoadAttributes: getRepositoryByID `,
	`:PullRequestSynchronized() [E] pr.Issue.LoadRepo: getRepositoryByID [`,
	`:handler() [E] Was unable to create issue notification: issue does not exist [`,
	`:func1() [E] PullRequestList.LoadAttributes: issues and prs may be not in sync: cannot find issue`,
	`:func1() [E] checkForInvalidation: GetRepositoryByIDCtx: repository does not exist `,

	// TestAPIPullReview
	`PullRequestReview() [E] Unsupported review webhook type`,

	// TestAPIPullReviewRequest
	`ToAPIPullRequest() [E] unable to resolve PR head ref`,

	// TestAPILFSUpload
	`Put() [E] Whilst putting LFS OID[ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb]: Failed to copy to tmpPath: ca/97/8112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb Error: content size does not match`,
	`[E] Error putting LFS MetaObject [ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb] into content store. Error: content size does not match`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb]. Error: content size does not match`,
	`Put() [E] Whilst putting LFS OID[2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a]: Failed to copy to tmpPath: 25/81/dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a Error: content hash does not match OID`,
	`[E] Error putting LFS MetaObject [2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a] into content store. Error: content hash does not match OID`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a]. Error: content hash does not match OID`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [83de2e488b89a0aa1c97496b888120a28b0c1e15463a4adb8405578c540f36d4]. Error: content size does not match`,

	// TestAPILFSVerify
	`getAuthenticatedMeta() [E] Unable to get LFS OID[fb8f7d8435968c4f82a726a92395be4d16f2f63116caf36c8ad35c60831ab042] Error: LFS Meta object does not exist`,

	// TestAPIUpdateOrgAvatar
	`UpdateAvatar() [E] UploadAvatar: image.DecodeConfig: image: unknown format`,

	// TestGetAttachment
	`/data/attachments/a/0/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a18: no such file or directory`,

	// TestBlockUser
	`BlockedUsersUnblock() [E] IsOrganization: org3 is an organization not a user`,
	`BlockedUsersBlock() [E] IsOrganization: org3 is an organization not a user`,
	`Action() [E] Cannot perform this action on an organization "unblock"`,
	`Action() [E] Cannot perform this action on an organization "block"`,

	// TestBlockActions
	`/gitea-repositories/user10/repo7.git Error: no such file or directory`,

	// TestRebuildCargo
	`RebuildCargoIndex() [E] RebuildIndex failed: GetRepositoryByOwnerAndName: repository does not exist [id: 0, uid: 0, owner_name: user2, name: _cargo-index]`,

	// TestDangerZoneConfirmation/Convert_fork/Fail
	`/gitea-repositories/user20/big_test_public_fork_7.git Error: no such file or directory`,
	// TestGitSmartHTTP
	`:sendFile() [E] request file path contains invalid path: objects/info/..\..\..\..\custom\conf\app.ini`,
	// TestGit/HTTP/BranchProtectMerge
	`:SSHLog() [E] ssh: Not allowed to push to protected branch protected. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestGit/HTTP/BranchProtectMerge
	`:SSHLog() [E] ssh: Not allowed to push to protected branch protected. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestGit/HTTP/BranchProtectMerge
	`:SSHLog() [E] ssh: branch protected is protected from force push. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestGit/HTTP/BranchProtect
	`:SSHLog() [E] ssh: branch before-create-2 is protected from changing file protected-file-data-`,
	// TestGit/HTTP/MergeFork/CreatePRAndMerge
	`:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 1099 name: user2:master]`,                          // sqlite
	"s/web/repo/branch.go:108:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 10000 name: user2:master]", // mysql
	"s/web/repo/branch.go:108:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 1060 name: user2:master]",  // pgsql
	// TestGit/HTTP/BranchProtectMerge
	`:func1() [E] PushToBaseRepo: PushRejected Error: exit status 1 - remote: error: cannot lock ref`,
	// TestGit/SSH/BranchProtectMerge
	`:func1() [E] PushToBaseRepo: PushRejected Error: exit status 1 - remote: error: cannot lock ref`,
	// TestGit/SSH/LFS/PushCommit/Little
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/PushCommit/Little
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/PushCommit/Big
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/PushCommit/Big
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/Locks
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/Locks
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/Locks
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/Locks
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/LFS/Locks
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/NoParams
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/NoParams
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/TitleOverride
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/TitleOverride
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/DescriptionOverride
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/PushParams/DescriptionOverride
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push/Fails
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push/Fails
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push/Succeeds
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push/Succeeds
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Force_push
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Branch_already_contains_commit
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull/Branch_already_contains_commit
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/CreateAgitFlowPull
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Not allowed to push to protected branch protected. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: branch protected is protected from force push. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/BranchProtectMerge
	`:SSHLog() [E] ssh: Unknown git command. Unknown git command git-lfs-transfer`,
	// TestGit/SSH/MergeFork/CreatePRAndMerge
	`:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 1102 name: user2:master]`,                          // sqlite
	"s/web/repo/branch.go:108:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 10003 name: user2:master]", // mysql
	"s/web/repo/branch.go:108:DeleteBranchPost() [E] DeleteBranch: GetBranch: branch does not exist [repo_id: 1063 name: user2:master]",  // pgsql
	// TestGit/SSH/PushCreate
	`:SSHLog() [E] ssh: Push to create is not enabled for users. ServCommand failed: internal API error response, status=403`,
	// TestGit/SSH/PushCreate
	`:SSHLog() [E] ssh: Cannot find repository: user2/repo-tmp-push-create-ssh. ServCommand failed: internal API error response, status=404`,
	// TestGit/SSH/PushCreate
	`:SSHLog() [E] ssh: Invalid repo name. Invalid repo name: invalid/repo-tmp-push-create-ssh`,
	// TestIssueReaction
	`:ChangeIssueReaction() [E] ChangeIssueReaction: '8ball' is not an allowed reaction`,
	// TestIssuePinMove
	`:IssuePinMove() [E] Issue does not belong to this repository`,
	// TestLinksLogin
	`:GetIssuesAllCommitStatus() [E] getAllCommitStatus: can't get commit statuses of pull [6]: object does not exist [id: refs/pull/2/head, rel_path: ]`,
	// TestLinksLogin
	`:GetIssuesAllCommitStatus() [E] getAllCommitStatus: can't get commit statuses of pull [6]: object does not exist [id: refs/pull/2/head, rel_path: ]`,
	// TestLinksLogin
	`:GetIssuesAllCommitStatus() [E] getAllCommitStatus: can't get commit statuses of pull [6]: object does not exist [id: refs/pull/2/head, rel_path: ]`,
	// TestLinksLogin
	`:GetIssuesAllCommitStatus() [E] Cannot open git repository <Repository 23:org17/big_test_public_4> for issue #1[20]. Error: no such file or directory`,
	// TestMigrate
	`] for OwnerID[2] failed: error while listing issues: token does not have at least one of required scope(s): [read:issue]`,
	// TestMigrate
	`:handler() [E] Run task failed: error while listing issues: token does not have at least one of required scope(s): [read:issue]`,
	// TestMigrate
	`] for OwnerID[2] failed: error while listing issues: token does not have at least one of required scope(s): [read:issue]`,
	// TestMigrate
	`:handler() [E] Run task failed: error while listing issues: token does not have at least one of required scope(s): [read:issue]`,
	// TestMirrorPush
	`:GetInfoRefs() [E] fork/exec /usr/bin/git: no such file or directory -`,

	// TestOrgMembers
	`:loadOrganizationOwners() [E] Organization does not have owner team: 25`,
	// TestOrgMembers
	`:loadOrganizationOwners() [E] Organization does not have owner team: 25`,
	// TestOrgMembers
	`:loadOrganizationOwners() [E] Organization does not have owner team: 25`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:SyncRepoBranches() [E] OpenRepository[user30/repo50]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:handlerBranchSync() [E] syncRepoBranches [50] failed: no such file or directory`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:SyncRepoBranches() [E] OpenRepository[user30/repo51]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:handlerBranchSync() [E] syncRepoBranches [51] failed: no such file or directory`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:SyncRepoBranches() [E] OpenRepository[user2/scoped_label]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestRecentlyPushed/unrelated_branches_are_not_shown
	`:handlerBranchSync() [E] syncRepoBranches [55] failed: no such file or directory`,
	// TestCantMergeConflict
	"]user1/repo1#1[base...conflict]> Unable to merge tracking into base: Merge Conflict Error: exit status 1: \nAuto-merging README.md\nCONFLICT (content): Merge conflict in README.md\nAutomatic merge failed; fix conflicts and then commit the result.",

	// TestCantMergeUnrelated
	`]user1/repo1#1[base...unrelated]> Unable to merge tracking into base: Merge UnrelatedHistories Error: exit status 128: fatal: refusing to merge unrelated histories`,
	// TestCantFastForwardOnlyMergeDiverging
	"]user1/repo1#1[master...diverging]> Unable to merge tracking into base: Merge DivergingFastForwardOnly Error: exit status 128: hint: Diverging branches can't be fast-forwarded, you need to either:\nhint: \nhint: \tgit merge --no-ff\nhint: \nhint: or:\nhint: \nhint: \tgit rebase\nhint: \nhint: Disable this message with \"git config advice.diverging false\"\nfatal: Not possible to fast-forward, aborting.",
	// TestPullrequestReopen/Base_branch_deleted
	`fatal: couldn't find remote ref base-branch`,
	// TestPullrequestReopen/Head_branch_deleted
	`]user2/reopen-base#1[base-branch...org26/reopen-head:head-branch]>]: branch does not exist [repo_id: 0 name: head-branch]`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user30/repo50]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [50] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user30/repo51]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [51] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user2/scoped_label]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [55] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	`:LoadBranches() [E] loadOneBranch() on repo #1, branch 'will-be-missing' failed: CountDivergingCommits: exit status 128 - fatal: bad revision 'master...refs/heads/will-be-missing'
		- fatal: bad revision 'master...refs/heads/will-be-missing'`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user30/repo50]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [50] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user30/repo51]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [51] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	`:SyncRepoBranches() [E] OpenRepository[user2/scoped_label]: %!w(*errors.errorString=&{no such file or directory})`,
	// TestDatabaseMissingABranch
	`:handlerBranchSync() [E] syncRepoBranches [55] failed: no such file or directory`,
	// TestDatabaseMissingABranch
	"LoadBranches() [E] loadOneBranch() on repo #1, branch 'will-be-missing' failed: CountDivergingCommits: exit status 128 - fatal: bad revision 'master...refs/heads/will-be-missing'\n - fatal: bad revision 'master...refs/heads/will-be-missing'",

	// TestCreateNewTagProtected/Git
	`:SSHLog() [E] ssh: Tag v-2 is protected. HookPreReceive(last) failed: internal API error response, status=403`,
	// TestMarkDownReadmeImage
	`:checkOutdatedBranch() [E] GetBranch: branch does not exist [repo_id: 1 name: home-md-img-check]`,
	// TestMarkDownReadmeImage
	`:checkOutdatedBranch() [E] GetBranch: branch does not exist [repo_id: 1 name: home-md-img-check]`,
	// TestMarkDownReadmeImageSubfolder
	`:checkOutdatedBranch() [E] GetBranch: branch does not exist [repo_id: 1 name: sub-home-md-img-check]`,
	// TestMarkDownReadmeImageSubfolder
	`:checkOutdatedBranch() [E] GetBranch: branch does not exist [repo_id: 1 name: sub-home-md-img-check]`,

	// TestKeyOnlyOneType
	`:ssh-key-test-repo-push is not authorized to write to user2/ssh-key-test-repo. ServCommand failed: internal API error response, status=401`,

	// TestPullRebase
	"/gitea-repositories/user2/repo1.git' does not appear to be a git repository\nfatal: Could not read from remote repository.\n\nPlease make sure you have the correct access rights\nand the repository exists.",

	// TestPullRebaseMerge
	"]user2/repo1#3[master...branch2]>]: branch does not exist [repo_id: 0 name: branch2]",

	// TestAuthorizeNoClientID
	`TrString() [E] Missing translation "form.ResponseType"`,

	// TestWebhookForms
	`TrString() [E] Missing translation "form.AuthorizationHeader"`,
	`TrString() [E] Missing translation "form.Channel"`,
	`TrString() [E] Missing translation "form.ContentType"`,
	`TrString() [E] Missing translation "form.HTTPMethod"`,
	`TrString() [E] Missing translation "form.PayloadURL"`,

	// TestRenameInvalidUsername
	`TrString() [E] Missing translation "form.Name"`,

	// TestDatabaseCollation
	`[E] [Error SQL Query] INSERT INTO test_collation_tbl (txt) VALUES ('main') []`,

	// Test_CmdForgejo_Actions
	`DB: No dedicated replica host defined; falling back to primary DSN for replica connections`,

	// TestDevtestErrorpages
	`ErrorPage() [E] Example error: Example error`,
}

func (w *testLoggerWriterCloser) recordError(msg string) {
	for _, s := range ignoredErrorMessage {
		if strings.Contains(msg, s) {
			return
		}
	}

	w.Lock()
	defer w.Unlock()

	err := w.err
	if len(w.errs) > 0 {
		err = w.errs[len(w.errs)-1]
	}

	if len(w.t) > 0 {
		// format error message to easily add it to the ignore list
		msg = fmt.Sprintf("// %s\n\t%q,", w.t[len(w.t)-1].Name(), msg)
	}

	err = errors.Join(err, errors.New(msg))

	if len(w.errs) > 0 {
		w.errs[len(w.errs)-1] = err
	} else {
		w.err = err
	}
}

func (w *testLoggerWriterCloser) printMsg(msg string) {
	// There was a data race problem: the logger system could still try to output logs after the runner is finished.
	// So we must ensure that the "t" in stack is still valid.
	w.RLock()
	defer w.RUnlock()

	if len(w.t) > 0 {
		t := w.t[len(w.t)-1]
		t.Log(msg)
	} else {
		// if there is no running test, the log message should be outputted to console, to avoid losing important information.
		// the "???" prefix is used to match the "===" and "+++" in PrintCurrentTest
		fmt.Fprintln(os.Stdout, "??? [TestLogger]", msg)
	}
}

func (w *testLoggerWriterCloser) popT() error {
	w.Lock()
	defer w.Unlock()

	if len(w.t) > 0 {
		w.t = w.t[:len(w.t)-1]
		err := w.errs[len(w.errs)-1]
		w.errs = w.errs[:len(w.errs)-1]
		return err
	}
	return w.err
}

func (w *testLoggerWriterCloser) Reset() error {
	w.Lock()
	if len(w.t) > 0 {
		for _, t := range w.t {
			if t == nil {
				continue
			}
			_, _ = fmt.Fprintf(os.Stdout, "Unclosed logger writer in test: %s", t.Name())
			t.Errorf("Unclosed logger writer in test: %s", t.Name())
		}
		w.t = nil
		w.errs = nil
	}
	err := w.err
	w.err = nil
	w.Unlock()
	return err
}

// PrintCurrentTest prints the current test to os.Stdout
func PrintCurrentTest(t testing.TB, skip ...int) func() {
	t.Helper()
	start := time.Now()
	actualSkip := util.OptionalArg(skip) + 1
	_, filename, line, _ := runtime.Caller(actualSkip)

	if log.CanColorStdout {
		_, _ = fmt.Fprintf(os.Stdout, "=== %s (%s:%d)\n", fmt.Formatter(log.NewColoredValue(t.Name())), strings.TrimPrefix(filename, prefix), line)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "=== %s (%s:%d)\n", t.Name(), strings.TrimPrefix(filename, prefix), line)
	}
	WriterCloser.pushT(t)
	return func() {
		took := time.Since(start)
		if took > SlowTest {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s is a slow test (took %v)\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgYellow)), fmt.Formatter(log.NewColoredValue(took, log.Bold, log.FgYellow)))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s is a slow test (took %v)\n", t.Name(), took)
			}
		}
		timer := time.AfterFunc(SlowFlush, func() {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s ... still flushing after %v ...\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgRed)), SlowFlush)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s ... still flushing after %v ...\n", t.Name(), SlowFlush)
			}
		})
		if err := queue.GetManager().FlushAll(t.Context(), time.Minute); err != nil {
			t.Errorf("Flushing queues failed with error %v", err)
		}
		timer.Stop()
		flushTook := time.Since(start) - took
		if flushTook > SlowFlush {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s had a slow clean-up flush (took %v)\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgRed)), fmt.Formatter(log.NewColoredValue(flushTook, log.Bold, log.FgRed)))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s had a slow clean-up flush (took %v)\n", t.Name(), flushTook)
			}
		}

		if err := WriterCloser.popT(); err != nil {
			// disable test failure for now (too flacky)
			_, _ = fmt.Fprintf(os.Stdout, "testlogger.go:recordError() FATAL ERROR: log.Error has been called: %v\n", err)
			// t.Errorf("testlogger.go:recordError() FATAL ERROR: log.Error has been called: %v", err)
		}
	}
}

// Printf takes a format and args and prints the string to os.Stdout
func Printf(format string, args ...any) {
	if log.CanColorStdout {
		for i := 0; i < len(args); i++ {
			args[i] = log.NewColoredValue(args[i])
		}
	}
	_, _ = fmt.Fprintf(os.Stdout, "\t"+format, args...)
}

// NewTestLoggerWriter creates a TestLogEventWriter as a log.LoggerProvider
func NewTestLoggerWriter(name string, mode log.WriterMode) log.EventWriter {
	w := &TestLogEventWriter{}
	w.base = log.NewEventWriterBase(name, "test-log-writer", mode)
	w.writer = WriterCloser
	return w
}

// TestLogEventWriter is a logger which will write to the testing log
type TestLogEventWriter struct {
	base   *log.EventWriterBaseImpl
	writer *testLoggerWriterCloser
}

// Base implements log.EventWriter.
func (t *TestLogEventWriter) Base() *log.EventWriterBaseImpl {
	return t.base
}

// GetLevel implements log.EventWriter.
func (t *TestLogEventWriter) GetLevel() log.Level {
	return t.base.GetLevel()
}

// GetWriterName implements log.EventWriter.
func (t *TestLogEventWriter) GetWriterName() string {
	return t.base.GetWriterName()
}

// GetWriterType implements log.EventWriter.
func (t *TestLogEventWriter) GetWriterType() string {
	return t.base.GetWriterType()
}

// Run implements log.EventWriter.
func (t *TestLogEventWriter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-t.base.Queue:
			if !ok {
				return
			}

			var errorMsg string

			switch msg := event.Msg.(type) {
			case string:
				errorMsg = msg
			case []byte:
				errorMsg = string(msg)
			case io.WriterTo:
				var buf bytes.Buffer
				if _, err := msg.WriteTo(&buf); err != nil {
					panic(err)
				}
				errorMsg = buf.String()
			default:
				errorMsg = fmt.Sprint(msg)
			}
			t.writer.Log(event.Origin.Level, errorMsg)
		}
	}
}

func init() {
	const relFilePath = "modules/testlogger/testlogger.go"
	_, filename, _, _ := runtime.Caller(0)
	if !strings.HasSuffix(filename, relFilePath) {
		panic("source code file path doesn't match expected: " + relFilePath)
	}
	prefix = strings.TrimSuffix(filename, relFilePath)
}
