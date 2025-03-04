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
  await expect(globalError).toContainText('This Forgejo instance is configured to be served on ');
  await expect(globalError).toContainText('You are currently viewing Forgejo through a different URL, which may cause parts of the application to break. The canonical URL is controlled by Forgejo admins via the ROOT_URL setting in the app.ini.');
});
