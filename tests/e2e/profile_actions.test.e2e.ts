// @watch start
// routers/web/user/**
// templates/shared/user/**
// web_src/js/features/common-global.js
// web_src/js/modules/dropdown.ts
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Follow and block actions', async ({page}) => {
  await page.goto('/user1');

  // Check if following and then unfollowing works.
  const followButton = page.locator('.primary-action button');
  await expect(followButton).toContainText('Follow');
  await followButton.click();
  await expect(followButton).toContainText('Unfollow');
  await followButton.click();
  await expect(followButton).toContainText('Follow');

  // Simple block interaction.
  const actionsDropdownBtn = page.locator('.actions .dropdown summary');
  const blockButton = page.locator('#action-block');
  await expect(blockButton).toBeHidden();

  await actionsDropdownBtn.click();
  await expect(blockButton).toBeVisible();
  await expect(blockButton).toContainText('Block');

  await blockButton.click();
  await expect(page.locator('#block-user')).toBeVisible();
  await save_visual(page);
  await page.locator('#block-user .ok').click();
  await expect(blockButton).toContainText('Unblock');
  await expect(page.locator('#block-user')).toBeHidden();

  // Check that following the user yields in a error being shown.
  await followButton.click();
  const flashMessage = page.locator('#flash-message');
  await expect(flashMessage).toBeVisible();
  await expect(flashMessage).toContainText('You cannot follow this user because you have blocked this user or this user has blocked you.');
  await save_visual(page);

  // Unblock interaction.
  await actionsDropdownBtn.click();
  await blockButton.click();
  await expect(blockButton).toContainText('Block');
});
