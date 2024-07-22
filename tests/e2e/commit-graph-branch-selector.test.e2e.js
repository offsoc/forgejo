// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
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
