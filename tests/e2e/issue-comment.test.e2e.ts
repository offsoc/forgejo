// @watch start
// web_src/js/features/comp/**
// web_src/js/features/repo-**
// templates/repo/issue/view_content/*
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Menu accessibility', async ({page}) => {
  await page.goto('/user2/repo1/issues/1');
  await expect(page.getByLabel('user2 reacted eyes. Remove eyes')).toBeVisible();
  await expect(page.getByLabel('reacted laugh. Remove laugh')).toBeVisible();
  await expect(page.locator('#issue-1').getByLabel('Comment menu')).toBeVisible();
  await expect(page.locator('#issue-1').getByRole('heading').getByLabel('Add reaction')).toBeVisible();
  page.getByLabel('reacted laugh. Remove').click();
  await expect(page.getByLabel('user1 reacted laugh. Add laugh')).toBeVisible();
  page.getByLabel('user1 reacted laugh.').click();
  await expect(page.getByLabel('user1, user2 reacted laugh. Remove laugh')).toBeVisible();
});

test('Hyperlink paste behaviour', async ({page}, workerInfo) => {
  test.skip(['Mobile Safari', 'Mobile Chrome', 'webkit'].includes(workerInfo.project.name), 'Mobile clients seem to have very weird behaviour with this test, which I cannot confirm with real usage');
  await page.goto('/user2/repo1/issues/new');
  await page.locator('textarea').click();
  // same URL
  await page.locator('textarea').fill('https://codeberg.org/forgejo/forgejo#some-anchor');
  await page.locator('textarea').press('Shift+Home');
  await page.locator('textarea').press('ControlOrMeta+c');
  await page.locator('textarea').press('ControlOrMeta+v');
  await expect(page.locator('textarea')).toHaveValue('https://codeberg.org/forgejo/forgejo#some-anchor');
  // other text
  await page.locator('textarea').fill('Some other text');
  await page.locator('textarea').press('ControlOrMeta+a');
  await page.locator('textarea').press('ControlOrMeta+v');
  await expect(page.locator('textarea')).toHaveValue('[Some other text](https://codeberg.org/forgejo/forgejo#some-anchor)');
  // subset of URL
  await page.locator('textarea').fill('https://codeberg.org/forgejo/forgejo#some');
  await page.locator('textarea').press('ControlOrMeta+a');
  await page.locator('textarea').press('ControlOrMeta+v');
  await expect(page.locator('textarea')).toHaveValue('https://codeberg.org/forgejo/forgejo#some-anchor');
  // superset of URL
  await page.locator('textarea').fill('https://codeberg.org/forgejo/forgejo#some-anchor-on-the-page');
  await page.locator('textarea').press('ControlOrMeta+a');
  await page.locator('textarea').press('ControlOrMeta+v');
  await expect(page.locator('textarea')).toHaveValue('https://codeberg.org/forgejo/forgejo#some-anchor');
  // completely separate URL
  await page.locator('textarea').fill('http://example.com');
  await page.locator('textarea').press('ControlOrMeta+a');
  await page.locator('textarea').press('ControlOrMeta+v');
  await expect(page.locator('textarea')).toHaveValue('https://codeberg.org/forgejo/forgejo#some-anchor');
  await page.locator('textarea').fill('');
});

test('Always focus edit tab first on edit', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  // Switch to preview tab and save
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  await page.locator('#issue-1 .comment-container a[data-tab-for=markdown-previewer]').click();
  await page.click('#issue-1 .comment-container .save');

  await page.waitForLoadState();

  // Edit again and assert that edit tab should be active (and not preview tab)
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  const editTab = page.locator('#issue-1 .comment-container a[data-tab-for=markdown-writer]');
  const previewTab = page.locator('#issue-1 .comment-container a[data-tab-for=markdown-previewer]');

  await expect(editTab).toHaveClass(/active/);
  await expect(previewTab).not.toHaveClass(/active/);
  await save_visual(page);
});

test('Reset content of comment edit field on cancel', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  const editorTextarea = page.locator('[id="_combo_markdown_editor_1"]');

  // Change the content of the edit field
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  await expect(editorTextarea).toHaveValue('content for the first issue');
  await editorTextarea.fill('some random string');
  await expect(editorTextarea).toHaveValue('some random string');
  await page.click('#issue-1 .comment-container .edit .cancel');

  // Edit again and assert that the edit field should be reset to the initial content
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  await expect(editorTextarea).toHaveValue('content for the first issue');
  await save_visual(page);
});

test('Quote reply', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name !== 'firefox', 'Uses Firefox specific selection quirks');
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  const editorTextarea = page.locator('textarea.markdown-text-editor');

  // Full quote.
  await page.click('#issuecomment-1001 .comment-container .context-menu');
  await page.click('#issuecomment-1001 .quote-reply');

  await expect(editorTextarea).toHaveValue('@user2 wrote in http://localhost:3003/user2/repo1/issues/1#issuecomment-1001:\n\n' +
                                           '> ## [](#lorem-ipsum)Lorem Ipsum\n' +
                                           '> \n' +
                                           '> I would like to say that **I am not appealed** that it took _so long_ for this `feature` to be [created](https://example.com) \\(e^{\\pi i} + 1 = 0\\)\n' +
                                           '> \n' +
                                           '> \\[e^{\\pi i} + 1 = 0\\]\n' +
                                           '> \n' +
                                           '> #1\n' +
                                           '> \n' +
                                           '> ```js\n' +
                                           "> console.log('evil')\n" +
                                           "> alert('evil')\n" +
                                           '> ```\n' +
                                           '> \n' +
                                           '> :+1: :100:\n\n');

  await editorTextarea.fill('');

  // Partial quote.
  await page.click('#issuecomment-1001 .comment-container .context-menu');

  await page.evaluate(() => {
    const range = new Range();
    range.setStart(document.querySelector('#issuecomment-1001-content #user-content-lorem-ipsum').childNodes[1], 6);
    range.setEnd(document.querySelector('#issuecomment-1001-content p').childNodes[1].childNodes[0], 7);

    const selection = window.getSelection();

    // Add range to window selection
    selection.addRange(range);
  });

  await page.click('#issuecomment-1001 .quote-reply');

  await expect(editorTextarea).toHaveValue('@user2 wrote in http://localhost:3003/user2/repo1/issues/1#issuecomment-1001:\n\n' +
                                           '> ## Ipsum\n' +
                                           '> \n' +
                                           '> I would like to say that **I am no**\n\n');

  await editorTextarea.fill('');

  // Another partial quote.
  await page.click('#issuecomment-1001 .comment-container .context-menu');

  await page.evaluate(() => {
    const range = new Range();
    range.setStart(document.querySelector('#issuecomment-1001-content p').childNodes[1].childNodes[0], 7);
    range.setEnd(document.querySelector('#issuecomment-1001-content p').childNodes[7].childNodes[0], 3);

    const selection = window.getSelection();

    // Add range to window selection
    selection.addRange(range);
  });

  await page.click('#issuecomment-1001 .quote-reply');

  await expect(editorTextarea).toHaveValue('@user2 wrote in http://localhost:3003/user2/repo1/issues/1#issuecomment-1001:\n\n' +
                                           '> **t appealed** that it took _so long_ for this `feature` to be [cre](https://example.com)\n\n');

  await editorTextarea.fill('');
});

test('Pull quote reply', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name !== 'firefox', 'Uses Firefox specific selection quirks');
  const response = await page.goto('/user2/commitsonpr/pulls/1/files');
  expect(response?.status()).toBe(200);

  const editorTextarea = page.locator('form.comment-form textarea.markdown-text-editor');

  // Full quote with no reply handler being open.
  await page.click('.comment-code-cloud .context-menu');
  await page.click('.comment-code-cloud .quote-reply');

  await expect(editorTextarea).toHaveValue('@user2 wrote in http://localhost:3003/user2/commitsonpr/pulls/1/files#issuecomment-1002:\n\n' +
                                           '> ## [](#lorem-ipsum)Lorem Ipsum\n' +
                                           '> \n' +
                                           '> I would like to say that **I am not appealed** that it took _so long_ for this `feature` to be [created](https://example.com) \\(e^{\\pi i} + 1 = 0\\)\n' +
                                           '> \n' +
                                           '> \\[e^{\\pi i} + 1 = 0\\]\n' +
                                           '> \n' +
                                           '> #1\n' +
                                           '> \n' +
                                           '> ```js\n' +
                                           "> console.log('evil')\n" +
                                           "> alert('evil')\n" +
                                           '> ```\n' +
                                           '> \n' +
                                           '> :+1: :100:\n\n');

  await editorTextarea.fill('');
});
