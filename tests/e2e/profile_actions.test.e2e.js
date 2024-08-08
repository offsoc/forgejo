// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test('Follow actions', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();

  await page.goto('/user1');
  await page.waitForLoadState('networkidle');

  // Check if following and then unfollowing works.
  // This checks that the event listeners of
  // the buttons aren't disappearing.
  const followButton = page.locator('.follow');
  await expect(followButton).toContainText('Follow');
  await followButton.click();
  await expect(followButton).toContainText('Unfollow');
  await followButton.click();
  await expect(followButton).toContainText('Follow');

  // Simple block interaction.
  await expect(page.locator('.block')).toContainText('Block');

  await page.locator('.block').click();
  await expect(page.locator('#block-user')).toBeVisible();
  await page.locator('#block-user .ok').click();
  await expect(page.locator('.block')).toContainText('Unblock');
  await expect(page.locator('#block-user')).toBeHidden();

  // Check that following the user yields in a error being shown.
  await followButton.click();
  const flashMessage = page.locator('#flash-message');
  await expect(flashMessage).toBeVisible();
  await expect(flashMessage).toContainText('You cannot follow this user because you have blocked this user or this user has blocked you.');

  // Unblock interaction.
  await page.locator('.block').click();
  await expect(page.locator('.block')).toContainText('Block');
});
