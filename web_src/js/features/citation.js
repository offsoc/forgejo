import $ from 'jquery';
import {getCurrentLocale} from '../utils.js';

const {pageData} = window.config;

async function initInputCitationValue(inputContent) {
  const [{Cite, plugins}] = await Promise.all([
    import(/* webpackChunkName: "citation-js-core" */'@citation-js/core'),
    import(/* webpackChunkName: "citation-js-formats" */'@citation-js/plugin-software-formats'),
    import(/* webpackChunkName: "citation-js-bibtex" */'@citation-js/plugin-bibtex'),
  ]);
  const {citationFileContent} = pageData;
  const config = plugins.config.get('@bibtex');
  config.constants.fieldTypes.doi = ['field', 'literal'];
  config.constants.fieldTypes.version = ['field', 'literal'];
  const citationFormatter = new Cite(citationFileContent);
  const lang = getCurrentLocale() || 'en-US';
  const bibtexOutput = citationFormatter.format('bibtex', {lang});
  inputContent.value = bibtexOutput;
}

export async function initCitationFileCopyContent() {
  if (!pageData.citationFileContent) return;

  const inputContent = document.getElementById('citation-copy-content');

  if (!inputContent) return;

  document.getElementById('cite-repo-button')?.addEventListener('click', async (e) => {
    const dropdownBtn = e.target.closest('.ui.dropdown.button');
    dropdownBtn.classList.add('is-loading');

    try {
      try {
        await initInputCitationValue(inputContent);
      } catch (e) {
        console.error(`initCitationFileCopyContent error: ${e}`, e);
        return;
      }

      inputContent.addEventListener('click', () => {
        inputContent.select();
      });
    } finally {
      dropdownBtn.classList.remove('is-loading');
    }

    $('#cite-repo-modal').modal('show');
  });
}
