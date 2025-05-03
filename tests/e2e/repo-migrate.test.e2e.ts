// @watch start
// web_src/js/features/repo-migrate.js
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual, test_context, dynamic_id} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Migration type seleciton screen', async ({page}) => {
  await page.goto('/repo/migrate');

  // For branding purposes, it is desired that `gitea-` prefixes in SVGs are
  // replaced with something like `productlogo-`.
  await expect(page.locator('svg.gitea-git')).toBeVisible();
  await expect(page.locator('svg.octicon-mark-github')).toBeVisible();
  await expect(page.locator('svg.gitea-gitlab')).toBeVisible();
  await expect(page.locator('svg.gitea-forgejo')).toBeVisible();
  await expect(page.locator('svg.gitea-gitea')).toBeVisible();
  await expect(page.locator('svg.gitea-gogs')).toBeVisible();
  await expect(page.locator('svg.gitea-onedev')).toBeVisible();
  await expect(page.locator('svg.gitea-gitbucket')).toBeVisible();
  await expect(page.locator('svg.gitea-codebase')).toBeVisible();

  await save_visual(page);
});

test('Migration Repo Name detection', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky actionability checks on Mobile Safari');

  await page.goto('/repo/migrate?service_type=2');

  const form = page.locator('form');

  // Test trailing slashes are stripped
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill('https://github.com/example/test/');
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).blur();
  await expect(form.getByRole('textbox', {name: 'Repository Name'})).toHaveValue('test');

  // Test trailing .git is stripped
  await page.reload();
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill('https://github.com/example/test.git');
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).blur();
  await expect(form.getByRole('textbox', {name: 'Repository Name'})).toHaveValue('test');

  // Test trailing .git and trailing / together is stripped
  await page.reload();
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill('https://github.com/example/test.git/');
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).blur();
  await expect(form.getByRole('textbox', {name: 'Repository Name'})).toHaveValue('test');

  // Save screenshot only once
  await save_visual(page);
});

test('Migration Progress Page', async ({page, browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky actionability checks on Mobile Safari');

  const repoName = dynamic_id();
  expect((await page.goto(`/user2/${repoName}`))?.status(), 'repo should not exist yet').toBe(404);

  await page.goto('/repo/migrate?service_type=1');

  const form = page.locator('form');
  await form.getByRole('textbox', {name: 'Repository Name'}).fill(repoName);
  await form.getByRole('textbox', {name: 'Migrate / Clone from URL'}).fill(`https://codeberg.org/forgejo/${repoName}`);
  await save_visual(page);
  await form.locator('button.primary').click({timeout: 5000});
  await expect(page).toHaveURL(`user2/${repoName}`);
  await save_visual(page);

  const ctx = await test_context(browser, {storageState: {cookies: [], origins: []}});
  const unauthenticatedPage = await ctx.newPage();
  expect((await unauthenticatedPage.goto(`/user2/${repoName}`))?.status(), 'public migration page should be accessible').toBe(200);
  await expect(unauthenticatedPage.locator('#repo_migrating_progress')).toBeVisible();

  await page.reload();
  await expect(page.locator('#repo_migrating_failed')).toBeVisible();
  await save_visual(page);
  await page.getByRole('button', {name: 'Delete this repository'}).click();
  const deleteModal = page.locator('#delete-repo-modal');
  await deleteModal.getByRole('textbox', {name: 'Confirmation string'}).fill(`user2/${repoName}`);
  await save_visual(page);
  await deleteModal.getByRole('button', {name: 'Delete repository'}).click();
  await expect(page).toHaveURL('/');
  // checked last to preserve the order of screenshots from first run
  await save_visual(unauthenticatedPage);
});
