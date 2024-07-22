// @ts-check
import {test, expect} from '@playwright/test';

test('markup with #xyz-mode-only', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  await expect(response?.status()).toBe(200);
  await page.waitForLoadState('networkidle');

  const comment = page.locator('.comment-body>.markup', {hasText: 'test markup light/dark-mode-only'});
  await expect(comment).toBeVisible();
  await expect(comment.locator('[src$="#gh-light-mode-only"]')).toBeVisible();
  await expect(comment.locator('[src$="#gh-dark-mode-only"]')).toBeHidden();
});
