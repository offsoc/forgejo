// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/diff/new_review.tmpl
// web_src/js/features/repo-issue.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('PR: Finish review', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/5/files');
  expect(response?.status()).toBe(200);

  await expect(page.locator('.tippy-box .review-box-panel')).toBeHidden();
  await save_visual(page);

  // Review panel should appear after clicking Finish review
  await page.locator('#review-box .js-btn-review').click();
  await expect(page.locator('.tippy-box .review-box-panel')).toBeVisible();
  await save_visual(page);
});
