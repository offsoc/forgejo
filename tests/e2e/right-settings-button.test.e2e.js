// @ts-check

// @watch start
// templates/org/**
// templates/repo/**
// web_src/js/webcomponents/overflow-menu.js
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test.describe('desktop viewport', () => {
  test.use({viewport: {width: 1920, height: 300}});

  test('Settings button on right of repo header', async ({browser}, workerInfo) => {
    const context = await load_logged_in_context(browser, workerInfo, 'user2');
    const page = await context.newPage();

    await page.goto('/user2/repo1');

    const settingsBtn = page.locator('.overflow-menu-items>#settings-btn');
    await expect(settingsBtn).toBeVisible();
    await expect(settingsBtn).toHaveClass(/right/);

    await expect(page.locator('.overflow-menu-button')).toHaveCount(0);
  });

  test('Settings button on right of repo header also when add more button is shown', async ({browser}, workerInfo) => {
    await login_user(browser, workerInfo, 'user12');
    const context = await load_logged_in_context(browser, workerInfo, 'user12');
    const page = await context.newPage();

    await page.goto('/user12/repo10');

    const settingsBtn = page.locator('.overflow-menu-items>#settings-btn');
    await expect(settingsBtn).toBeVisible();
    await expect(settingsBtn).toHaveClass(/right/);

    await expect(page.locator('.overflow-menu-button')).toHaveCount(0);
  });

  test('Settings button on right of org header', async ({browser}, workerInfo) => {
    const context = await load_logged_in_context(browser, workerInfo, 'user2');
    const page = await context.newPage();

    await page.goto('/org3');

    const settingsBtn = page.locator('.overflow-menu-items>#settings-btn');
    await expect(settingsBtn).toBeVisible();
    await expect(settingsBtn).toHaveClass(/right/);

    await expect(page.locator('.overflow-menu-button')).toHaveCount(0);
  });

  test('User overview overflow menu should not be influenced', async ({page}) => {
    await page.goto('/user2');

    await expect(page.locator('.overflow-menu-items>#settings-btn')).toHaveCount(0);

    await expect(page.locator('.overflow-menu-button')).toHaveCount(0);
  });
});

test.describe('small viewport', () => {
  test.use({viewport: {width: 800, height: 300}});

  test('Settings button in overflow menu of repo header', async ({browser}, workerInfo) => {
    const context = await load_logged_in_context(browser, workerInfo, 'user2');
    const page = await context.newPage();

    await page.goto('/user2/repo1');

    await expect(page.locator('.overflow-menu-items>#settings-btn')).toHaveCount(0);

    await expect(page.locator('.overflow-menu-button')).toBeVisible();

    await page.click('.overflow-menu-button');
    await expect(page.locator('.tippy-target>#settings-btn')).toBeVisible();

    // Verify that we have no duplicated items
    const shownItems = await page.locator('.overflow-menu-items>a').all();
    expect(shownItems).not.toHaveLength(0);
    const overflowItems = await page.locator('.tippy-target>a').all();
    expect(overflowItems).not.toHaveLength(0);

    const items = shownItems.concat(overflowItems);
    expect(Array.from(new Set(items))).toHaveLength(items.length);
  });

  test('Settings button in overflow menu of org header', async ({browser}, workerInfo) => {
    const context = await load_logged_in_context(browser, workerInfo, 'user2');
    const page = await context.newPage();

    await page.goto('/org3');

    await expect(page.locator('.overflow-menu-items>#settings-btn')).toHaveCount(0);

    await expect(page.locator('.overflow-menu-button')).toBeVisible();

    await page.click('.overflow-menu-button');
    await expect(page.locator('.tippy-target>#settings-btn')).toBeVisible();

    // Verify that we have no duplicated items
    const shownItems = await page.locator('.overflow-menu-items>a').all();
    expect(shownItems).not.toHaveLength(0);
    const overflowItems = await page.locator('.tippy-target>a').all();
    expect(overflowItems).not.toHaveLength(0);

    const items = shownItems.concat(overflowItems);
    expect(Array.from(new Set(items))).toHaveLength(items.length);
  });

  test('User overview overflow menu should not be influenced', async ({page}) => {
    await page.goto('/user2');

    await expect(page.locator('.overflow-menu-items>#settings-btn')).toHaveCount(0);

    await expect(page.locator('.overflow-menu-button')).toBeVisible();
    await page.click('.overflow-menu-button');
    await expect(page.locator('.tippy-target>#settings-btn')).toHaveCount(0);

    // Verify that we have no duplicated items
    const shownItems = await page.locator('.overflow-menu-items>a').all();
    expect(shownItems).not.toHaveLength(0);
    const overflowItems = await page.locator('.tippy-target>a').all();
    expect(overflowItems).not.toHaveLength(0);

    const items = shownItems.concat(overflowItems);
    expect(Array.from(new Set(items))).toHaveLength(items.length);
  });
});
