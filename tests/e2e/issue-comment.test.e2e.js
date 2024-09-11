// @ts-check
import {expect, test} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Hyperlink paste behaviour', async ({browser}, workerInfo) => {
  test.skip(['Mobile Safari', 'Mobile Chrome', 'webkit'].includes(workerInfo.project.name), 'Mobile clients seem to have very weird behaviour with this test, which I cannot confirm with real usage');
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();
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
});

test('Always focus edit tab first on edit', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();
  const response = await page.goto('/user2/repo1/issues/1');
  await expect(response?.status()).toBe(200);

  // Switch to preview tab and save
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  await page.locator('#issue-1 .comment-container a[data-tab-for=markdown-previewer]').click();
  await page.click('#issue-1 .comment-container .save');

  await page.waitForLoadState('networkidle');

  // Edit again and assert that edit tab should be active (and not preview tab)
  await page.click('#issue-1 .comment-container .context-menu');
  await page.click('#issue-1 .comment-container .menu>.edit-content');
  const editTab = page.locator('#issue-1 .comment-container a[data-tab-for=markdown-writer]');
  const previewTab = page.locator('#issue-1 .comment-container a[data-tab-for=markdown-previewer]');

  await expect(editTab).toHaveClass(/active/);
  await expect(previewTab).not.toHaveClass(/active/);
});
