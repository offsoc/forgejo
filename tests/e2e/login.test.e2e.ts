// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/user/auth/**
// web_src/js/features/user-**
// web_src/js/features/common-global.js
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual, test_context} from './utils_e2e.ts';

test('Mismatched ROOT_URL', async ({browser}) => {
  const context = await test_context(browser);
  const page = await context.newPage();

  // Ugly hack to override the appUrl of `window.config`.
  await page.addInitScript(() => {
    setInterval(() => {
      if (window.config) {
        window.config.appUrl = 'https://example.com';
      }
    }, 1);
  });

  const response = await page.goto('/user/login');
  expect(response?.status()).toBe(200);

  await save_visual(page);
  const globalError = page.locator('.js-global-error');
  await expect(globalError).toContainText('Your ROOT_URL, as set in app.ini, is');
  await expect(globalError).toContainText('which does not correspond to the site you are currently visiting. A mismatched ROOT_URL configuration can cause the application to break.');
});
