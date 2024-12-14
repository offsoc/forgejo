import {svg} from '../svg.js';

const markdownIDPrefix = '~';

export function initMarkupAnchors() {
  compatHashOldFormat();
  globalThis.addEventListener('hashchange', () => compatHashOldFormat());

  const markupEls = document.querySelectorAll('.markup');
  if (!markupEls.length) return;

  for (const markupEl of markupEls) {
    // create link icons for markup headings
    // TODO create this on the server too
    for (const heading of markupEl.querySelectorAll('h1, h2, h3, h4, h5, h6')) {
      const a = document.createElement('a');
      a.classList.add('anchor');
      a.setAttribute('href', `#${encodeURIComponent(heading.id)}`);
      a.innerHTML = svg('octicon-link');
      heading.prepend(a);
    }
  }
}

function compatHashOldFormat() {
  // convert from old user-content- prefix format
  if (globalThis.location.hash.startsWith('#user-content-')) {
    globalThis.location.hash = `#${markdownIDPrefix}${globalThis.location.hash.slice('#user-content-'.length)}`;
  }

  // this format is ambiguous and so it is only converted when the markdown element exists but the exact id does not.
  if (globalThis.location.hash.startsWith('#')) {
    const id = globalThis.location.hash.slice('#'.length);
    if (document.getElementById(`${markdownIDPrefix}${id}`) && !document.getElementById(id)) {
      globalThis.location.hash = `#${markdownIDPrefix}${id}`;
    }
  }
}
