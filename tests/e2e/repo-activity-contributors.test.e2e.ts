// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// web_src/js/features/contributors.js
// web_src/js/components/RepoContributors.vue
// templates/repo/*
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('Contributor graph', async ({page}) => {
  await page.goto('/user2/commits_search_test/activity/contributors');
  await page.getByRole('link', {name: '2 Commits'}).click();
  await expect(page.getByRole('cell', {name: 'Bob'})).toHaveCount(2);
});
