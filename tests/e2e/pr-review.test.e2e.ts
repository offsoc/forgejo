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

test('PR: Navigate by single commit', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/3/commits');
  expect(response?.status()).toBe(200);

  await page.locator('tbody.commit-list td.message a').nth(1).click();
  await page.waitForURL(/.*\/user2\/repo1\/pulls\/3\/commits\/4a357436d925b5c974181ff12a994538ddc5a269/);
  await save_visual(page);

  let prevButton = page.locator('.commit-header-buttons').getByText(/Prev/);
  let nextButton = page.locator('.commit-header-buttons').getByText(/Next/);
  await prevButton.waitFor();
  await nextButton.waitFor();

  await expect(prevButton).toHaveClass(/disabled/);
  await expect(nextButton).not.toHaveClass(/disabled/);
  await expect(nextButton).toHaveAttribute('href', '/user2/repo1/pulls/3/commits/5f22f7d0d95d614d25a5b68592adb345a4b5c7fd');
  await nextButton.click();

  await page.waitForURL(/.*\/user2\/repo1\/pulls\/3\/commits\/5f22f7d0d95d614d25a5b68592adb345a4b5c7fd/);
  await save_visual(page);

  prevButton = page.locator('.commit-header-buttons').getByText(/Prev/);
  nextButton = page.locator('.commit-header-buttons').getByText(/Next/);
  await prevButton.waitFor();
  await nextButton.waitFor();

  await expect(prevButton).not.toHaveClass(/disabled/);
  await expect(nextButton).toHaveClass(/disabled/);
  await expect(prevButton).toHaveAttribute('href', '/user2/repo1/pulls/3/commits/4a357436d925b5c974181ff12a994538ddc5a269');
});
