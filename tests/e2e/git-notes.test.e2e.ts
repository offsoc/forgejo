// @ts-check
import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Change git note', async ({page}) => {
  let response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  await page.locator('#commit-notes-edit-button').click();

  let textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).toBeVisible();
  await textarea.fill('This is a new note');
  await save_visual(page);

  await page.locator('#notes-save-button').click();
  await save_visual(page);

  response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).toHaveText('This is a new note');
  await save_visual(page);
});
