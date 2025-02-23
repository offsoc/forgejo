import {hideElem, showElem} from '../utils/dom.js';
import {initComboMarkdownEditor} from './comp/ComboMarkdownEditor.js';

export function initRepoRelease() {
  for (const el of document.querySelectorAll('.remove-rel-attach')) {
    el.addEventListener('click', (e) => {
      const uuid = e.target.getAttribute('data-uuid');
      const id = e.target.getAttribute('data-id');
      document.querySelector(`input[name='attachment-del-${uuid}']`).value =
        'true';
      hideElem(`#attachment-${id}`);
    });
  }
}

export function initRepoReleaseNew() {
  if (!document.querySelector('.repository.new.release')) return;

  initTagNameEditor();
  initRepoReleaseEditor();
  initAddExternalLinkButton();
}

function initTagNameEditor() {
  const el = document.getElementById('tag-name-editor');
  if (!el) return;

  const existingTags = JSON.parse(el.getAttribute('data-existing-tags'));
  if (!Array.isArray(existingTags)) return;

  const defaultTagHelperText = el.getAttribute('data-tag-helper');
  const newTagHelperText = el.getAttribute('data-tag-helper-new');
  const existingTagHelperText = el.getAttribute('data-tag-helper-existing');

  let previousTag = '';
  document.getElementById('tag-name').addEventListener('keyup', (e) => {
    const value = e.target.value;
    const tagHelper = document.getElementById('tag-helper');
    if (existingTags.includes(value)) {
      // If the tag already exists, hide the target branch selector.
      hideElem('#tag-target-selector');
      tagHelper.textContent = existingTagHelperText;
    } else {
      showElem('#tag-target-selector');
      tagHelper.textContent = value ? newTagHelperText : defaultTagHelperText;
    }

    const title_input = document.getElementById('release-title');
    if (!title_input.value || previousTag === title_input.value) {
      title_input.value = value;
    }
    previousTag = value;
  });
}

function initRepoReleaseEditor() {
  const editor = document.querySelector(
    '.repository.new.release .combo-markdown-editor',
  );
  if (!editor) {
    return;
  }
  initComboMarkdownEditor(editor);
}

let newAttachmentCount = 0;

function initAddExternalLinkButton() {
  const addExternalLinkButton = document.getElementById('add-external-link');
  if (!addExternalLinkButton) return;

  addExternalLinkButton.addEventListener('click', () => {
    newAttachmentCount += 1;
    const attachmentTemplate = document.getElementById('attachment-template');

    const newAttachment = attachmentTemplate.cloneNode(true);
    newAttachment.id = `attachment-N${newAttachmentCount}`;
    newAttachment.classList.remove('tw-hidden');

    const attachmentName = newAttachment.querySelector(
      'input[name="attachment-template-new-name"]',
    );
    attachmentName.name = `attachment-new-name-${newAttachmentCount}`;
    attachmentName.required = true;

    const attachmentExtUrl = newAttachment.querySelector(
      'input[name="attachment-template-new-exturl"]',
    );
    attachmentExtUrl.name = `attachment-new-exturl-${newAttachmentCount}`;
    attachmentExtUrl.required = true;

    const attachmentDel = newAttachment.querySelector('.remove-rel-attach');
    attachmentDel.addEventListener('click', () => {
      newAttachment.remove();
    });

    attachmentTemplate.parentNode.insertBefore(
      newAttachment,
      attachmentTemplate,
    );
  });
}
