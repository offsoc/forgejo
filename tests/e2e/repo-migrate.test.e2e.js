// @ts-check

// @watch start
// web_src/js/features/repo-migrate.js
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(({browser}, workerInfo) => login_user(browser, workerInfo, 'user2'));

test('Migration Progress Page', async ({page: unauthedPage, browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky actionability checks on Mobile Safari');

  const page = await (await load_logged_in_context(browser, workerInfo, 'user2')).newPage();

  await expect((await page.goto('/user2/invalidrepo'))?.status(), 'repo should not exist yet').toBe(404);

  await page.goto('/repo/migrate?service_type=1');

  const form = page.locator('form');
  await form.getByRole('textbox', {name: 'Repository Name'}).fill('invalidrepo');
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill('https://codeberg.org/forgejo/invalidrepo');
  await form.locator('button.primary').click({timeout: 5000});
  await expect(page).toHaveURL('user2/invalidrepo');

  await expect((await unauthedPage.goto('/user2/invalidrepo'))?.status(), 'public migration page should be accessible').toBe(200);
  await expect(unauthedPage.locator('#repo_migrating_progress')).toBeVisible();

  await page.reload();
  await expect(page.locator('#repo_migrating_failed')).toBeVisible();
  await page.getByRole('button', {name: 'Delete this repository'}).click();
  const deleteModal = page.locator('#delete-repo-modal');
  await deleteModal.getByRole('textbox', {name: 'Confirmation string'}).fill('user2/invalidrepo');
  await deleteModal.getByRole('button', {name: 'Delete repository'}).click();
  await expect(page).toHaveURL('/');
});
