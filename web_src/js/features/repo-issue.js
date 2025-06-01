import $ from 'jquery';
import {htmlEscape} from 'escape-goat';
import {showTemporaryTooltip, createTippy} from '../modules/tippy.js';
import {hideElem, showElem, toggleElem} from '../utils/dom.js';
import {setFileFolding} from './file-fold.js';
import {getComboMarkdownEditor, initComboMarkdownEditor} from './comp/ComboMarkdownEditor.js';
import {toAbsoluteUrl} from '../utils.js';
import {initDropzone} from './common-global.js';
import {POST, GET} from '../modules/fetch.js';
import {showErrorToast} from '../modules/toast.js';
import {emojiHTML} from './emoji.js';

const {appSubUrl} = window.config;

// if there are draft comments, confirm before reloading, to avoid losing comments
export function reloadConfirmDraftComment() {
  const commentTextareas = [
    document.querySelector('.edit-content-zone:not(.tw-hidden) textarea'),
    document.querySelector('#comment-form textarea'),
  ];
  for (const textarea of commentTextareas) {
    // Most users won't feel too sad if they lose a comment with 10 chars, they can re-type these in seconds.
    // But if they have typed more (like 50) chars and the comment is lost, they will be very unhappy.
    if (textarea && textarea.value.trim().length > 10) {
      textarea.parentElement.scrollIntoView();
      if (!window.confirm('Page will be reloaded, but there are draft comments. Continuing to reload will discard the comments. Continue?')) {
        return;
      }
      break;
    }
  }
  window.location.reload();
}

export function initRepoIssueTimeTracking() {
  $(document).on('click', '.issue-add-time', () => {
    $('.issue-start-time-modal').modal({
      duration: 200,
      onApprove() {
        document.getElementById('add_time_manual_form').requestSubmit();
      },
    }).modal('show');
    $('.issue-start-time-modal input').on('keydown', (e) => {
      if ((e.keyCode || e.key) === 13) {
        document.getElementById('add_time_manual_form').requestSubmit();
      }
    });
  });
  $(document).on('click', '.issue-start-time, .issue-stop-time', () => {
    document.getElementById('toggle_stopwatch_form').requestSubmit();
  });
  $(document).on('click', '.issue-cancel-time', () => {
    document.getElementById('cancel_stopwatch_form').requestSubmit();
  });
  $(document).on('click', 'button.issue-delete-time', function () {
    const sel = `.issue-delete-time-modal[data-id="${$(this).data('id')}"]`;
    $(sel).modal({
      duration: 200,
      onApprove() {
        document.getElementById(`${sel} form`).requestSubmit();
      },
    }).modal('show');
  });
}

async function updateDeadline(deadlineString) {
  hideElem('#deadline-err-invalid-date');
  document.getElementById('deadline-loader')?.classList.add('is-loading');

  let realDeadline = null;
  if (deadlineString !== '') {
    const newDate = Date.parse(deadlineString);

    if (Number.isNaN(newDate)) {
      document.getElementById('deadline-loader')?.classList.remove('is-loading');
      showElem('#deadline-err-invalid-date');
      return false;
    }
    realDeadline = new Date(newDate);
  }

  try {
    const response = await POST(document.getElementById('update-issue-deadline-form').getAttribute('action'), {
      data: {due_date: realDeadline},
    });

    if (response.ok) {
      window.location.reload();
    } else {
      throw new Error('Invalid response');
    }
  } catch (error) {
    console.error(error);
    document.getElementById('deadline-loader').classList.remove('is-loading');
    showElem('#deadline-err-invalid-date');
  }
}

export function initRepoIssueDue() {
  $(document).on('click', '.issue-due-edit', () => {
    toggleElem('#deadlineForm');
  });
  $(document).on('click', '.issue-due-remove', () => {
    updateDeadline('');
  });
  $(document).on('submit', '.issue-due-form', () => {
    updateDeadline($('#deadlineDate').val());
    return false;
  });
}

/**
 * @param {HTMLElement} item
 */
function excludeLabel(item) {
  const href = item.getAttribute('href');
  const id = item.getAttribute('data-label-id');

  const regStr = `labels=((?:-?[0-9]+%2c)*)(${id})((?:%2c-?[0-9]+)*)&`;
  const newStr = 'labels=$1-$2$3&';

  window.location.assign(href.replace(new RegExp(regStr), newStr));
}

export function initRepoIssueSidebarList() {
  const repolink = $('#repolink').val();
  const repoId = $('#repoId').val();
  const crossRepoSearch = $('#crossRepoSearch').val() === 'true';
  const tp = $('#type').val();
  $('#new-dependency-drop-list')
    .dropdown({
      apiSettings: {
        beforeSend(settings) {
          if (!settings.urlData.query.trim()) {
            settings.url = `${appSubUrl}/${repolink}/issues/search?q={query}&type=${tp}&sort=updated`;
          } else if (crossRepoSearch) {
            settings.url = `${appSubUrl}/issues/search?q={query}&priority_repo_id=${repoId}&type=${tp}&sort=relevance`;
          } else {
            settings.url = `${appSubUrl}/${repolink}/issues/search?q={query}&type=${tp}&sort=relevance`;
          }
          return settings;
        },
        onResponse(response) {
          const filteredResponse = {success: true, results: []};
          const currIssueId = $('#new-dependency-drop-list').data('issue-id');
          // Parse the response from the api to work with our dropdown
          for (const [_, issue] of Object.entries(response)) {
            // Don't list current issue in the dependency list.
            if (issue.id === currIssueId) {
              continue;
            }
            filteredResponse.results.push({
              name: `#${issue.number} ${issueTitleHTML(htmlEscape(issue.title))
              }<div class="text small tw-break-anywhere">${htmlEscape(issue.repository.full_name)}</div>`,
              value: issue.id,
            });
          }
          return filteredResponse;
        },
        cache: false,
      },

      fullTextSearch: true,
    });

  $('.menu a.label-filter-item').each(function () {
    $(this).on('click', function (e) {
      if (e.altKey) {
        e.preventDefault();
        excludeLabel(this);
      }
    });
  });

  $('.menu .ui.dropdown.label-filter').on('keydown', (e) => {
    if (e.altKey && e.keyCode === 13) {
      const selectedItem = document.querySelector('.menu .ui.dropdown.label-filter .menu .item.selected');
      if (selectedItem) {
        excludeLabel(selectedItem);
      }
    }
  });
  $('.ui.dropdown.label-filter, .ui.dropdown.select-label').dropdown('setting', {'hideDividers': 'empty'}).dropdown('refreshItems');
}

export function initRepoIssueCommentDelete() {
  // Delete comment
  document.addEventListener('click', async (e) => {
    if (!e.target.matches('.delete-comment')) return;
    e.preventDefault();

    const deleteButton = e.target;
    if (window.confirm(deleteButton.getAttribute('data-locale'))) {
      try {
        const response = await POST(deleteButton.getAttribute('data-url'));
        if (!response.ok) throw new Error('Failed to delete comment');

        const conversationHolder = deleteButton.closest('.conversation-holder');
        const parentTimelineItem = deleteButton.closest('.timeline-item');
        const parentTimelineGroup = deleteButton.closest('.timeline-item-group');

        // Check if this was a pending comment.
        if (conversationHolder?.querySelector('.pending-label')) {
          const counter = document.querySelector('#review-box .review-comments-counter');
          let num = parseInt(counter?.getAttribute('data-pending-comment-number')) - 1 || 0;
          num = Math.max(num, 0);
          counter.setAttribute('data-pending-comment-number', num);
          counter.textContent = String(num);
        }

        document.getElementById(deleteButton.getAttribute('data-comment-id'))?.remove();

        if (conversationHolder && !conversationHolder.querySelector('.comment')) {
          const path = conversationHolder.getAttribute('data-path');
          const side = conversationHolder.getAttribute('data-side');
          const idx = conversationHolder.getAttribute('data-idx');
          const lineType = conversationHolder.closest('tr')?.getAttribute('data-line-type');

          // the conversation holder could appear either on the "Conversation" page, or the "Files Changed" page
          // on the Conversation page, there is no parent "tr", so no need to do anything for "add-code-comment"
          if (lineType) {
            if (lineType === 'same') {
              document.querySelector(`[data-path="${path}"] .add-code-comment[data-idx="${idx}"]`).classList.remove('tw-invisible');
            } else {
              document.querySelector(`[data-path="${path}"] .add-code-comment[data-side="${side}"][data-idx="${idx}"]`).classList.remove('tw-invisible');
            }
          }
          conversationHolder.remove();
        }

        // Check if there is no review content, move the time avatar upward to avoid overlapping the content below.
        if (!parentTimelineGroup?.querySelector('.timeline-item.comment') && !parentTimelineItem?.querySelector('.conversation-holder')) {
          const timelineAvatar = parentTimelineGroup?.querySelector('.timeline-avatar');
          timelineAvatar?.classList.remove('timeline-avatar-offset');
        }
      } catch (error) {
        console.error(error);
      }
    }
  });
}

export function initRepoIssueDependencyDelete() {
  // Delete Issue dependency
  $(document).on('click', '.delete-dependency-button', (e) => {
    const id = e.currentTarget.getAttribute('data-id');
    const type = e.currentTarget.getAttribute('data-type');

    $('.remove-dependency').modal({
      closable: false,
      duration: 200,
      onApprove: () => {
        $('#removeDependencyID').val(id);
        $('#dependencyType').val(type);
        document.getElementById('removeDependencyForm').requestSubmit();
      },
    }).modal('show');
  });
}

export function initRepoIssueCodeCommentCancel() {
  // Cancel inline code comment
  document.addEventListener('click', (e) => {
    if (!e.target.matches('.cancel-code-comment')) return;

    const form = e.target.closest('form');
    if (form?.classList.contains('comment-form')) {
      hideElem(form);
      showElem(form.closest('.comment-code-cloud')?.querySelectorAll('button.comment-form-reply'));
    } else {
      form.closest('.comment-code-cloud')?.remove();
    }
  });
}

export function initRepoPullRequestUpdate() {
  // Pull Request update button
  const pullUpdateButton = document.querySelector('.update-button > button');
  if (!pullUpdateButton) return;

  pullUpdateButton.addEventListener('click', async function (e) {
    e.preventDefault();
    const redirect = this.getAttribute('data-redirect');
    this.classList.add('is-loading');
    let response;
    try {
      response = await POST(this.getAttribute('data-do'));
    } catch (error) {
      console.error(error);
    } finally {
      this.classList.remove('is-loading');
    }
    let data;
    try {
      data = await response?.json(); // the response is probably not a JSON
    } catch (error) {
      console.error(error);
    }
    if (data?.redirect) {
      window.location.href = data.redirect;
    } else if (redirect) {
      window.location.href = redirect;
    } else {
      window.location.reload();
    }
  });

  $('.update-button > .dropdown').dropdown({
    onChange(_text, _value, $choice) {
      const url = $choice[0].getAttribute('data-do');
      if (url) {
        const buttonText = pullUpdateButton.querySelector('.button-text');
        if (buttonText) {
          buttonText.textContent = $choice.text();
        }
        pullUpdateButton.setAttribute('data-do', url);
      }
    },
  });
}

export function initRepoPullRequestAllowMaintainerEdit() {
  const wrapper = document.getElementById('allow-edits-from-maintainers');
  if (!wrapper) return;
  const checkbox = wrapper.querySelector('input[type="checkbox"]');
  checkbox.addEventListener('input', async () => {
    const url = `${wrapper.getAttribute('data-url')}/set_allow_maintainer_edit`;
    wrapper.classList.add('is-loading');
    try {
      const resp = await POST(url, {data: new URLSearchParams({allow_maintainer_edit: checkbox.checked})});
      if (!resp.ok) {
        throw new Error('Failed to update maintainer edit permission');
      }
      const data = await resp.json();
      checkbox.checked = data.allow_maintainer_edit;
    } catch (error) {
      checkbox.checked = !checkbox.checked;
      console.error(error);
      showTemporaryTooltip(wrapper, wrapper.getAttribute('data-prompt-error'));
    } finally {
      wrapper.classList.remove('is-loading');
    }
  });
}

export function initRepoIssueReferenceRepositorySearch() {
  $('.issue_reference_repository_search')
    .dropdown({
      apiSettings: {
        url: `${appSubUrl}/repo/search?q={query}&limit=20`,
        onResponse(response) {
          const filteredResponse = {success: true, results: []};
          for (const repo of response.data) {
            filteredResponse.results.push({
              name: htmlEscape(repo.repository.full_name),
              value: repo.repository.full_name,
            });
          }
          return filteredResponse;
        },
        cache: false,
      },
      onChange(_value, _text, $choice) {
        const $form = $choice.closest('form');
        if (!$form.length) return;

        $form[0].setAttribute('action', `${appSubUrl}/${_text}/issues/new`);
      },
      fullTextSearch: true,
    });
}

export function initRepoIssueWipTitle() {
  $('.title_wip_desc > a').on('click', (e) => {
    e.preventDefault();

    const issueTitleEl = document.getElementById('issue_title');
    issueTitleEl.focus();
    const value = issueTitleEl.value.trim().toUpperCase();

    const wipPrefixes = $('.title_wip_desc').data('wip-prefixes');
    for (const prefix of wipPrefixes) {
      if (value.startsWith(prefix.toUpperCase())) {
        return;
      }
    }

    issueTitleEl.value = `${wipPrefixes[0]} ${issueTitleEl.value}`;
  });
}

export async function updateIssuesMeta(url, action, issue_ids, id) {
  try {
    const response = await POST(url, {data: new URLSearchParams({action, issue_ids, id})});
    if (!response.ok) {
      throw new Error('Failed to update issues meta');
    }
  } catch (error) {
    console.error(error);
  }
}

export function initRepoIssueComments() {
  if (!$('.repository.view.issue .timeline').length) return;

  $('.re-request-review').on('click', async function (e) {
    e.preventDefault();
    const url = this.getAttribute('data-update-url');
    const issueId = this.getAttribute('data-issue-id');
    const id = this.getAttribute('data-id');
    const isChecked = this.classList.contains('checked');

    await updateIssuesMeta(url, isChecked ? 'detach' : 'attach', issueId, id);
    window.location.reload();
  });

  document.addEventListener('click', (e) => {
    const urlTarget = document.querySelector(':target');
    if (!urlTarget) return;

    const urlTargetId = urlTarget.id;
    if (!urlTargetId) return;

    if (!/^(issue|pull)(comment)?-\d+$/.test(urlTargetId)) return;

    if (!e.target.closest(`#${urlTargetId}`)) {
      const scrollPosition = $(window).scrollTop();
      window.location.hash = '';
      $(window).scrollTop(scrollPosition);
      window.history.pushState(null, null, ' ');
    }
  });
}

export async function handleReply($el) {
  hideElem($el);
  const $form = $el.closest('.comment-code-cloud').find('.comment-form');
  showElem($form);

  const $textarea = $form.find('textarea');
  let editor = getComboMarkdownEditor($textarea);
  if (!editor) {
    // FIXME: the initialization of the dropzone is not consistent.
    // When the page is loaded, the dropzone is initialized by initGlobalDropzone, but the editor is not initialized.
    // When the form is submitted and partially reload, none of them is initialized.
    const dropzone = $form.find('.dropzone')[0];
    if (!dropzone.dropzone) initDropzone(dropzone);
    editor = await initComboMarkdownEditor($form.find('.combo-markdown-editor'));
  }
  editor.focus();
  return editor;
}

export function initRepoPullRequestReview() {
  if (window.location.hash && window.location.hash.startsWith('#issuecomment-')) {
    // set scrollRestoration to 'manual' when there is a hash in url, so that the scroll position will not be remembered after refreshing
    if (window.history.scrollRestoration !== 'manual') {
      window.history.scrollRestoration = 'manual';
    }
    const commentDiv = document.querySelector(window.location.hash);
    if (commentDiv) {
      // get the name of the parent id
      const groupID = commentDiv.closest('div[id^="code-comments-"]')?.getAttribute('id');
      if (groupID && groupID.startsWith('code-comments-')) {
        const id = groupID.slice(14);
        const ancestorDiffBox = commentDiv.closest('.diff-file-box');
        // on pages like conversation, there is no diff header
        const diffHeader = ancestorDiffBox?.querySelector('.diff-file-header');

        // offset is for scrolling
        let offset = 30;
        if (diffHeader) {
          offset += $('.diff-detail-box').outerHeight() + $(diffHeader).outerHeight();
        }

        hideElem(`#show-outdated-${id}`);
        showElem(`#code-comments-${id}, #code-preview-${id}, #hide-outdated-${id}`);
        // if the comment box is folded, expand it
        if (ancestorDiffBox?.getAttribute('data-folded') === 'true') {
          setFileFolding(ancestorDiffBox, ancestorDiffBox.querySelector('.fold-file'), false);
        }

        window.scrollTo({
          top: $(commentDiv).offset().top - offset,
          behavior: 'instant',
        });
      }
    }
  } else if (window.history.scrollRestoration === 'manual') {
    // reset scrollRestoration to 'auto' if there is no hash in url and we set it to 'manual' before
    window.history.scrollRestoration = 'auto';
  }

  $(document).on('click', '.show-outdated', function (e) {
    e.preventDefault();
    const id = this.getAttribute('data-comment');
    hideElem(this);
    showElem(`#code-comments-${id}`);
    showElem(`#code-preview-${id}`);
    showElem(`#hide-outdated-${id}`);
  });

  $(document).on('click', '.hide-outdated', function (e) {
    e.preventDefault();
    const id = this.getAttribute('data-comment');
    hideElem(this);
    hideElem(`#code-comments-${id}`);
    hideElem(`#code-preview-${id}`);
    showElem(`#show-outdated-${id}`);
  });

  $(document).on('click', 'button.comment-form-reply', async function (e) {
    e.preventDefault();
    await handleReply($(this));
  });

  const $reviewBox = $('.review-box-panel');
  if ($reviewBox.length === 1) {
    const _promise = initComboMarkdownEditor($reviewBox.find('.combo-markdown-editor'));
  }

  // The following part is only for diff views
  if (!$('.repository.pull.diff').length) return;

  const $reviewBtn = $reviewBox.parent().find('.js-btn-review');
  const $panel = $reviewBox.parent().find('.review-box-panel');
  const $closeBtn = $panel.find('.close');

  if ($reviewBtn.length && $panel.length) {
    const tippy = createTippy($reviewBtn[0], {
      content: $panel[0],
      placement: 'bottom',
      trigger: 'click',
      maxWidth: 'none',
      interactive: true,
      hideOnClick: true,
    });

    $closeBtn.on('click', (e) => {
      e.preventDefault();
      tippy.hide();
    });
  }

  $(document).on('click', '.add-code-comment', async function (e) {
    if (e.target.classList.contains('btn-add-single')) return; // https://github.com/go-gitea/gitea/issues/4745
    e.preventDefault();

    const isSplit = this.closest('.code-diff')?.classList.contains('code-diff-split');
    const side = this.getAttribute('data-side');
    const idx = this.getAttribute('data-idx');
    const path = this.closest('[data-path]')?.getAttribute('data-path');
    const tr = this.closest('tr');
    const lineType = tr.getAttribute('data-line-type');

    const ntr = tr.nextElementSibling;
    let $ntr = $(ntr);
    if (!ntr?.classList.contains('add-comment')) {
      $ntr = $(`
        <tr class="add-comment" data-line-type="${lineType}">
          ${isSplit ? `
            <td class="add-comment-left" colspan="4"></td>
            <td class="add-comment-right" colspan="4"></td>
          ` : `
            <td class="add-comment-left add-comment-right" colspan="5"></td>
          `}
        </tr>`);
      $(tr).after($ntr);
    }

    const $td = $ntr.find(`.add-comment-${side}`);
    const $commentCloud = $td.find('.comment-code-cloud');
    if (!$commentCloud.length && !$ntr.find('button[name="pending_review"]').length) {
      try {
        const response = await GET(this.closest('[data-new-comment-url]')?.getAttribute('data-new-comment-url'));
        const html = await response.text();
        $td.html(html);
        $td.find("input[name='line']").val(idx);
        $td.find("input[name='side']").val(side === 'left' ? 'previous' : 'proposed');
        $td.find("input[name='path']").val(path);

        initDropzone($td.find('.dropzone')[0]);
        const editor = await initComboMarkdownEditor($td.find('.combo-markdown-editor'));
        editor.focus();
      } catch (error) {
        console.error(error);
      }
    }
  });
}

export function initRepoIssueReferenceIssue() {
  // Reference issue
  $(document).on('click', '.reference-issue', function (event) {
    const $this = $(this);
    const content = $(`#${$this.data('target')}`).text();
    const poster = $this.data('poster-username');
    const reference = toAbsoluteUrl($this.data('reference'));
    const $modal = $($this.data('modal'));
    $modal.find('textarea[name="content"]').val(`${content}\n\n_Originally posted by @${poster} in ${reference}_`);
    $modal.modal('show');

    event.preventDefault();
  });
}

export function initRepoIssueWipToggle() {
  // Toggle WIP
  $('.toggle-wip a, .toggle-wip button').on('click', async (e) => {
    e.preventDefault();
    const toggleWip = e.currentTarget.closest('.toggle-wip');
    const title = toggleWip.getAttribute('data-title');
    const wipPrefixes = JSON.parse(toggleWip.getAttribute('data-wip-prefixes'));
    const updateUrl = toggleWip.getAttribute('data-update-url');
    const prefix = wipPrefixes.find((prefix) => title.startsWith(prefix));

    try {
      const params = new URLSearchParams();
      params.append('title', prefix !== undefined ? title.slice(prefix.length).trim() : `${wipPrefixes[0].trim()} ${title}`);

      const response = await POST(updateUrl, {data: params});
      if (!response.ok) {
        throw new Error('Failed to toggle WIP status');
      }
      window.location.reload();
    } catch (error) {
      console.error(error);
    }
  });
}

export function initRepoIssueTitleEdit() {
  const issueTitleDisplay = document.querySelector('#issue-title-display');
  const issueTitleEditor = document.querySelector('#issue-title-editor');
  if (!issueTitleEditor) return;

  const issueTitleInput = issueTitleEditor.querySelector('input');
  const oldTitle = issueTitleInput.getAttribute('data-old-title');
  const normalModeElements = [issueTitleDisplay, '#pull-desc-display', '#agit-label', '#editable-label'];
  issueTitleDisplay.querySelector('#issue-title-edit-show').addEventListener('click', () => {
    for (const element of normalModeElements) {
      hideElem(element);
    }
    showElem(issueTitleEditor);
    showElem('#pull-desc-editor');
    if (!issueTitleInput.value.trim()) {
      issueTitleInput.value = oldTitle;
    }
    issueTitleInput.focus();
  });
  issueTitleEditor.querySelector('.ui.cancel.button').addEventListener('click', () => {
    hideElem(issueTitleEditor);
    hideElem('#pull-desc-editor');
    for (const element of normalModeElements) {
      showElem(element);
    }
  });

  const pullDescEditor = document.querySelector('#pull-desc-editor'); // it may not exist for a merged PR
  const prTargetUpdateUrl = pullDescEditor?.getAttribute('data-target-update-url');

  const editSaveButton = issueTitleEditor.querySelector('.ui.primary.button');
  const saveAndRefresh = async () => {
    const newTitle = issueTitleInput.value.trim();
    try {
      if (newTitle && newTitle !== oldTitle) {
        const resp = await POST(editSaveButton.getAttribute('data-update-url'), {data: new URLSearchParams({title: newTitle})});
        if (!resp.ok) {
          throw new Error(`Failed to update issue title: ${resp.statusText}`);
        }
      }
      if (prTargetUpdateUrl) {
        const newTargetBranch = document.querySelector('#pull-target-branch').getAttribute('data-branch');
        const oldTargetBranch = document.querySelector('#branch_target').textContent;
        if (newTargetBranch !== oldTargetBranch) {
          const resp = await POST(prTargetUpdateUrl, {data: new URLSearchParams({target_branch: newTargetBranch})});
          if (!resp.ok) {
            throw new Error(`Failed to update PR target branch: ${resp.statusText}`);
          }
        }
      }
      window.location.reload();
    } catch (error) {
      console.error(error);
      showErrorToast(error.message);
    }
  };
  editSaveButton.addEventListener('click', saveAndRefresh);
  issueTitleEditor.querySelector('input').addEventListener('ce-quick-submit', saveAndRefresh);
}

export function initRepoIssueBranchSelect() {
  document.querySelector('#branch-select')?.addEventListener('click', (e) => {
    const el = e.target.closest('.item[data-branch]');
    if (!el) return;
    const pullTargetBranch = document.querySelector('#pull-target-branch');
    const baseName = pullTargetBranch.getAttribute('data-basename');
    const branchNameNew = el.getAttribute('data-branch');
    const branchNameOld = pullTargetBranch.getAttribute('data-branch');
    pullTargetBranch.textContent = pullTargetBranch.textContent.replace(`${baseName}:${branchNameOld}`, `${baseName}:${branchNameNew}`);
    pullTargetBranch.setAttribute('data-branch', branchNameNew);
  });
}

export function initRepoIssueAssignMe() {
  // Assign to me button
  document.querySelector('.ui.assignees.list .item.no-select .select-assign-me')
    ?.addEventListener('click', (e) => {
      e.preventDefault();
      const selectMe = e.target;
      const noSelect = selectMe.parentElement;
      const selectorList = document.querySelector('.ui.select-assignees .menu');

      if (selectMe.getAttribute('data-action') === 'update') {
        (async () => {
          await updateIssuesMeta(
            selectMe.getAttribute('data-update-url'),
            selectMe.getAttribute('data-action'),
            selectMe.getAttribute('data-issue-id'),
            selectMe.getAttribute('data-id'),
          );
          reloadConfirmDraftComment();
        })();
      } else {
        for (const item of selectorList.querySelectorAll('.item')) {
          if (item.getAttribute('data-id') === selectMe.getAttribute('data-id')) {
            item.classList.add('checked');
            item.querySelector('.octicon-check').classList.remove('tw-invisible');
          }
        }
        document.querySelector(selectMe.getAttribute('data-id-selector')).classList.remove('tw-hidden');
        noSelect.classList.add('tw-hidden');
        document.querySelector(selectorList.getAttribute('data-id')).value = selectMe.getAttribute('data-id');
        return false;
      }
    });
}

export function initSingleCommentEditor($commentForm) {
  // pages:
  // * normal new issue/pr page, no status-button
  // * issue/pr view page, with comment form, has status-button
  const opts = {};
  const statusButton = document.getElementById('status-button');
  if (statusButton) {
    opts.onContentChanged = (editor) => {
      const statusText = statusButton.getAttribute(editor.value().trim() ? 'data-status-and-comment' : 'data-status');
      statusButton.textContent = statusText;
    };
  }
  initComboMarkdownEditor($commentForm.find('.combo-markdown-editor'), opts);
}

export function initIssueTemplateCommentEditors($commentForm) {
  // pages:
  // * new issue with issue template
  const $comboFields = $commentForm.find('.combo-editor-dropzone');

  const initCombo = async ($combo) => {
    const $dropzoneContainer = $combo.find('.form-field-dropzone');
    const $formField = $combo.find('.form-field-real');
    const $markdownEditor = $combo.find('.combo-markdown-editor');

    const editor = await initComboMarkdownEditor($markdownEditor, {
      onContentChanged: (editor) => {
        $formField.val(editor.value());
      },
    });

    $formField.on('focus', async () => {
      // deactivate all markdown editors
      showElem($commentForm.find('.combo-editor-dropzone .form-field-real'));
      hideElem($commentForm.find('.combo-editor-dropzone .combo-markdown-editor'));
      hideElem($commentForm.find('.combo-editor-dropzone .form-field-dropzone'));

      // activate this markdown editor
      hideElem($formField);
      showElem($markdownEditor);
      showElem($dropzoneContainer);

      await editor.switchToUserPreference();
      editor.focus();
    });
  };

  for (const el of $comboFields) {
    initCombo($(el));
  }
}

// This function used to show and hide archived label on issue/pr
//  page in the sidebar where we select the labels
//  If we have any archived label tagged to issue and pr. We will show that
//  archived label with checked classed otherwise we will hide it
//  with the help of this function.
//  This function runs globally.
export function initArchivedLabelHandler() {
  if (!document.querySelector('.archived-label-hint')) return;
  for (const label of document.querySelectorAll('[data-is-archived]')) {
    toggleElem(label, label.classList.contains('checked'));
  }
}

// Render the issue's title. It converts emojis and code blocks syntax into their respective HTML equivalent.
export function issueTitleHTML(title) {
  return title.replaceAll(/:[-+\w]+:/g, (emoji) => emojiHTML(emoji.substring(1, emoji.length - 1)))
    .replaceAll(/`[^`]+`/g, (code) => `<code class="inline-code-block">${code.substring(1, code.length - 1)}</code>`);
}
