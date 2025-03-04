// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/issue/view_title.tmpl
// web_src/js/features/repo-issue.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('PR: title edit', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/5');
  expect(response?.status()).toBe(200);

  await expect(page.locator('#editable-label')).toBeVisible();
  await save_visual(page);

  // Labels AGit and Editable are hidden when title is in edit mode
  await page.locator('#issue-title-edit-show').click();
  await expect(page.locator('#editable-label')).toBeHidden();
  await save_visual(page);
});
