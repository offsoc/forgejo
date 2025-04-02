// @watch start
// templates/shared/user/**
// web_src/css/modules/dimmer.ts
// web_src/css/modules/dimmer.css
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Dimmed modal', async ({page}) => {
  await page.goto('/user1');

  await expect(page.locator('.block')).toContainText('Block');

  // Ensure the modal is hidden
  await expect(page.locator('#block-user')).toBeHidden();

  await page.locator('.block').click();

  // Modal and dimmer should be visible.
  await expect(page.locator('#block-user')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toBeVisible();
  await save_visual(page);

  // After canceling, modal and dimmer should be hidden.
  await page.locator('#block-user .cancel').click();
  await expect(page.locator('.ui.dimmer')).toBeHidden();
  await expect(page.locator('#block-user')).toBeHidden();
  await save_visual(page);

  // Open the block modal and make the dimmer visible again.
  await page.locator('.block').click();
  await expect(page.locator('#block-user')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toHaveCount(1);
  await save_visual(page);
});
