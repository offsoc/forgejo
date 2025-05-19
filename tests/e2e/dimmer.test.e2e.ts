// @watch start
// templates/shared/user/**
// web_src/css/modules/dimmer.ts
// web_src/css/modules/dimmer.css
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Dimmed modal', async ({page}) => {
  await page.goto('/user1');

  await expect(page.locator('#action-block')).toContainText('Block');

  // Ensure the modal is hidden
  await expect(page.locator('#block-user')).toBeHidden();

  await page.locator('.actions .dropdown').click();
  await page.locator('#action-block').click();

  // Modal and dimmer should be visible.
  await expect(page.locator('#block-user')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toBeVisible();
  await save_visual(page);

  // After canceling, modal and dimmer should be hidden.
  await page.locator('#block-user .cancel').click();
  await expect(page.locator('.ui.dimmer')).toBeHidden();
  await expect(page.locator('#block-user')).toBeHidden();
  await save_visual(page);

  // Open the block modal and make the dimmer visible again.
  await page.locator('.actions .dropdown').click();
  await page.locator('#action-block').click();
  await expect(page.locator('#block-user')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toBeVisible();
  await expect(page.locator('.ui.dimmer')).toHaveCount(1);
  await save_visual(page);
});

test('Dimmed overflow', async ({page}, workerInfo) => {
  test.skip(['Mobile Safari'].includes(workerInfo.project.name), 'Mouse wheel is not supported in mobile WebKit');
  await page.goto('/user2/repo1/_new/master/');

  // Type in a file name.
  await page.locator('#file-name').click();
  await page.keyboard.type('todo.txt');

  // Scroll to the bottom.
  const scrollY = await page.evaluate(() => document.body.scrollHeight);
  await page.mouse.wheel(0, scrollY);

  // Click on 'Commit changes'
  await page.locator('#commit-button').click();

  // Expect a 'are you sure, this file is empty' modal.
  await expect(page.locator('.ui.dimmer')).toBeVisible();
  await expect(page.locator('.ui.dimmer .header')).toContainText('Commit an empty file');
  await save_visual(page);

  // Trickery to check that the dimmer covers the whole page.
  const viewport = page.viewportSize();
  const box = await page.locator('.ui.dimmer').boundingBox();
  expect(box.x).toBe(0);
  expect(box.y).toBe(0);
  expect(box.width).toBe(viewport.width);
  expect(box.height).toBe(viewport.height);

  // Trickery to check the page cannot be scrolled.
  const {scrollHeight, clientHeight} = await page.evaluate(() => document.body);
  expect(scrollHeight).toBe(clientHeight);
});
