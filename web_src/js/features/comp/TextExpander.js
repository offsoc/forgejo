import {matchEmoji, matchMention, matchIssue} from '../../utils/match.js';
import {emojiString} from '../emoji.js';
import {getIssueIcon, getIssueColor,isIssueSuggestionsLoaded, fetchIssueSuggestions} from '../issue.js'
import {svg} from '../../svg.js'
import {createElementFromHTML} from '../../utils/dom.js';
import { GET } from '../../modules/fetch.js';

async function issueSuggestions(text) {
  const key = '#';

  const matches = matchIssue(text);
  if (!matches.length) return {matched: false};

  const ul = document.createElement('ul');
  ul.classList.add('suggestions');
  for (const issue of matches) {
    const li = document.createElement('li');
    li.setAttribute('role', 'option');
    li.setAttribute('data-value', `${key}${issue.number}`);
    li.classList.add('tw-flex', 'tw-gap-2')

    const icon = svg(getIssueIcon(issue), 16, ['text', getIssueColor(issue)].join(' '));
    li.append(createElementFromHTML(icon));

    const id = document.createElement('span');
    id.textContent = issue.number.toString();
    li.append(id);

    const nameSpan = document.createElement('span');
    nameSpan.textContent = issue.title;
    li.append(nameSpan);

    ul.append(li);
  }

  return {matched: true, fragment: ul};
}

export function initTextExpander(expander) {
  if (!expander) return;

  const textarea = expander.querySelector('textarea');

  expander?.addEventListener('text-expander-change', ({detail: {key, provide, text}}) => {
    if (key === ':') {
      const matches = matchEmoji(text);
      if (!matches.length) return provide({matched: false});

      const ul = document.createElement('ul');
      ul.classList.add('suggestions');
      for (const name of matches) {
        const emoji = emojiString(name);
        const li = document.createElement('li');
        li.setAttribute('role', 'option');
        li.setAttribute('data-value', emoji);
        li.textContent = `${emoji} ${name}`;
        ul.append(li);
      }

      provide({matched: true, fragment: ul});
    } else if (key === '@') {
      const matches = matchMention(text);
      if (!matches.length) return provide({matched: false});

      const ul = document.createElement('ul');
      ul.classList.add('suggestions');
      for (const {value, name, fullname, avatar} of matches) {
        const li = document.createElement('li');
        li.setAttribute('role', 'option');
        li.setAttribute('data-value', `${key}${value}`);

        const img = document.createElement('img');
        img.src = avatar;
        li.append(img);

        const nameSpan = document.createElement('span');
        nameSpan.textContent = name;
        li.append(nameSpan);

        if (fullname && fullname.toLowerCase() !== name) {
          const fullnameSpan = document.createElement('span');
          fullnameSpan.classList.add('fullname');
          fullnameSpan.textContent = fullname;
          li.append(fullnameSpan);
        }

        ul.append(li);
      }

      provide({matched: true, fragment: ul});
    } else if (key === '#') {
      if (!isIssueSuggestionsLoaded()) {
        provide(fetchIssueSuggestions().then(() => issueSuggestions(text)));
      } else {
        provide(issueSuggestions(text));
      }
    }
  });
  expander?.addEventListener('text-expander-value', ({detail}) => {
    if (detail?.item) {
      // add a space after @mentions and #issue as it's likely the user wants one
      const suffix = ['@', '#'].includes(detail.key) ? ' ' : '';
      detail.value = `${detail.item.getAttribute('data-value')}${suffix}`;
    }
  });
}
