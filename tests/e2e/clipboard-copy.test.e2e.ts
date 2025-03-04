// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/home.tmpl
// templates/repo/diff/box.tmpl
// web_src/js/features/clipboard.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test('copy src file path to clipboard', async ({page}, workerInfo) => {
  test.skip(['Mobile Safari', 'webkit'].includes(workerInfo.project.name), 'Apple clipboard API addon - starting at just $499!');

  const response = await page.goto('/user2/repo1/src/branch/master/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');
  const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipboardText).toContain('README.md');
  await expect(page.getByText('Copied')).toBeVisible();
  await save_visual(page);
});

test('copy diff file path to clipboard', async ({page}, workerInfo) => {
  test.skip(['Mobile Safari', 'webkit'].includes(workerInfo.project.name), 'Apple clipboard API addon - starting at just $499!');

  const response = await page.goto('/user2/repo1/src/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');
  const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipboardText).toContain('README.md');
  await expect(page.getByText('Copied')).toBeVisible();
  await save_visual(page);
});
