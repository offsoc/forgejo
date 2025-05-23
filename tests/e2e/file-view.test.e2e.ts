// @watch start
// web_src/css/markup/**
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

const unselectedBg = 'rgba(0, 0, 0, 0)';
const selectedBg = 'rgb(255, 237, 213)';

test('Unicod escape highlight', async ({page}) => {
  const response = await page.goto('/user2/unicode-escaping/src/branch/main/a-file');
  expect(response?.status()).toBe(200);

  await expect(page.locator('.unicode-escape-prompt')).toBeVisible();
  await expect(await page.locator('.lines-num').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);
  await expect(await page.locator('.lines-escape').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);
  await expect(await page.locator('.lines-code').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);

  await page.locator('#L1').click()
  await expect(await page.locator('.lines-num').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(selectedBg);
  await expect(await page.locator('.lines-escape').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(selectedBg);
  await expect(await page.locator('.lines-code').evaluate(el => getComputedStyle(el).backgroundColor)).toBe(selectedBg);
});
