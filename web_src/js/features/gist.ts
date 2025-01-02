import {showTemporaryTooltip} from '../modules/tippy.js';
import {initRepoCloneLink} from './repo-common.js';
import {clippie} from 'clippie';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const {i18n} = (window as any).config;

function addGistFileButtonClicked() {
  const template = document.getElementById('file-field-template') as HTMLTemplateElement;
  const clone = document.importNode(template.content, true);

  let currentID = 0;
  while (true) {
    if (document.querySelector(`input[name="file-name-${currentID}"]`) === null) {
      break;
    } else {
      currentID += 1;
    }
  }

  clone.querySelector('[data-template-name="field"]').setAttribute('id', `file-field-${currentID}`);
  clone.querySelector('[data-template-name="name-label"]').setAttribute('for', `file-name-${currentID}`);
  clone.querySelector('[data-template-name="name-input"]').setAttribute('name', `file-name-${currentID}`);
  clone.querySelector('[data-template-name="content-label"]').setAttribute('for', `file-content-${currentID}`);
  clone.querySelector('[data-template-name="content-input"]').setAttribute('name', `file-content-${currentID}`);

  const deleteButton = clone.querySelector('[data-template-name="delete-button"]') as HTMLButtonElement;
  deleteButton.setAttribute('data-file-id', currentID.toString());
  deleteButton.addEventListener('click', deleteFileButtonClicked);

  document.getElementById('file-field-container').append(clone);
}

function deleteFileButtonClicked(event: Event) {
  const fileID = (event.target as HTMLButtonElement).getAttribute('data-file-id');
  const fileField = document.getElementById(`file-field-${fileID}`);
  fileField.remove();
}

function initAddGistFileButton() {
  const button = document.getElementById('add-gist-file-button');

  if (button !== null) {
    button.addEventListener('click', addGistFileButtonClicked);
  }
}

function initGistCopyContent() {
  for (const elem of document.querySelectorAll('span[data-gist-copy-content]')) {
    elem.addEventListener('click', async (event: Event) => {
      const target = event.currentTarget as HTMLSpanElement;

      if (target.classList.contains('is-loading')) {
        return;
      }

      target.classList.add('is-loading', 'loading-icon-2px');

      const fileID = target.getAttribute('data-gist-copy-content');

      const lineEls = document.getElementById(`gist-file-view-${fileID}`).querySelectorAll('.lines-code');
      const text = Array.from(lineEls, (el) => el.textContent).join('');

      const success = await clippie(text);

      if (success) {
        showTemporaryTooltip(target, i18n.copy_success);
      } else {
        showTemporaryTooltip(target, i18n.copy_error);
      }

      target.classList.remove('is-loading', 'loading-icon-2px');
    });
  }
}

function initGistFileDeleteButtons() {
  for (const elem of document.querySelectorAll('#edit-gist-form > * button[data-template-name="delete-button"]')) {
    elem.addEventListener('click', deleteFileButtonClicked);
  }
}

export function initGist() {
  initAddGistFileButton();
  initGistCopyContent();
  initGistFileDeleteButtons();

  if (window.location.pathname.startsWith('/gists')) {
    initRepoCloneLink();
  }
}
