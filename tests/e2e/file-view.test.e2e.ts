// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/view_file.tmpl
// web_src/css/repo/file-view.css
// web_src/js/features/repo-code.js
// web_src/js/features/repo-unicode-escape.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

const unselectedBg = 'rgba(0, 0, 0, 0)';
const selectedBg = 'rgb(255, 237, 213)';

test('Unicode escape highlight', async ({page}) => {
  const response = await page.goto('/user2/unicode-escaping/src/branch/main/a-file');
  expect(response?.status()).toBe(200);

  await expect(page.locator('.unicode-escape-prompt')).toBeVisible();
  expect(await page.locator('.lines-num').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);
  expect(await page.locator('.lines-escape').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);
  expect(await page.locator('.lines-code').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(unselectedBg);

  await page.locator('#L1').click();
  expect(await page.locator('.lines-num').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(selectedBg);
  expect(await page.locator('.lines-escape').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(selectedBg);
  expect(await page.locator('.lines-code').evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(selectedBg);

  await page.locator('.code-line-button').click();
  await expect(page.locator('.tippy-box .view_git_blame[href$="/a-file#L1"]')).toBeVisible();
  await expect(page.locator('.tippy-box .copy-line-permalink[data-url$="/a-file#L1"]')).toBeVisible();
});
