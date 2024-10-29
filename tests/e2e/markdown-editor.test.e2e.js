// @ts-check

// @watch start
// web_src/js/features/comp/ComboMarkdownEditor.js
// web_src/css/editor/combomarkdowneditor.css
// templates/shared/combomarkdowneditor.tmpl
// @watch end

import {expect} from '@playwright/test';
import {test, load_logged_in_context, login_user} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('markdown indentation', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');

  const initText = `* first\n* second\n* third\n* last`;

  const page = await context.newPage();
  const response = await page.goto('/user2/repo1/issues/new');
  await expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const tab = '    ';
  const indent = page.locator('button[data-md-action="indent"]');
  const unindent = page.locator('button[data-md-action="unindent"]');
  await textarea.fill(initText);
  await textarea.click(); // Tab handling is disabled until pointer event or input.

  // Indent, then unindent first line
  await textarea.focus();
  await textarea.evaluate((it) => it.setSelectionRange(0, 0));
  await indent.click();
  await expect(textarea).toHaveValue(`${tab}* first\n* second\n* third\n* last`);
  await unindent.click();
  await expect(textarea).toHaveValue(initText);

  // Indent second line while somewhere inside of it
  await textarea.focus();
  await textarea.press('ArrowDown');
  await textarea.press('ArrowRight');
  await textarea.press('ArrowRight');
  await indent.click();
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n* third\n* last`);

  // Subsequently, select a chunk of 2nd and 3rd line and indent both, preserving the cursor position in relation to text
  await textarea.focus();
  await textarea.evaluate((it) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('hird')));
  await indent.click();
  const lines23 = `* first\n${tab}${tab}* second\n${tab}* third\n* last`;
  await expect(textarea).toHaveValue(lines23);
  await expect(textarea).toHaveJSProperty('selectionStart', lines23.indexOf('cond'));
  await expect(textarea).toHaveJSProperty('selectionEnd', lines23.indexOf('hird'));

  // Then unindent twice, erasing all indents.
  await unindent.click();
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n* third\n* last`);
  await unindent.click();
  await expect(textarea).toHaveValue(initText);

  // Indent and unindent with cursor at the end of the line
  await textarea.focus();
  await textarea.evaluate((it) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('cond')));
  await textarea.press('End');
  await indent.click();
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n* third\n* last`);
  await unindent.click();
  await expect(textarea).toHaveValue(initText);

  // Check that Tab does work after input
  await textarea.focus();
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Shift+Enter'); // Avoid triggering the prefix continuation feature
  await textarea.pressSequentially('* least');
  await indent.click();
  await expect(textarea).toHaveValue(`* first\n* second\n* third\n* last\n${tab}* least`);

  // Check that partial indents are cleared
  await textarea.focus();
  await textarea.fill(initText);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.indexOf('* second'), it.value.indexOf('* second')));
  await textarea.pressSequentially('  ');
  await unindent.click();
  await expect(textarea).toHaveValue(initText);
});

test('markdown list continuation', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');

  const initText = `* first\n* second\n* third\n* last`;

  const page = await context.newPage();
  const response = await page.goto('/user2/repo1/issues/new');
  await expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const tab = '    ';
  const indent = page.locator('button[data-md-action="indent"]');
  await textarea.fill(initText);

  // Test continuation of '* ' prefix
  await textarea.evaluate((it) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('cond')));
  await textarea.press('End');
  await textarea.press('Enter');
  await textarea.pressSequentially('middle');
  await expect(textarea).toHaveValue(`* first\n* second\n* middle\n* third\n* last`);

  // Test continuation of '    * ' prefix
  await indent.click();
  await textarea.press('Enter');
  await textarea.pressSequentially('muddle');
  await expect(textarea).toHaveValue(`* first\n* second\n${tab}* middle\n${tab}* muddle\n* third\n* last`);

  // Test breaking in the middle of a line
  await textarea.evaluate((it) => it.setSelectionRange(it.value.lastIndexOf('ddle'), it.value.lastIndexOf('ddle')));
  await textarea.pressSequentially('tate');
  await textarea.press('Enter');
  await textarea.pressSequentially('me');
  await expect(textarea).toHaveValue(`* first\n* second\n${tab}* middle\n${tab}* mutate\n${tab}* meddle\n* third\n* last`);

  // Test not triggering when Shift held
  await textarea.fill(initText);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Shift+Enter');
  await textarea.press('Enter');
  await textarea.pressSequentially('...but not least');
  await expect(textarea).toHaveValue(`* first\n* second\n* third\n* last\n\n...but not least`);

  // Test continuation of ordered list
  await textarea.fill(`1. one\n2. two`);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially('three');
  await expect(textarea).toHaveValue(`1. one\n2. two\n3. three`);

  // Test continuation of alternative ordered list syntax
  await textarea.fill(`1) one\n2) two`);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially('three');
  await expect(textarea).toHaveValue(`1) one\n2) two\n3) three`);

  // Test continuation of blockquote
  await textarea.fill(`> knowledge is power`);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially('france is bacon');
  await expect(textarea).toHaveValue(`> knowledge is power\n> france is bacon`);

  // Test continuation of checklists
  await textarea.fill(`- [ ] have a problem\n- [x] create a solution`);
  await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially('write a test');
  await expect(textarea).toHaveValue(`- [ ] have a problem\n- [x] create a solution\n- [ ] write a test`);

  // Test all conceivable syntax (except ordered lists)
  const prefixes = [
    '- ', // A space between the bullet and the content is required.
    ' - ', // I have seen single space in front of -/* being used and even recommended, I think.
    '* ',
    '+ ',
    '  ',
    '    ',
    '    - ',
    '\t',
    '\t\t* ',
    '> ',
    '> > ',
    '- [ ] ',
    '- [ ]', // This does seem to render, so allow.
    '* [ ] ',
    '+ [ ] ',
  ];
  for (const prefix of prefixes) {
    await textarea.fill(`${prefix}one`);
    await textarea.evaluate((it) => it.setSelectionRange(it.value.length, it.value.length));
    await textarea.press('Enter');
    await textarea.pressSequentially('two');
    await expect(textarea).toHaveValue(`${prefix}one\n${prefix}two`);
  }
});

test('markdown insert table', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');

  const page = await context.newPage();
  const response = await page.goto('/user2/repo1/issues/new');
  await expect(response?.status()).toBe(200);

  const newTableButton = page.locator('button[data-md-action="new-table"]');
  await newTableButton.click();

  const newTableModal = page.locator('div[data-markdown-table-modal-id="0"]');
  await expect(newTableModal).toBeVisible();

  await newTableModal.locator('input[name="table-rows"]').fill('3');
  await newTableModal.locator('input[name="table-columns"]').fill('2');

  await newTableModal.locator('button[data-selector-name="ok-button"]').click();

  await expect(newTableModal).toBeHidden();

  const textarea = page.locator('textarea[name=content]');
  await expect(textarea).toHaveValue('| Header  | Header  |\n|---------|---------|\n| Content | Content |\n| Content | Content |\n| Content | Content |\n');
});
