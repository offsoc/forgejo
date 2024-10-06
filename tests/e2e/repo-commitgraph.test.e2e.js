// @ts-check

// @watch start
// templates/repo/graph.tmpl
// web_src/css/features/gitgraph.css
// web_src/js/features/repo-graph.js
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Commit graph overflow', async ({page}) => {
  await page.goto('/user2/diff-test/graph');
  await expect(page.getByRole('button', {name: 'Mono'})).toBeInViewport({ratio: 1});
  await expect(page.getByRole('button', {name: 'Color'})).toBeInViewport({ratio: 1});
  await expect(page.locator('.selection.search.dropdown')).toBeInViewport({ratio: 1});
});

test('Switch branch', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();
  const response = await page.goto('/user2/repo1/graph');
  await expect(response?.status()).toBe(200);

  await page.click('#flow-select-refs-dropdown');
  const input = page.locator('#flow-select-refs-dropdown');
  await input.pressSequentially('develop', {delay: 50});
  await input.press('Enter');

  await page.waitForLoadState('networkidle');

  await expect(page.locator('#loading-indicator')).toBeHidden();
  await expect(page.locator('#rel-container')).toBeVisible();
  await expect(page.locator('#rev-container')).toBeVisible();
});
