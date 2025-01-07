// @watch start
// web_src/js/components/DashboardRepoList.vue
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test.describe.configure({retries: 2});

test('Correct link and tooltip', async ({page}, testInfo) => {
  if (testInfo.retry) {
    await page.goto('/user2/test_workflows/actions');
  }

  const searchResponse = page.waitForResponse((resp) => resp.url().includes('/repo/search?') && resp.status() === 200);
  const response = await page.goto('/?repo-search-query=test_workflows');
  expect(response?.status()).toBe(200);

  await searchResponse;

  const repoStatus = page.locator('.dashboard-repos .repo-owner-name-list > li:nth-child(1) > a:nth-child(2)');
  await expect(repoStatus).toHaveAttribute('href', '/user2/test_workflows/actions', {timeout: 10000});
  await expect(repoStatus).toHaveAttribute('data-tooltip-content', /^(Error|Failure)$/);
  await save_visual(page);
});
