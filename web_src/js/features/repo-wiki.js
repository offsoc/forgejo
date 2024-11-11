import {initMarkupContent} from '../markup/content.js';
import {validateTextareaNonEmpty, initComboMarkdownEditor} from './comp/ComboMarkdownEditor.js';
import {fomanticMobileScreen} from '../modules/fomantic.js';
import {POST} from '../modules/fetch.js';

async function initRepoWikiFormEditor() {
  const editArea = document.querySelector('.repository.wiki .combo-markdown-editor textarea');
  if (!editArea) return;

  const form = document.querySelector('.repository.wiki.new .ui.form');
  const editorContainer = form.querySelector('.combo-markdown-editor');
  let editor;

  let renderRequesting = false;
  let lastContent;
  const renderEasyMDEPreview = async function () {
    if (renderRequesting) return;

    const previewFull = editorContainer.querySelector('.EasyMDEContainer .editor-preview-active');
    const previewSide = editorContainer.querySelector('.EasyMDEContainer .editor-preview-active-side');
    const previewTarget = previewSide || previewFull;
    const newContent = editArea.value;
    if (editor && previewTarget && lastContent !== newContent) {
      renderRequesting = true;
      const formData = new FormData();
      formData.append('mode', editor.previewMode);
      formData.append('context', editor.previewContext);
      formData.append('text', newContent);
      formData.append('wiki', editor.previewWiki);
      try {
        const response = await POST(editor.previewUrl, {data: formData});
        const data = await response.text();
        lastContent = newContent;
        previewTarget.innerHTML = `<div class="markup ui segment">${data}</div>`;
        initMarkupContent();
      } catch (error) {
        console.error('Error rendering preview:', error);
      } finally {
        renderRequesting = false;
        setTimeout(renderEasyMDEPreview, 1000);
      }
    } else {
      setTimeout(renderEasyMDEPreview, 1000);
    }
  };
  renderEasyMDEPreview();

  editor = await initComboMarkdownEditor(editorContainer, {
    useScene: 'wiki',
    // EasyMDE has some problems of height definition, it has inline style height 300px by default, so we also use inline styles to override it.
    // And another benefit is that we only need to write the style once for both editors.
    // TODO: Move height style to CSS after EasyMDE removal.
    editorHeights: {minHeight: '300px', height: 'calc(100vh - 600px)'},
    previewWiki: true,
    easyMDEOptions: {
      previewRender: (_content, previewTarget) => previewTarget.innerHTML, // disable builtin preview render
      toolbar: ['bold', 'italic', 'strikethrough', '|',
        'heading-1', 'heading-2', 'heading-3', 'heading-bigger', 'heading-smaller', '|',
        'gitea-code-inline', 'code', 'quote', '|', 'gitea-checkbox-empty', 'gitea-checkbox-checked', '|',
        'unordered-list', 'ordered-list', '|',
        'link', 'image', 'table', 'horizontal-rule', '|',
        'preview', 'fullscreen', 'side-by-side', '|', 'gitea-switch-to-textarea',
      ],
    },
  });

  form.addEventListener('submit', (e) => {
    if (!validateTextareaNonEmpty(editArea)) {
      e.preventDefault();
      e.stopPropagation();
    }
  });
}

function collapseWikiTocForMobile(collapse) {
  if (collapse) {
    document.querySelector('.wiki-content-toc details')?.removeAttribute('open');
  }
}

export function initRepoWikiForm() {
  if (!document.querySelector('.page-content.repository.wiki')) return;

  fomanticMobileScreen.addEventListener('change', (e) => collapseWikiTocForMobile(e.matches));
  collapseWikiTocForMobile(fomanticMobileScreen.matches);

  initRepoWikiFormEditor();
}
