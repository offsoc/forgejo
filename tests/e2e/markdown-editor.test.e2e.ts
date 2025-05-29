// @watch start
// web_src/js/features/comp/ComboMarkdownEditor.js
// web_src/css/editor/combomarkdowneditor.css
// templates/shared/combomarkdowneditor.tmpl
// @watch end

import {expect} from '@playwright/test';
import {accessibilityCheck} from './shared/accessibility.ts';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Markdown image preview behaviour', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari;');

  // Editing the root README.md file for image preview
  const editPath = '/user2/repo1/src/branch/master/README.md';

  const response = await page.goto(editPath, {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  // Click 'Edit file' tab
  await page.locator('[data-tooltip-content="Edit file"]').click();

  // This yields the monaco editor
  const editor = page.getByRole('presentation').nth(0);
  await editor.click();
  // Clear all the content
  await page.keyboard.press('ControlOrMeta+KeyA');
  // Add the image
  await page.keyboard.type('![Logo of Forgejo](./assets/logo.svg "Logo of Forgejo")');

  // Click 'Preview' tab
  await page.locator('a[data-tab="preview"]').click();

  // Check for the image preview via the expected attribute
  const preview = page.locator('div[data-tab="preview"] p[dir="auto"] a');
  await expect(preview).toHaveAttribute('href', 'http://localhost:3003/user2/repo1/media/branch/master/assets/logo.svg');
  await save_visual(page);
});

test('Markdown indentation via toolbar', async ({page}) => {
  const initText = `* first\n* second\n* third\n* last`;

  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const tab = '    ';
  const indent = page.locator('button[data-md-action="indent"]');
  const unindent = page.locator('button[data-md-action="unindent"]');
  await textarea.fill(initText);

  // Indent, then unindent first line
  await textarea.focus();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(0, 0));
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
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('hird')));
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
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('cond')));
  await textarea.press('End');
  await indent.click();
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n* third\n* last`);
  await unindent.click();
  await expect(textarea).toHaveValue(initText);

  // Check that Tab does work after input
  await textarea.focus();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Shift+Enter'); // Avoid triggering the prefix continuation feature
  await textarea.pressSequentially('* least');
  await indent.click();
  await expect(textarea).toHaveValue(`* first\n* second\n* third\n* last\n${tab}* least`);

  // Check that partial indents are cleared
  await textarea.focus();
  await textarea.fill(initText);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('* second'), it.value.indexOf('* second')));
  await textarea.pressSequentially('  ');
  await unindent.click();
  await expect(textarea).toHaveValue(initText);
});

test('markdown indentation with Tab', async ({page}) => {
  const initText = `* first\n* second\n* third\n* last`;

  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const toast = page.locator('.toastify');
  const tab = '    ';

  await textarea.fill(initText);

  await textarea.click(); // Tab handling is disabled until pointer event or input.

  // Indent, then unindent first line
  await textarea.focus();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(0, 0));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`${tab}* first\n* second\n* third\n* last`);
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(initText);

  // Attempt unindent again, ensure focus is not immediately lost and toast is shown, but then focus is lost on next attempt.
  await expect(toast).toBeHidden(); // toast should not already be there
  await textarea.press('Shift+Tab');
  await expect(textarea).toBeFocused();
  await expect(toast).toBeVisible();
  await textarea.press('Shift+Tab');
  await expect(textarea).not.toBeFocused();

  // Indent lines 2-4
  await textarea.click();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('\n') + 1, it.value.length));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n${tab}* third\n${tab}* last`);

  // Indent second line while in whitespace, then unindent.
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf(' * third'), it.value.indexOf(' * third')));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n${tab}${tab}* third\n${tab}* last`);
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(`* first\n${tab}* second\n${tab}* third\n${tab}* last`);

  // Select all and unindent, then lose focus.
  await textarea.evaluate((it:HTMLTextAreaElement) => it.select());
  await textarea.press('Shift+Tab'); // Everything is unindented.
  await expect(textarea).toHaveValue(initText);
  await textarea.press('Shift+Tab'); // Valid, but nothing happens -> switch to "about to lose focus" state.
  await expect(textarea).toBeFocused();
  await textarea.press('Shift+Tab');
  await expect(textarea).not.toBeFocused();

  // Attempt the same with cursor within list element body.
  await textarea.focus();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(0, 0));
  await textarea.press('ArrowRight');
  await textarea.press('ArrowRight');
  await textarea.press('Tab');
  // Whole line should be indented.
  await expect(textarea).toHaveValue(`${tab}* first\n* second\n* third\n* last`);
  await textarea.press('Shift+Tab');

  // Subsequently, select a chunk of 2nd and 3rd line and indent both, preserving the cursor position in relation to text
  const line3 = `* first\n* second\n${tab}* third\n* last`;
  const lines23 = `* first\n${tab}* second\n${tab}${tab}* third\n* last`;
  await textarea.focus();
  await textarea.fill(line3);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('cond'), it.value.indexOf('hird')));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(lines23);
  await expect(textarea).toHaveJSProperty('selectionStart', lines23.indexOf('cond'));
  await expect(textarea).toHaveJSProperty('selectionEnd', lines23.indexOf('hird'));

  // Then unindent twice, erasing all indents.
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(line3);
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(initText);

  // Check that partial indents are cleared
  await textarea.focus();
  await textarea.fill(initText);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('* second'), it.value.indexOf('* second')));
  await textarea.pressSequentially('  ');
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(initText);
});

test('markdown block quote indentation', async ({page}) => {
  const initText = `> first\n> second\n> third\n> last`;

  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const toast = page.locator('.toastify');

  await textarea.fill(initText);

  await textarea.click(); // Tab handling is disabled until pointer event or input.

  // Indent, then unindent first line twice (quotes can quote quotes!)
  await textarea.focus();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(0, 0));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`> > first\n> second\n> third\n> last`);
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`> > > first\n> second\n> third\n> last`);
  await textarea.press('Shift+Tab');
  await textarea.press('Shift+Tab');
  await expect(textarea).toHaveValue(initText);

  // Attempt unindent again.
  await expect(toast).toBeHidden(); // toast should not already be there
  await textarea.press('Shift+Tab');
  // Nothing happens - quote should not stop being a quote
  await expect(textarea).toHaveValue(initText);
  // Focus is not immediately lost and toast is shown,
  await expect(textarea).toBeFocused();
  await expect(toast).toBeVisible();
  // Focus is lost on next attempt,
  await textarea.press('Shift+Tab');
  await expect(textarea).not.toBeFocused();

  // Indent lines 2-4
  await textarea.click();
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('\n') + 1, it.value.length));
  await textarea.press('Tab');
  await expect(textarea).toHaveValue(`> first\n> > second\n> > third\n> > last`);

  // Select all and unindent, then lose focus.
  await textarea.evaluate((it:HTMLTextAreaElement) => it.select());
  await textarea.press('Shift+Tab'); // Everything is unindented.
  await expect(textarea).toHaveValue(initText);
  await textarea.press('Shift+Tab'); // Valid, but nothing happens -> switch to "about to lose focus" state.
  await expect(textarea).toBeFocused();
  await textarea.press('Shift+Tab');
  await expect(textarea).not.toBeFocused();
});

test('Markdown list continuation', async ({page}) => {
  const initText = `* first\n* second`;

  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const tab = '    ';
  const indent = page.locator('button[data-md-action="indent"]');
  await textarea.fill(initText);

  // Test continuation of '    * ' prefix
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('rst'), it.value.indexOf('rst')));
  await indent.click();
  await textarea.press('End');
  await textarea.press('Enter');
  await textarea.pressSequentially('muddle');
  await expect(textarea).toHaveValue(`${tab}* first\n${tab}* muddle\n* second`);

  // Test breaking in the middle of a line
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.lastIndexOf('ddle'), it.value.lastIndexOf('ddle')));
  await textarea.pressSequentially('tate');
  await textarea.press('Enter');
  await textarea.pressSequentially('me');
  await expect(textarea).toHaveValue(`${tab}* first\n${tab}* mutate\n${tab}* meddle\n* second`);

  // Test not triggering when Shift held
  await textarea.fill(initText);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Shift+Enter');
  await textarea.press('Enter');
  await textarea.pressSequentially('...but not least');
  await expect(textarea).toHaveValue(`* first\n* second\n\n...but not least`);

  // Test continuation of ordered list
  await textarea.fill(`1. one`);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially(' ');
  await textarea.press('Enter');
  await textarea.pressSequentially('three');
  await textarea.press('Enter');
  await textarea.press('Enter');
  await expect(textarea).toHaveValue(`1. one\n2.  \n3. three\n\n`);

  // Test continuation of alternative ordered list syntax
  await textarea.fill(`1) one`);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially(' ');
  await textarea.press('Enter');
  await textarea.pressSequentially('three');
  await textarea.press('Enter');
  await textarea.press('Enter');
  await expect(textarea).toHaveValue(`1) one\n2)  \n3) three\n\n`);

  // Test continuation of checklists
  await textarea.fill(`- [ ]have a problem\n- [x]create a solution`);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
  await textarea.press('Enter');
  await textarea.pressSequentially('write a test');
  await expect(textarea).toHaveValue(`- [ ]have a problem\n- [x]create a solution\n- [ ]write a test`);

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
    '* [ ] ',
    '+ [ ] ',
  ];
  for (const prefix of prefixes) {
    await textarea.fill(`${prefix}one`);
    await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.length, it.value.length));
    await textarea.press('Enter');
    await textarea.pressSequentially(' ');
    await textarea.press('Enter');
    await textarea.pressSequentially('two');
    await textarea.press('Enter');
    await textarea.press('Enter');
    await expect(textarea).toHaveValue(`${prefix}one\n${prefix} \n${prefix}two\n\n`);
  }
});

test('Markdown insert table', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const newTableButton = page.locator('button[data-md-action="new-table"]');
  await newTableButton.click();

  const newTableModal = page.locator('div[data-markdown-table-modal-id="0"]');
  await expect(newTableModal).toBeVisible();
  await save_visual(page);

  await newTableModal.locator('input[name="table-rows"]').fill('3');
  await newTableModal.locator('input[name="table-columns"]').fill('2');

  await newTableModal.locator('button[data-selector-name="ok-button"]').click();

  await expect(newTableModal).toBeHidden();

  const textarea = page.locator('textarea[name=content]');
  await expect(textarea).toHaveValue('| Header  | Header  |\n|---------|---------|\n| Content | Content |\n| Content | Content |\n| Content | Content |\n');
  await save_visual(page);
});

test('Markdown insert link', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const newLinkButton = page.locator('button[data-md-action="new-link"]');
  await newLinkButton.click();

  const newLinkModal = page.locator('div[data-markdown-link-modal-id="0"]');
  await expect(newLinkModal).toBeVisible();
  await accessibilityCheck({page}, ['[data-modal-name="new-markdown-link"]'], [], []);
  await save_visual(page);

  const url = 'https://example.com';
  const description = 'Where does this lead?';

  await newLinkModal.locator('input[name="link-url"]').fill(url);
  await newLinkModal.locator('input[name="link-description"]').fill(description);

  await newLinkModal.locator('button[data-selector-name="ok-button"]').click();

  await expect(newLinkModal).toBeHidden();

  const textarea = page.locator('textarea[name=content]');
  await expect(textarea).toHaveValue(`[${description}](${url})`);
  await save_visual(page);
});

test('text expander has higher prio then prefix continuation', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const textarea = page.locator('textarea[name=content]');
  const initText = `* first`;
  await textarea.fill(initText);
  await textarea.evaluate((it:HTMLTextAreaElement) => it.setSelectionRange(it.value.indexOf('rst'), it.value.indexOf('rst')));
  await textarea.press('End');

  // Test emoji completion
  await textarea.press('Enter');
  await textarea.pressSequentially(':smile_c');
  await textarea.press('Enter');
  await expect(textarea).toHaveValue(`* first\n* 😸`);

  // Test username completion
  await textarea.press('Enter');
  await textarea.pressSequentially('@user');
  await textarea.press('Enter');
  await expect(textarea).toHaveValue(`* first\n* 😸\n* @user2 `);

  await textarea.press('Enter');
  await expect(textarea).toHaveValue(`* first\n* 😸\n* @user2 \n* `);
});

test('Combo Markdown: preview mode switch', async ({page}) => {
  // Load page with editor
  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const toolbarItem = page.locator('md-header');
  const editorPanel = page.locator('[data-tab-panel="markdown-writer"]');
  const previewPanel = page.locator('[data-tab-panel="markdown-previewer"]');

  // Verify correct visibility of related UI elements
  await expect(toolbarItem).toBeVisible();
  await expect(editorPanel).toBeVisible();
  await expect(previewPanel).toBeHidden();

  // Fill some content
  const textarea = page.locator('textarea.markdown-text-editor');
  await textarea.fill('**Content** :100: _100_');

  // Switch to preview mode
  await page.locator('a[data-tab-for="markdown-previewer"]').click();

  // Verify that the related UI elements were switched correctly
  await expect(toolbarItem).toBeHidden();
  await expect(editorPanel).toBeHidden();
  await expect(previewPanel).toBeVisible();
  await save_visual(page);

  // Verify that some content rendered
  await expect(page.locator('[data-tab-panel="markdown-previewer"] .emoji[data-alias="100"]')).toBeVisible();

  // Switch back to edit mode
  await page.locator('a[data-tab-for="markdown-writer"]').click();

  // Verify that the related UI elements were switched back correctly
  await expect(toolbarItem).toBeVisible();
  await expect(editorPanel).toBeVisible();
  await expect(previewPanel).toBeHidden();
  await save_visual(page);
});
