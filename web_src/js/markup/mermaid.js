import {isDarkTheme} from '../utils.js';
const {mermaidMaxSourceCharacters} = window.config;

const iframeStyle = `
  body {margin: 0; padding: 0}
  .mermaid-chart {display: block; margin: 0 auto}
`;

function displayError(el, err) {
  el.closest('pre').classList.remove('is-loading');
  const errorNode = document.createElement('div');
  errorNode.setAttribute('class', 'ui message error markup-block-error mono');
  errorNode.textContent = err.str || err.message || String(err);
  el.closest('pre').before(errorNode);
}

export async function renderMermaid() {
  const els = document.querySelectorAll('.markup code.language-mermaid');
  if (!els.length) return;

  const {default: mermaid} = await import(/* webpackChunkName: "mermaid" */'mermaid');

  mermaid.initialize({
    startOnLoad: false,
    theme: isDarkTheme() ? 'dark' : 'neutral',
    securityLevel: 'strict',
  });

  for (const el of els) {
    if (mermaidMaxSourceCharacters >= 0 && el.textContent.length > mermaidMaxSourceCharacters) {
      displayError(el, new Error(`Mermaid source of ${el.textContent.length} characters exceeds the maximum allowed length of ${mermaidMaxSourceCharacters}.`));
      continue;
    }

    let valid;
    try {
      valid = mermaid.parse(el.textContent);
    } catch (err) {
      displayError(el, err);
    }

    if (!valid) {
      el.closest('pre').classList.remove('is-loading');
      continue;
    }

    try {
      mermaid.init(undefined, el, (id) => {
        const svg = document.getElementById(id);
        svg.classList.add('mermaid-chart');
        const iframe = document.createElement('iframe');
        iframe.classList.add('markup-render');
        iframe.sandbox = 'allow-scripts';
        iframe.scrolling = 'no';
        iframe.loading = 'lazy';
        iframe.style.height = `${Math.ceil(svg.getAttribute('height'))}px`;
        iframe.srcdoc = `<html><head><style>${iframeStyle}</style></head><body>${svg.outerHTML}</body></html>`;
        svg.closest('pre').replaceWith(iframe);
      });
    } catch (err) {
      displayError(el, err);
    }
  }
}
