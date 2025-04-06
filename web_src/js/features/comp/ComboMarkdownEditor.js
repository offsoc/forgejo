import '@github/markdown-toolbar-element';
import '@github/text-expander-element';
import $ from 'jquery';
import {attachTribute} from '../tribute.js';
import {hideElem, showElem, autosize, isElemVisible, replaceTextareaSelection} from '../../utils/dom.js';
import {initEasyMDEPaste, initTextareaPaste} from './Paste.js';
import {handleGlobalEnterQuickSubmit} from './QuickSubmit.js';
import {renderPreviewPanelContent} from '../repo-editor.js';
import {easyMDEToolbarActions} from './EasyMDEToolbarActions.js';
import {initTextExpander} from './TextExpander.js';
import {showErrorToast} from '../../modules/toast.js';
import {POST} from '../../modules/fetch.js';

let elementIdCounter = 0;

/**
 * validate if the given textarea is non-empty.
 * @param {HTMLElement} textarea - The textarea element to be validated.
 * @returns {boolean} returns true if validation succeeded.
 */
export function validateTextareaNonEmpty(textarea) {
  // When using EasyMDE, the original edit area HTML element is hidden, breaking HTML5 input validation.
  // The workaround (https://github.com/sparksuite/simplemde-markdown-editor/issues/324) doesn't work with contenteditable, so we just show an alert.
  if (!textarea.value) {
    if (isElemVisible(textarea)) {
      textarea.required = true;
      const form = textarea.closest('form');
      form?.reportValidity();
    } else {
      // The alert won't hurt users too much, because we are dropping the EasyMDE and the check only occurs in a few places.
      showErrorToast('Require non-empty content');
    }
    return false;
  }
  return true;
}

class ComboMarkdownEditor {
  constructor(container, options = {}) {
    container._giteaComboMarkdownEditor = this;
    this.options = options;
    this.container = container;
  }

  async init() {
    this.prepareEasyMDEToolbarActions();
    this.setupContainer();
    this.setupTab();
    this.setupDropzone();
    this.setupTextarea();
    this.setupTableInserter();
    this.setupLinkInserter();

    await this.switchToUserPreference();

    elementIdCounter++;
  }

  applyEditorHeights(el, heights) {
    if (!heights) return;
    if (heights.minHeight) el.style.minHeight = heights.minHeight;
    if (heights.height) el.style.height = heights.height;
    if (heights.maxHeight) el.style.maxHeight = heights.maxHeight;
  }

  setupContainer() {
    initTextExpander(this.container.querySelector('text-expander'));
    this.container.addEventListener('ce-editor-content-changed', (e) => this.options?.onContentChanged?.(this, e));
  }

  setupTextarea() {
    this.textarea = this.container.querySelector('.markdown-text-editor');
    this.textarea._giteaComboMarkdownEditor = this;
    this.textarea.id = `_combo_markdown_editor_${elementIdCounter}`;
    this.textarea.addEventListener('input', (e) => this.options?.onContentChanged?.(this, e));
    this.applyEditorHeights(this.textarea, this.options.editorHeights);

    if (this.textarea.getAttribute('data-disable-autosize') !== 'true') {
      this.textareaAutosize = autosize(this.textarea, {viewportMarginBottom: 130});
    }

    this.textareaMarkdownToolbar = this.container.querySelector('markdown-toolbar');
    this.textareaMarkdownToolbar.setAttribute('for', this.textarea.id);
    for (const el of this.textareaMarkdownToolbar.querySelectorAll('.markdown-toolbar-button')) {
      // upstream bug: The role code is never executed in base MarkdownButtonElement https://github.com/github/markdown-toolbar-element/issues/70
      el.setAttribute('role', 'button');
      // the editor usually is in a form, so the buttons should have "type=button", avoiding conflicting with the form's submit.
      if (el.nodeName === 'BUTTON' && !el.getAttribute('type')) el.setAttribute('type', 'button');
    }
    this.textareaMarkdownToolbar.querySelector('button[data-md-action="indent"]')?.addEventListener('click', () => {
      this.indentSelection(false);
    });
    this.textareaMarkdownToolbar.querySelector('button[data-md-action="unindent"]')?.addEventListener('click', () => {
      this.indentSelection(true);
    });
    this.textareaMarkdownToolbar.querySelector('button[data-md-action="new-table"]')?.setAttribute('data-modal', `div[data-markdown-table-modal-id="${elementIdCounter}"]`);
    this.textareaMarkdownToolbar.querySelector('button[data-md-action="new-link"]')?.setAttribute('data-modal', `div[data-markdown-link-modal-id="${elementIdCounter}"]`);

    this.textarea.addEventListener('keydown', (e) => {
      if (e.shiftKey) {
        e.target._shiftDown = true;
      }
      if (e.key === 'Enter' && !e.shiftKey && !e.ctrlKey && !e.altKey) {
        // Prevent special line break handling if currently a text expander popup is open
        if (this.textarea.hasAttribute('aria-expanded')) return;
        if (!this.breakLine()) return; // Nothing changed, let the default handler work.
        this.options?.onContentChanged?.(this, e);
        e.preventDefault();
      }
    });
    this.textarea.addEventListener('keyup', (e) => {
      if (!e.shiftKey) {
        e.target._shiftDown = false;
      }
    });

    const monospaceButton = this.container.querySelector('.markdown-switch-monospace');
    const monospaceEnabled = localStorage?.getItem('markdown-editor-monospace') === 'true';
    const monospaceText = monospaceButton.getAttribute(monospaceEnabled ? 'data-disable-text' : 'data-enable-text');
    monospaceButton.setAttribute('data-tooltip-content', monospaceText);
    monospaceButton.setAttribute('aria-checked', String(monospaceEnabled));

    monospaceButton?.addEventListener('click', (e) => {
      e.preventDefault();
      const enabled = localStorage?.getItem('markdown-editor-monospace') !== 'true';
      localStorage.setItem('markdown-editor-monospace', String(enabled));
      this.textarea.classList.toggle('tw-font-mono', enabled);
      const text = monospaceButton.getAttribute(enabled ? 'data-disable-text' : 'data-enable-text');
      monospaceButton.setAttribute('data-tooltip-content', text);
      monospaceButton.setAttribute('aria-checked', String(enabled));
    });

    const easymdeButton = this.container.querySelector('.markdown-switch-easymde');
    easymdeButton?.addEventListener('click', async (e) => {
      e.preventDefault();
      this.userPreferredEditor = 'easymde';
      await this.switchToEasyMDE();
    });

    if (this.dropzone) {
      initTextareaPaste(this.textarea, this.dropzone);
    }
  }

  setupDropzone() {
    const dropzoneParentContainer = this.container.getAttribute('data-dropzone-parent-container');
    if (dropzoneParentContainer) {
      this.dropzone = this.container.closest(this.container.getAttribute('data-dropzone-parent-container'))?.querySelector('.dropzone');
    }
  }

  setupTab() {
    const $container = $(this.container);
    const tabs = $container[0].querySelectorAll('.switch > .item');

    // Fomantic Tab requires the "data-tab" to be globally unique.
    // So here it uses our defined "data-tab-for" and "data-tab-panel" to generate the "data-tab" attribute for Fomantic.
    const tabEditor = Array.from(tabs).find((tab) => tab.getAttribute('data-tab-for') === 'markdown-writer');
    const tabPreviewer = Array.from(tabs).find((tab) => tab.getAttribute('data-tab-for') === 'markdown-previewer');
    tabEditor.setAttribute('data-tab', `markdown-writer-${elementIdCounter}`);
    tabPreviewer.setAttribute('data-tab', `markdown-previewer-${elementIdCounter}`);
    const toolbar = $container[0].querySelector('markdown-toolbar');
    const panelEditor = $container[0].querySelector('.ui.tab[data-tab-panel="markdown-writer"]');
    const panelPreviewer = $container[0].querySelector('.ui.tab[data-tab-panel="markdown-previewer"]');
    panelEditor.setAttribute('data-tab', `markdown-writer-${elementIdCounter}`);
    panelPreviewer.setAttribute('data-tab', `markdown-previewer-${elementIdCounter}`);

    tabEditor.addEventListener('click', () => {
      toolbar.classList.remove('markdown-toolbar-hidden');
      requestAnimationFrame(() => {
        this.focus();
      });
    });

    $(tabs).tab();

    this.previewUrl = tabPreviewer.getAttribute('data-preview-url');
    this.previewContext = tabPreviewer.getAttribute('data-preview-context');
    this.previewMode = this.options.previewMode ?? 'comment';
    this.previewWiki = this.options.previewWiki ?? false;
    tabPreviewer.addEventListener('click', async () => {
      toolbar.classList.add('markdown-toolbar-hidden');
      const formData = new FormData();
      formData.append('mode', this.previewMode);
      formData.append('context', this.previewContext);
      formData.append('text', this.value());
      formData.append('wiki', this.previewWiki);
      const response = await POST(this.previewUrl, {data: formData});
      const data = await response.text();
      renderPreviewPanelContent($(panelPreviewer), data);
    });
  }

  addNewTable(event) {
    const elementId = event.target.getAttribute('data-element-id');
    const newTableModal = document.querySelector(`div[data-markdown-table-modal-id="${elementId}"]`);
    const form = newTableModal.querySelector('div[data-selector-name="form"]');

    // Validate input fields
    for (const currentInput of form.querySelectorAll('input')) {
      if (!currentInput.checkValidity()) {
        currentInput.reportValidity();
        return;
      }
    }

    let headerText = form.querySelector('input[name="table-header"]').value;
    let contentText = form.querySelector('input[name="table-content"]').value;
    const rowCount = parseInt(form.querySelector('input[name="table-rows"]').value);
    const columnCount = parseInt(form.querySelector('input[name="table-columns"]').value);

    headerText = headerText.padEnd(contentText.length);
    contentText = contentText.padEnd(headerText.length);

    let code = `| ${(new Array(columnCount)).fill(headerText).join(' | ')} |\n`;
    code += `|-${(new Array(columnCount)).fill('-'.repeat(headerText.length)).join('-|-')}-|\n`;
    for (let i = 0; i < rowCount; i++) {
      code += `| ${(new Array(columnCount)).fill(contentText).join(' | ')} |\n`;
    }

    replaceTextareaSelection(document.getElementById(`_combo_markdown_editor_${elementId}`), code);

    // Close the modal
    newTableModal.querySelector('button[data-selector-name="cancel-button"]').click();
  }

  setupTableInserter() {
    const newTableModal = this.container.querySelector('div[data-modal-name="new-markdown-table"]');
    newTableModal.setAttribute('data-markdown-table-modal-id', elementIdCounter);

    const button = newTableModal.querySelector('button[data-selector-name="ok-button"]');
    button.setAttribute('data-element-id', elementIdCounter);
    button.addEventListener('click', this.addNewTable);
  }

  addNewLink(event) {
    const elementId = event.target.getAttribute('data-element-id');
    const newLinkModal = document.querySelector(`div[data-markdown-link-modal-id="${elementId}"]`);
    const form = newLinkModal.querySelector('div[data-selector-name="form"]');

    // Validate input fields
    for (const currentInput of form.querySelectorAll('input')) {
      if (!currentInput.checkValidity()) {
        currentInput.reportValidity();
        return;
      }
    }

    const url = form.querySelector('input[name="link-url"]').value;
    const description = form.querySelector('input[name="link-description"]').value;

    const code = `[${description}](${url})`;

    replaceTextareaSelection(document.getElementById(`_combo_markdown_editor_${elementId}`), code);

    // Close the modal then clear its fields in case the user wants to add another one.
    newLinkModal.querySelector('button[data-selector-name="cancel-button"]').click();
    form.querySelector('input[name="link-url"]').value = '';
    form.querySelector('input[name="link-description"]').value = '';
  }

  setupLinkInserter() {
    const newLinkModal = this.container.querySelector('div[data-modal-name="new-markdown-link"]');
    newLinkModal.setAttribute('data-markdown-link-modal-id', elementIdCounter);
    const textarea = document.getElementById(`_combo_markdown_editor_${elementIdCounter}`);

    $(newLinkModal).modal({
      // Pre-fill the description field from the selection to create behavior similar
      // to pasting an URL over selected text.
      onShow: () => {
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;

        if (start !== end) {
          const selection = textarea.value.slice(start ?? undefined, end ?? undefined);
          newLinkModal.querySelector('input[name="link-description"]').value = selection;
        } else {
          newLinkModal.querySelector('input[name="link-description"]').value = '';
        }
      },
    });

    const button = newLinkModal.querySelector('button[data-selector-name="ok-button"]');
    button.setAttribute('data-element-id', elementIdCounter);
    button.addEventListener('click', this.addNewLink);
  }

  prepareEasyMDEToolbarActions() {
    this.easyMDEToolbarDefault = [
      'bold', 'italic', 'strikethrough', '|', 'heading-1', 'heading-2', 'heading-3',
      'heading-bigger', 'heading-smaller', '|', 'code', 'quote', '|', 'gitea-checkbox-empty',
      'gitea-checkbox-checked', '|', 'unordered-list', 'ordered-list', '|', 'link', 'image',
      'table', 'horizontal-rule', '|', 'gitea-switch-to-textarea',
    ];
  }

  parseEasyMDEToolbar(EasyMDE, actions) {
    this.easyMDEToolbarActions = this.easyMDEToolbarActions || easyMDEToolbarActions(EasyMDE, this);
    const processed = [];
    for (const action of actions) {
      const actionButton = this.easyMDEToolbarActions[action];
      if (!actionButton) throw new Error(`Unknown EasyMDE toolbar action ${action}`);
      processed.push(actionButton);
    }
    return processed;
  }

  async switchToUserPreference() {
    if (this.userPreferredEditor === 'easymde') {
      await this.switchToEasyMDE();
    } else {
      this.switchToTextarea();
    }
  }

  switchToTextarea() {
    if (!this.easyMDE) return;
    showElem(this.textareaMarkdownToolbar);
    if (this.easyMDE) {
      this.easyMDE.toTextArea();
      this.easyMDE = null;
    }
  }

  async switchToEasyMDE() {
    if (this.easyMDE) return;
    // EasyMDE's CSS should be loaded via webpack config, otherwise our own styles can not overwrite the default styles.
    const {default: EasyMDE} = await import(/* webpackChunkName: "easymde" */'easymde');
    const easyMDEOpt = {
      autoDownloadFontAwesome: false,
      element: this.textarea,
      forceSync: true,
      renderingConfig: {singleLineBreaks: false},
      indentWithTabs: false,
      tabSize: 4,
      spellChecker: false,
      inputStyle: 'contenteditable', // nativeSpellcheck requires contenteditable
      nativeSpellcheck: true,
      ...this.options.easyMDEOptions,
    };
    easyMDEOpt.toolbar = this.parseEasyMDEToolbar(EasyMDE, easyMDEOpt.toolbar ?? this.easyMDEToolbarDefault);

    this.easyMDE = new EasyMDE(easyMDEOpt);
    this.easyMDE.codemirror.on('change', (...args) => {this.options?.onContentChanged?.(this, ...args)});
    this.easyMDE.codemirror.setOption('extraKeys', {
      'Cmd-Enter': (cm) => handleGlobalEnterQuickSubmit(cm.getTextArea()),
      'Ctrl-Enter': (cm) => handleGlobalEnterQuickSubmit(cm.getTextArea()),
      Enter: (cm) => {
        const tributeContainer = document.querySelector('.tribute-container');
        if (!tributeContainer || tributeContainer.style.display === 'none') {
          cm.execCommand('newlineAndIndent');
        }
      },
      Up: (cm) => {
        const tributeContainer = document.querySelector('.tribute-container');
        if (!tributeContainer || tributeContainer.style.display === 'none') {
          return cm.execCommand('goLineUp');
        }
      },
      Down: (cm) => {
        const tributeContainer = document.querySelector('.tribute-container');
        if (!tributeContainer || tributeContainer.style.display === 'none') {
          return cm.execCommand('goLineDown');
        }
      },
    });
    this.applyEditorHeights(this.container.querySelector('.CodeMirror-scroll'), this.options.editorHeights);
    await attachTribute(this.easyMDE.codemirror.getInputField(), {mentions: true, emoji: true});
    initEasyMDEPaste(this.easyMDE, this.dropzone);
    hideElem(this.textareaMarkdownToolbar);
  }

  value(v = undefined) {
    if (v === undefined) {
      if (this.easyMDE) {
        return this.easyMDE.value();
      }
      return this.textarea.value;
    }

    if (this.easyMDE) {
      this.easyMDE.value(v);
    } else {
      this.textarea.value = v;
    }
    this.textareaAutosize?.resizeToFit();
  }

  focus() {
    if (this.easyMDE) {
      this.easyMDE.codemirror.focus();
    } else {
      this.textarea.focus();
    }
  }

  moveCursorToEnd() {
    this.textarea.focus();
    this.textarea.setSelectionRange(this.textarea.value.length, this.textarea.value.length);
    if (this.easyMDE) {
      this.easyMDE.codemirror.focus();
      this.easyMDE.codemirror.setCursor(this.easyMDE.codemirror.lineCount(), 0);
    }
  }

  indentSelection(unindent) {
    // Indent with 4 spaces, unindent 4 spaces or fewer or a lost tab.
    const indentPrefix = '    ';
    const unindentRegex = /^( {1,4}|\t)/;

    // Indent all lines that are included in the selection, partially or whole, while preserving the original selection at the end.
    const lines = this.textarea.value.split('\n');
    const changedLines = [];
    // The current selection or cursor position.
    const [start, end] = [this.textarea.selectionStart, this.textarea.selectionEnd];
    // The range containing whole lines that will effectively be replaced.
    let [editStart, editEnd] = [start, end];
    // The range that needs to be re-selected to match previous selection.
    let [newStart, newEnd] = [start, end];
    // The start and end position of the current line (where end points to the newline or EOF)
    let [lineStart, lineEnd] = [0, 0];

    for (const line of lines) {
      lineEnd = lineStart + line.length + 1;
      if (lineEnd <= start) {
        lineStart = lineEnd;
        continue;
      }

      const updated = unindent ? line.replace(unindentRegex, '') : indentPrefix + line;
      changedLines.push(updated);
      const move = updated.length - line.length;

      if (start >= lineStart && start < lineEnd) {
        editStart = lineStart;
        newStart = Math.max(start + move, lineStart);
      }

      newEnd += move;
      editEnd = lineEnd - 1;
      lineStart = lineEnd;
      if (lineStart > end) break;
    }

    // Update changed lines whole.
    const text = changedLines.join('\n');
    this.textarea.focus();
    this.textarea.setSelectionRange(editStart, editEnd);
    if (!document.execCommand('insertText', false, text)) {
      // execCommand is deprecated, but setRangeText (and any other direct value modifications) erases the native undo history.
      // So only fall back to it if execCommand fails.
      this.textarea.setRangeText(text);
    }

    // Set selection to (effectively) be the same as before.
    this.textarea.setSelectionRange(newStart, Math.max(newStart, newEnd));
  }

  breakLine() {
    const [start, end] = [this.textarea.selectionStart, this.textarea.selectionEnd];

    // Do nothing if a range is selected
    if (start !== end) return false;

    const value = this.textarea.value;
    // Find the beginning of the current line.
    const lineStart = Math.max(0, value.lastIndexOf('\n', start - 1) + 1);
    // Find the end and extract the line.
    const nextLF = value.indexOf('\n', start);
    const lineEnd = nextLF === -1 ? value.length : nextLF;
    const line = value.slice(lineStart, lineEnd);
    // Match any whitespace at the start + any repeatable prefix + exactly one space after.
    const prefix = line.match(/^\s*((\d+)[.)]\s|[-*+]\s{1,4}\[[ x]\]\s?|[-*+]\s|(>\s?)+)?/);

    // Defer to browser if we can't do anything more useful, or if the cursor is inside the prefix.
    if (!prefix) return false;
    const prefixLength = prefix[0].length;
    if (!prefixLength || lineStart + prefixLength > start) return false;
    // If the prefix is just indentation (which should always be an even number of spaces or tabs), check if a single whitespace is added to the end of the line.
    // If this is the case do not leave the indentation and continue with the prefix.
    if ((prefixLength % 2 === 1 && /^ +$/.test(prefix[0])) || /^\t+ $/.test(prefix[0])) {
      prefix[0] = prefix[0].slice(0, prefixLength - 1);
    } else if (prefixLength === lineEnd - lineStart) {
      this.textarea.setSelectionRange(lineStart, lineEnd);
      if (!document.execCommand('insertText', false, '\n')) {
        this.textarea.setRangeText('\n');
      }
      return true;
    }

    // Insert newline + prefix.
    let text = `\n${prefix[0]}`;
    // Increment a number if present. (perhaps detecting repeating 1. and not doing that then would be a good idea)
    const num = text.match(/\d+/);
    if (num) text = text.replace(num[0], Number(num[0]) + 1);
    text = text.replace('[x]', '[ ]');

    if (!document.execCommand('insertText', false, text)) {
      this.textarea.setRangeText(text);
    }

    return true;
  }

  get userPreferredEditor() {
    return window.localStorage.getItem(`markdown-editor-${this.options.useScene ?? 'default'}`);
  }
  set userPreferredEditor(s) {
    window.localStorage.setItem(`markdown-editor-${this.options.useScene ?? 'default'}`, s);
  }
}

export function getComboMarkdownEditor(el) {
  if (el instanceof $) el = el[0];
  return el?._giteaComboMarkdownEditor;
}

export async function initComboMarkdownEditor(container, options = {}) {
  if (container instanceof $) {
    if (container.length !== 1) {
      throw new Error('initComboMarkdownEditor: container must be a single element');
    }
    container = container[0];
  }
  if (!container) {
    throw new Error('initComboMarkdownEditor: container is null');
  }
  const editor = new ComboMarkdownEditor(container, options);
  await editor.init();
  return editor;
}
