import {GET} from '../modules/fetch.js';
import {parseIssueHref, parseRepoOwnerPathInfo} from '../utils.js';

export function getIssueIcon(issue) {
  if (issue.is_pr) {
    if (issue.state === 'open') {
      if (issue.draft === true) {
        return 'octicon-git-pull-request-draft'; // WIP PR
      }
      return 'octicon-git-pull-request'; // Open PR
    } else if (issue.merged === true) {
      return 'octicon-git-merge'; // Merged PR
    }
    return 'octicon-git-pull-request'; // Closed PR
  } else if (issue.state === 'open') {
    return 'octicon-issue-opened'; // Open Issue
  }
  return 'octicon-issue-closed'; // Closed Issue
}

export function getIssueColor(issue) {
  if (issue.is_pr) {
    if (issue.draft === true) {
      return 'grey'; // WIP PR
    } else if (issue.merged === true) {
      return 'purple'; // Merged PR
    }
  }
  if (issue.state === 'open') {
    return 'green'; // Open Issue
  }
  return 'red'; // Closed Issue
}

export function isIssueSuggestionsLoaded() {
  return Boolean(window.config.issueValues);
}

export async function fetchIssueSuggestions() {
  const issuePathInfo = parseIssueHref(window.location.href);
  if (!issuePathInfo.ownerName) {
    const repoOwnerPathInfo = parseRepoOwnerPathInfo(window.location.pathname);
    issuePathInfo.ownerName = repoOwnerPathInfo.ownerName;
    issuePathInfo.repoName = repoOwnerPathInfo.repoName;
    // then no issuePathInfo.indexString here, it is only used to exclude the current issue when "matchIssue"
  }

  const res = await GET(`${window.config.appSubUrl}/${issuePathInfo.ownerName}/${issuePathInfo.repoName}/issues/suggestions`);
  const issues = await res.json();
  window.config.issueValues = issues;
}
