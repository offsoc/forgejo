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
import {save_visual, test} from './utils_e2e.ts';
import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test.describe.configure({
  timeout: 30000,
});

test('External Release Attachments', async ({page, isMobile}) => {
  test.skip(isMobile);

  // Click "New Release"
  await page.goto('/user2/repo2/releases');
  await page.click('.button.small.primary');

  // Fill out form and create new release
  await expect(page).toHaveURL('/user2/repo2/releases/new');
  await validate_form({page}, 'fieldset');
  const textarea = page.locator('input[name=tag_name]');
  await textarea.pressSequentially('2.0');
  await expect(page.locator('input[name=title]')).toHaveAttribute('placeholder', '2.0');
  await page.fill('input[name=title]', '2.0');
  await page.click('#add-external-link');
  await page.click('#add-external-link');
  await page.fill('input[name=attachment-new-name-2]', 'Test');
  await page.fill('input[name=attachment-new-exturl-2]', 'https://forgejo.org/');
  await page.click('.remove-rel-attach');
  await save_visual(page);
  await page.click('.button.small.primary');

  // Validate release page and click edit
  await expect(page).toHaveURL('/user2/repo2/releases');
  await expect(page.locator('.download[open] li')).toHaveCount(3);

  await expect(page.locator('.download[open] li:nth-of-type(1)')).toContainText('Source code (ZIP)');
  await expect(page.locator('.download[open] li:nth-of-type(1) span[data-tooltip-content]')).toHaveAttribute('data-tooltip-content', 'This attachment is automatically generated.');
  await expect(page.locator('.download[open] li:nth-of-type(1) a')).toHaveAttribute('href', '/user2/repo2/archive/2.0.zip');
  await expect(page.locator('.download[open] li:nth-of-type(1) a')).toHaveAttribute('type', 'application/zip');

  await expect(page.locator('.download[open] li:nth-of-type(2)')).toContainText('Source code (TAR.GZ)');
  await expect(page.locator('.download[open] li:nth-of-type(2) span[data-tooltip-content]')).toHaveAttribute('data-tooltip-content', 'This attachment is automatically generated.');
  await expect(page.locator('.download[open] li:nth-of-type(2) a')).toHaveAttribute('href', '/user2/repo2/archive/2.0.tar.gz');
  await expect(page.locator('.download[open] li:nth-of-type(2) a')).toHaveAttribute('type', 'application/gzip');

  await expect(page.locator('.download[open] li:nth-of-type(3)')).toContainText('Test');
  await expect(page.locator('.download[open] li:nth-of-type(3) a')).toHaveAttribute('href', 'https://forgejo.org/');
  await save_visual(page);
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
  await save_visual(page);
  await page.click('.button.small.primary');

  // Validate release page and click edit
  await expect(page).toHaveURL('/user2/repo2/releases');
  await expect(page.locator('.download[open] li')).toHaveCount(4);
  await expect(page.locator('.download[open] li:nth-of-type(3)')).toContainText('Test2');
  await expect(page.locator('.download[open] li:nth-of-type(3) a')).toHaveAttribute('href', 'https://gitea.io/');
  await expect(page.locator('.download[open] li:nth-of-type(4)')).toContainText('Test3');
  await expect(page.locator('.download[open] li:nth-of-type(4) a')).toHaveAttribute('href', 'https://gitea.com/');
  await save_visual(page);
  await page.locator('.octicon-pencil').first().click();

  // Delete release
  await expect(page).toHaveURL('/user2/repo2/releases/edit/2.0');
  await page.click('.delete-button');
  await page.click('.button.ok');
  await expect(page).toHaveURL('/user2/repo2/releases');
});
