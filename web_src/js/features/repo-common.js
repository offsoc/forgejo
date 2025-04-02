import $ from 'jquery';
import {hideElem, queryElems, showElem} from '../utils/dom.js';
import {POST} from '../modules/fetch.js';
import {showErrorToast} from '../modules/toast.js';
import {sleep} from '../utils.js';

async function onDownloadArchive(e) {
  e.preventDefault();
  // there are many places using the "archive-link", eg: the dropdown on the repo code page, the release list
  const el = e.target.closest('a.archive-link[href]');
  const targetLoading = el.closest('.ui.dropdown') ?? el;
  targetLoading.classList.add('is-loading', 'loading-icon-2px');
  try {
    for (let tryCount = 0; ;tryCount++) {
      const response = await POST(el.href);
      if (!response.ok) throw new Error(`Invalid server response: ${response.status}`);

      const data = await response.json();
      if (data.complete) break;
      await sleep(Math.min((tryCount + 1) * 750, 2000));
    }
    window.location.href = el.href; // the archive is ready, start real downloading
  } catch (e) {
    console.error(e);
    showErrorToast(`Failed to download the archive: ${e}`, {duration: 2500});
  } finally {
    targetLoading.classList.remove('is-loading', 'loading-icon-2px');
  }
}

export function initRepoArchiveLinks() {
  queryElems('a.archive-link[href]', (el) => el.addEventListener('click', onDownloadArchive));
}

export function initRepoCloneLink() {
  const repoCloneSSH = document.getElementById('repo-clone-ssh');
  const repoCloneHTTPS = document.getElementById('repo-clone-https');
  const inputLink = document.getElementById('repo-clone-url');

  if ((!repoCloneSSH && !repoCloneHTTPS) || !inputLink) {
    return;
  }

  repoCloneSSH?.addEventListener('click', () => {
    localStorage.setItem('repo-clone-protocol', 'ssh');
    window.updateCloneStates();
  });
  repoCloneHTTPS?.addEventListener('click', () => {
    localStorage.setItem('repo-clone-protocol', 'https');
    window.updateCloneStates();
  });

  inputLink.addEventListener('focus', () => {
    inputLink.select();
  });
}

export function initRepoCommonBranchOrTagDropdown(selector) {
  $(selector).each(function () {
    const $dropdown = $(this);
    $dropdown.find('.reference.column').on('click', function () {
      hideElem($dropdown.find('.scrolling.reference-list-menu'));
      showElem($($(this).data('target')));
      return false;
    });
  });
}

export function initRepoCommonFilterSearchDropdown(selector) {
  const $dropdown = $(selector);
  if (!$dropdown.length) return;

  $dropdown.dropdown({
    fullTextSearch: 'exact',
    selectOnKeydown: false,
    onChange(_text, _value, $choice) {
      if ($choice[0].getAttribute('data-url')) {
        window.location.href = $choice[0].getAttribute('data-url');
      }
    },
    message: {noResults: $dropdown[0].getAttribute('data-no-results')},
  });
}
