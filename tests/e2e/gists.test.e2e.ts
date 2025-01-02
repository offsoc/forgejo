// @watch start
// templates/gist/**
// web_src/js/features/gist.ts
// @watch end

import {test, login_user, login} from './utils_e2e.ts';
import {expect} from '@playwright/test';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Create Gist', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  const response = await page.goto('/gists/-/new');
  expect(response?.status()).toBe(200);

  await page.locator('input[name="name"]').fill('NewGist');

  await page.locator('input[name="file-name-0"]').fill('file1.txt');
  await page.locator('textarea[name="file-content-0"]').fill('Hello');

  await page.locator('#add-gist-file-button').click();

  await page.locator('input[name="file-name-1"]').fill('file2.txt');
  await page.locator('textarea[name="file-content-1"]').fill('World');

  await page.locator('#submit-gist-button').click();

  await page.waitForSelector('#repo-clone-https');

  await expect(page.getByText('file1.txt')).toBeVisible();
  await expect(page.getByText('file2.txt')).toBeVisible();
});

test('Edit Gist', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  const response = await page.goto('/gists/dec037f3/edit');
  expect(response?.status()).toBe(200);

  await expect(page.locator('input[name="file-name-1"]')).toBeVisible()
  await expect(page.locator('textarea[name="file-content-1"]')).toBeVisible();

  await page.locator('button[data-file-id="1"]').click();

  await expect(page.locator('input[name="file-name-1"]')).not.toBeVisible();
  await expect(page.locator('textarea[name="file-content-1"]')).not.toBeVisible();
});
