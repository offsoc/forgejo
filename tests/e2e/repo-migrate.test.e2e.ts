// @watch start
// web_src/js/features/repo-migrate.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './_test-setup.ts';

test.describe('Migration progress page', () => {
  test.use({user: 'user2'});

  test('Migration Progress Page', async ({page, browser}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky actionability checks on Mobile Safari');

    const repoName = `invalid-repo-${workerInfo.workerIndex}`;
    const repoNameAbsolut = `user2/${repoName}`;
    const repoUrl = `/user2/${repoName}`;

    expect((await page.goto(repoUrl))?.status(), 'repo should not exist yet').toBe(404);

    await page.goto('/repo/migrate?service_type=1');

    const form = page.locator('form');
    await form.getByRole('textbox', {name: 'Repository Name'}).fill(repoName);
    await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill(`https://codeberg.org/forgejo/${repoName}`);
    await save_visual(page);
    await form.locator('button.primary').click({timeout: 5000});
    await expect(page).toHaveURL(repoUrl);
    await save_visual(page);

    // page screenshot of unauthedPage is checked automatically after the test
    await test.step('migration should have failed', async () => {
      const ctx = await browser.newContext();
      const unauthenticated = await ctx.newPage();
      expect((await unauthenticated.goto(repoUrl))?.status(), 'public migration page should be accessible').toBe(200);
      await expect(unauthenticated.locator('#repo_migrating_progress')).toBeVisible();
    });

    await test.step('migration should have failed', async () => {
      await page.reload();
      await expect(page.locator('#repo_migrating_failed')).toBeVisible();
      await save_visual(page);
    });

    await test.step('delete repo', async () => {
      await page.getByRole('button', {name: 'Delete this repository'}).click();
      const deleteModal = page.locator('#delete-repo-modal');
      await deleteModal.getByRole('textbox', {name: 'Confirmation string'}).fill(repoNameAbsolut);
      await save_visual(page);
      await deleteModal.getByRole('button', {name: 'Delete repository'}).click();
      await expect(page).toHaveURL('/');
    });
  });
});
