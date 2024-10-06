// @ts-check

// @watch start
// models/repo/attachment.go
// modules/structs/attachment.go
// routers/web/repo/**
// services/attachment/**
// services/release/**
// templates/repo/release/**
// web_src/js/features/repo-release.js
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, save_visual, load_logged_in_context} from './utils_e2e.js';
import {validate_form} from './shared/forms.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test.describe.configure({
  timeout: 30000,
});

test('External Release Attachments', async ({browser, isMobile}, workerInfo) => {
  test.skip(isMobile);

  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  /** @type {import('@playwright/test').Page} */
  const page = await context.newPage();

  // Click "New Release"
  await page.goto('/user2/repo2/releases');
  await page.click('.button.small.primary');

  // Fill out form and create new release
  await expect(page).toHaveURL('/user2/repo2/releases/new');
  await validate_form({page}, 'fieldset');
  await page.fill('input[name=tag_name]', '2.0');
  await page.fill('input[name=title]', '2.0');
  await page.click('#add-external-link');
  await page.click('#add-external-link');
  await page.fill('input[name=attachment-new-name-2]', 'Test');
  await page.fill('input[name=attachment-new-exturl-2]', 'https://forgejo.org/');
  await page.click('.remove-rel-attach');
  save_visual(page);
  await page.click('.button.small.primary');

  // Validate release page and click edit
  await expect(page).toHaveURL('/user2/repo2/releases');
  await expect(page.locator('.download[open] li')).toHaveCount(3);
  await expect(page.locator('.download[open] li:nth-of-type(3)')).toContainText('Test');
  await expect(page.locator('.download[open] li:nth-of-type(3) a')).toHaveAttribute('href', 'https://forgejo.org/');
  save_visual(page);
  await page.locator('.octicon-pencil').first().click();

  // Validate edit page and edit the release
  await expect(page).toHaveURL('/user2/repo2/releases/edit/2.0');
  await validate_form({page}, 'fieldset');
  await expect(page.locator('.attachment_edit:visible')).toHaveCount(2);
  await expect(page.locator('.attachment_edit:visible').nth(0)).toHaveValue('Test');
  await expect(page.locator('.attachment_edit:visible').nth(1)).toHaveValue('https://forgejo.org/');
  await page.locator('.attachment_edit:visible').nth(0).fill('Test2');
  await page.locator('.attachment_edit:visible').nth(1).fill('https://gitea.io/');
  await page.click('#add-external-link');
  await expect(page.locator('.attachment_edit:visible')).toHaveCount(4);
  await page.locator('.attachment_edit:visible').nth(2).fill('Test3');
  await page.locator('.attachment_edit:visible').nth(3).fill('https://gitea.com/');
  save_visual(page);
  await page.click('.button.small.primary');

  // Validate release page and click edit
  await expect(page).toHaveURL('/user2/repo2/releases');
  await expect(page.locator('.download[open] li')).toHaveCount(4);
  await expect(page.locator('.download[open] li:nth-of-type(3)')).toContainText('Test2');
  await expect(page.locator('.download[open] li:nth-of-type(3) a')).toHaveAttribute('href', 'https://gitea.io/');
  await expect(page.locator('.download[open] li:nth-of-type(4)')).toContainText('Test3');
  await expect(page.locator('.download[open] li:nth-of-type(4) a')).toHaveAttribute('href', 'https://gitea.com/');
  save_visual(page);
  await page.locator('.octicon-pencil').first().click();

  // Delete release
  await expect(page).toHaveURL('/user2/repo2/releases/edit/2.0');
  await page.click('.delete-button');
  await page.click('.button.ok');
  await expect(page).toHaveURL('/user2/repo2/releases');
});
