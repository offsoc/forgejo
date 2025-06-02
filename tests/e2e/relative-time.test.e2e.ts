// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/admin/dashboard.tmpl
// web_src/js/webcomponents/relative-time.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user1'});

test('Relative time after htmx swap', async ({page}, workerInfo) => {
  test.skip(
    workerInfo.project.name !== 'firefox' && workerInfo.project.name !== 'Mobile Chrome',
    'This is a really slow test, so limit to a subset of client.',
  );
  await page.goto('/admin');

  const relativeTime = page.locator('.admin-dl-horizontal > dd:nth-child(2) > relative-time');
  await expect(relativeTime).toContainText('ago');

  const body = page.locator('body');
  await body.evaluate(
    (element) =>
      new Promise((resolve) =>
        element.addEventListener('htmx:afterSwap', () => {
          resolve();
        }),
      ),
  );

  await expect(relativeTime).toContainText('ago');
});
