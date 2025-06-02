// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/user_cards.tmpl
// web_src/css/modules/user-cards.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('Usercards width', async ({page}) => {
  await page.goto('/user8?tab=followers');

  // Regardless of whether cards in a grid or flex mode, they should be ~same
  // width. Verifying this relies on fixtures with users that have long website
  // link or other content that could push the card width.
  const widths = [];
  const amount = 3;

  for (let i = 1; i <= amount; i++) {
    const card = await page.locator(`.user-cards .card:nth-child(${i})`).boundingBox();
    widths.push(Math.round(card.width));
  }

  for (const width of widths) {
    expect(width).toBe(widths[0]);
  }
});
