// @ts-check
import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
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
