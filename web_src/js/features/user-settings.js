import {hideElem, showElem} from '../utils/dom.js';

function onPronounsDropdownUpdate() {
  const pronounsCustom = document.getElementById('label-pronouns-custom');
  const pronounsCustomInput = pronounsCustom.querySelector('input');
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
      pronounsCustomInput.value = '';
    } else {
      pronounsCustomInput.value = pronounsInput.value;
    }
    showElem(pronounsCustom);
  } else {
    hideElem(pronounsCustom);
  }
}
function onPronounsCustomUpdate() {
  const pronounsCustomInput = document.querySelector('#label-pronouns-custom input');
  const pronounsInput = document.querySelector('#pronouns-dropdown input');
  pronounsInput.value = pronounsCustomInput.value;
}

export function initUserSettings() {
  if (!document.querySelectorAll('.user.settings.profile').length) return;

  const pronounsDropdown = document.getElementById('label-pronouns');
  const pronounsCustomInput = document.querySelector('#label-pronouns-custom input');
  const pronounsInput = pronounsDropdown.querySelector('input');

  // If JS is disabled, the page will show the custom input, as the dropdown requires JS to work.
  // JS progressively enhances the input by adding a dropdown, but it works regardless.
  pronounsCustomInput.removeAttribute('name');
  pronounsInput.setAttribute('name', 'pronouns');
  showElem(pronounsDropdown);

  onPronounsDropdownUpdate();
  pronounsInput.addEventListener('change', onPronounsDropdownUpdate);
  pronounsCustomInput.addEventListener('input', onPronounsCustomUpdate);
}
