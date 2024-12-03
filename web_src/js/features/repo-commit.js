import {createTippy} from '../modules/tippy.js';
import {toggleElem} from '../utils/dom.js';

export function initRepoEllipsisButton() {
  for (const button of document.querySelectorAll('.js-toggle-commit-body')) {
    button.addEventListener('click', function (e) {
      e.preventDefault();
      const expanded = this.getAttribute('aria-expanded') === 'true';
      toggleElem(this.parentElement.querySelector('.commit-body'));
      this.setAttribute('aria-expanded', String(!expanded));
    });
  }
}

export function initCommitStatuses() {
  for (const element of document.querySelectorAll('[data-tippy="commit-statuses"]')) {
    const top = document.querySelector('.repository.file.list') || document.querySelector('.repository.diff');

    createTippy(element, {
      content: element.nextElementSibling,
      placement: top ? 'top-start' : 'bottom-start',
      interactive: true,
      role: 'dialog',
      theme: 'box-with-header',
    });
  }
}

export function initCommitNotes() {
  const notesEditButton = document.getElementById('commit-notes-edit-button');
  if (notesEditButton !== null) {
    notesEditButton.addEventListener('click', () => {
      document.getElementById('commit-notes-display-area').classList.add('tw-hidden');
      document.getElementById('commit-notes-edit-area').classList.remove('tw-hidden');
    });
  }

  const notesAddButton = document.getElementById('commit-notes-add-button');
  if (notesAddButton !== null) {
    notesAddButton.addEventListener('click', () => {
      document.getElementById('commit-notes-add-area').classList.remove('tw-hidden');
    });
  }
}
