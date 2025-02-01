// @watch start
// web_src/js/features/common-global.js
// web_src/css/repo.css
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test('Language stats bar', async ({page}) => {
  const response = await page.goto('/user2/repo1');
  expect(response?.status()).toBe(200);

  await expect(page.locator('#language-stats-legend')).toBeVisible();
  await save_visual(page);

  await page.click('#language-stats-bar');
  await expect(page.locator('#language-stats-legend')).toBeHidden();
  await save_visual(page);
});
