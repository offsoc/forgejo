import {vi} from 'vitest';

import {issueTitleHTML} from './repo-issue.js';

// monaco-editor does not have any exports fields, which trips up vitest
vi.mock('./comp/ComboMarkdownEditor.js', () => ({}));
// jQuery is missing
vi.mock('./common-global.js', () => ({}));

test('Convert issue title to html', () => {
  expect(issueTitleHTML('')).toEqual('');
  expect(issueTitleHTML('issue title')).toEqual('issue title');

  const expected_thumbs_up = `<span class="emoji" title=":+1:">👍</span>`;
  expect(issueTitleHTML(':+1:')).toEqual(expected_thumbs_up);
  expect(issueTitleHTML(':invalid emoji:')).toEqual(':invalid emoji:');

  const expected_code_block = `<code class="inline-code-block">code</code>`;
  expect(issueTitleHTML('`code`')).toEqual(expected_code_block);
  expect(issueTitleHTML('`invalid code')).toEqual('`invalid code');
  expect(issueTitleHTML('invalid code`')).toEqual('invalid code`');

  expect(issueTitleHTML('issue title :+1: `code`')).toEqual(`issue title ${expected_thumbs_up} ${expected_code_block}`);
});
