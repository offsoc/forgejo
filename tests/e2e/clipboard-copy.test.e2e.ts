// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/home.tmpl
// templates/repo/diff/box.tmpl
// web_src/js/features/clipboard.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({
  permissions: ['clipboard-write'],
});

test('copy src file path to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/src/branch/master/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');
  const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipboardText).toContain('README.md');
});

test('copy diff file path to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/src/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');
  const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipboardText).toContain('README.md');
});
