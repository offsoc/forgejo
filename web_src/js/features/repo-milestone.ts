import {initComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';

export function initRepoMilestoneEditor() {
  const editor = document.querySelector('.page-content.repository.milestone .combo-markdown-editor');
  if (!editor) {
    return;
  }
  initComboMarkdownEditor(editor);
}
