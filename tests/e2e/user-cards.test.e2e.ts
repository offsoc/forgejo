// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/user_cards.tmpl
// web_src/css/modules/user-cards.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Usercards - grid', async ({page}) => {
  await page.goto('/user8?tab=followers');

  // Verify that the cards are ~same width. Testdata has users with long website
  // links that could push squash neighbor cards
  const widths = [];
  const amount = 3;

  for (let i = 1; i <= amount; i++) {
    const card = await page.locator(`.user-cards .card:nth-child(${i})`).boundingBox();
    widths.push(Math.round(card.width));
  }

  widths.forEach(width => {
    expect(width).toBe(widths[0]);
  });
});
