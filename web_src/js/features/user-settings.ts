import {hideElem, showElem} from '../utils/dom.ts';

function onPronounsDropdownUpdate() {
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsDropdown = document.getElementById('pronouns-dropdown');
  const pronounsInput = pronounsDropdown.querySelector('input');
  // must be kept in sync with `routers/web/user/setting/profile.go`
  const isCustom = !(
    pronounsInput.value === '' ||
    pronounsInput.value === 'he/him' ||
    pronounsInput.value === 'she/her' ||
    pronounsInput.value === 'they/them' ||
    pronounsInput.value === 'it/its' ||
    pronounsInput.value === 'any pronouns'
  );
  if (isCustom) {
    if (pronounsInput.value === '!') {
      pronounsCustom.value = '';
    } else {
      pronounsCustom.value = pronounsInput.value;
    }
    pronounsCustom.style.display = '';
  } else {
    pronounsCustom.style.display = 'none';
  }
}
function onPronounsCustomUpdate() {
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsInput = document.querySelector('#pronouns-dropdown input');
  pronounsInput.value = pronounsCustom.value;
}

export function initUserSettings() {
  if (!document.querySelectorAll('.user.settings.profile').length) return;

  const usernameInput = document.getElementById('username');
  if (!usernameInput) return;
  usernameInput.addEventListener('input', function () {
    const prompt = document.getElementById('name-change-prompt');
    const promptRedirect = document.getElementById('name-change-redirect-prompt');
    if (this.value.toLowerCase() !== this.getAttribute('data-name').toLowerCase()) {
      showElem(prompt);
      showElem(promptRedirect);
    } else {
      hideElem(prompt);
      hideElem(promptRedirect);
    }
  });

  const pronounsDropdown = document.getElementById('pronouns-dropdown');
  const pronounsCustom = document.getElementById('pronouns-custom');
  const pronounsInput = pronounsDropdown.querySelector('input');

  // If JS is disabled, the page will show the custom input, as the dropdown requires JS to work.
  // JS progressively enhances the input by adding a dropdown, but it works regardless.
  pronounsCustom.removeAttribute('name');
  pronounsInput.setAttribute('name', 'pronouns');
  pronounsDropdown.style.display = '';

  onPronounsDropdownUpdate();
  pronounsInput.addEventListener('change', onPronounsDropdownUpdate);
  pronounsCustom.addEventListener('input', onPronounsCustomUpdate);
}
