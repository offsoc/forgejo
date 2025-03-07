// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/diff/new_review.tmpl
// web_src/js/features/repo-issue.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('PR: Create review from files', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/5/files');
  expect(response?.status()).toBe(200);

  await expect(page.locator('.tippy-box .review-box-panel')).toBeHidden();
  await save_visual(page);

  // Review panel should appear after clicking Finish review
  await page.locator('#review-box .js-btn-review').click();
  await expect(page.locator('.tippy-box .review-box-panel')).toBeVisible();
  await save_visual(page);

  await page.locator('.review-box-panel textarea#_combo_markdown_editor_0')
    .fill('This is a review');
  await page.locator('.review-box-panel button.btn-submit[value="approve"]').click();
  await page.waitForURL(/.*\/user2\/repo1\/pulls\/5#issuecomment-\d+/);
  await save_visual(page);
});

test('PR: Create review from commit', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/3/commits/4a357436d925b5c974181ff12a994538ddc5a269');
  expect(response?.status()).toBe(200);

  await page.locator('button.add-code-comment').click();
  const code_comment = page.locator('.comment-code-cloud form textarea.markdown-text-editor');
  await expect(code_comment).toBeVisible();

  await code_comment.fill('This is a code comment');
  await save_visual(page);

  const start_button = page.locator('.comment-code-cloud form button.btn-start-review');
  // Workaround for #7152, where there might already be a pending review state from previous
  // test runs (most likely to happen when debugging tests).
  if (await start_button.isVisible({timeout: 100})) {
    await start_button.click();
  } else {
    await page.locator('.comment-code-cloud form button.btn-add-comment').click();
  }

  await expect(page.locator('.comment-list .comment-container')).toBeVisible();

  // We need to wait for the review to be processed. Checking the comment counter
  // conveniently does that.
  await expect(page.locator('#review-box .js-btn-review > span.review-comments-counter')).toHaveText('1');

  await page.locator('#review-box .js-btn-review').click();
  await expect(page.locator('.tippy-box .review-box-panel')).toBeVisible();
  await save_visual(page);

  await page.locator('.review-box-panel textarea.markdown-text-editor')
    .fill('This is a review');
  await page.locator('.review-box-panel button.btn-submit[value="approve"]').click();
  await page.waitForURL(/.*\/user2\/repo1\/pulls\/3#issuecomment-\d+/);
  await save_visual(page);

  // In addition to testing the ability to delete comments, this also
  // performs clean up. If tests are run for multiple platforms, the data isn't reset
  // in-between, and subsequent runs of this test would fail, because when there already is
  // a comment, the on-hover button to start a conversation doesn't appear anymore.
  await page.goto('/user2/repo1/pulls/3/commits/4a357436d925b5c974181ff12a994538ddc5a269');
  await page.locator('.comment-header-right.actions a.context-menu').click();

  await expect(page.locator('.comment-header-right.actions div.menu').getByText(/Copy link.*/)).toBeVisible();
  // The button to delete a comment will prompt for confirmation using a browser alert.
  page.on('dialog', (dialog) => dialog.accept());
  await page.locator('.comment-header-right.actions div.menu .delete-comment').click();

  await expect(page.locator('.comment-list .comment-container')).toBeHidden();
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
