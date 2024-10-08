import {POST} from '../../modules/fetch.js';
import {toggleElem} from '../../utils/dom.js';

export function initCompWebHookEditor() {
  if (!document.querySelectorAll('.new.webhook').length) {
    return;
  }

  // some webhooks (like Gitea) allow to set the request method (GET/POST), and it would toggle the "Content Type" field
  const httpMethodInput = document.getElementById('http_method');
  if (httpMethodInput) {
    const updateContentType = function () {
      const visible = httpMethodInput.value === 'POST';
      toggleElem(document.getElementById('content_type').closest('.field'), visible);
    };
    updateContentType();
    httpMethodInput.addEventListener('change', updateContentType);
  }

  // Test delivery
  document.getElementById('test-delivery')?.addEventListener('click', async function () {
    this.classList.add('is-loading', 'disabled');
    await POST(this.getAttribute('data-link'));
    setTimeout(() => {
      window.location.href = this.getAttribute('data-redirect');
    }, 5000);
  });
}
