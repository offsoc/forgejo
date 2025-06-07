// @watch start
// templates/shared/user/**
// web_src/js/modules/dropdown.ts
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('JS enhanced', async ({page}) => {
  await page.goto('/user1');

  const nojsNotice = page.locator('body .full noscript');
  await expect(nojsNotice).toBeHidden();

  // Open and close
  const dropdown = page.locator('details.dropdown.js-enhanced');
  const dropdownContent = page.locator('details.dropdown ul');
  await expect(dropdownContent).toBeHidden();
  await dropdown.click();
  await expect(dropdownContent).toBeVisible();
  await dropdown.click();
  await expect(dropdownContent).toBeHidden();

  // Open and close by opening a different dropdown
  const languageMenu = page.locator('.language-menu');
  await dropdown.click();
  await expect(dropdownContent).toBeVisible();
  await expect(languageMenu).toBeHidden();
  await page.locator('.language.dropdown').click();
  await expect(dropdownContent).toBeHidden();
  await expect(languageMenu).toBeVisible();

  // Todo: keyboard nav
});

test('No JS', async ({browser}) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const nojsPage = await context.newPage();
  await nojsPage.goto('/user1');

  const nojsNotice = nojsPage.locator('body .full noscript');
  await expect(nojsNotice).toBeVisible();

  // Open and close
  const dropdown = nojsPage.locator('details.dropdown');
  const dropdownContent = nojsPage.locator('details.dropdown ul');
  await expect(dropdownContent).toBeHidden();
  await dropdown.click();
  await expect(dropdownContent).toBeVisible();
  await dropdown.click();
  await expect(dropdownContent).toBeHidden();
});
