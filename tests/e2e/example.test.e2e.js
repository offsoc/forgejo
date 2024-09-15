// @ts-check
import {expect} from '@playwright/test';
import {test, login_user, save_visual} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Load Homepage', async ({page}) => {
  const response = await page.goto('/');
  await expect(response?.status()).toBe(200); // Status OK
  await expect(page).toHaveTitle(/^Forgejo: Beyond coding. We Forge.\s*$/);
  await expect(page.locator('.logo')).toHaveAttribute('src', '/assets/img/logo.svg');
});

test('Register Form', async ({page}, workerInfo) => {
  const response = await page.goto('/user/sign_up');
  await expect(response?.status()).toBe(200); // Status OK
  await page.type('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
  await page.type('input[name=email]', `e2e-test-${workerInfo.workerIndex}@test.com`);
  await page.type('input[name=password]', 'test123test123');
  await page.type('input[name=retype]', 'test123test123');
  await page.click('form button.ui.primary.button:visible');
  // Make sure we routed to the home page. Else login failed.
  await expect(page.url()).toBe(`${workerInfo.project.use.baseURL}/`);
  await expect(page.locator('.secondary-nav span>img.ui.avatar')).toBeVisible();
  await expect(page.locator('.ui.positive.message.flash-success')).toHaveText('Account was successfully created. Welcome!');

  save_visual(page);
});

// eslint-disable-next-line playwright/no-skipped-test
test.describe.skip('example with different viewports (not actually run)', () => {
  // only necessary when the default web / mobile devices are not enough.
  // If you need to use a single fixed viewport, you can also use:
  // test.use({viewport: {width: 400, height: 800}});
  // also see https://playwright.dev/docs/test-parameterize
  for (const width of [400, 1000]) {
    // do not actually run (skip) this test
    test(`Do x on width: ${width}px`, async ({page}) => {
      await page.setViewportSize({
        width,
        height: 800,
      });
      // do something, then check that an element is fully in viewport
      // (i.e. not overflowing)
      await expect(page.locator('#my-element')).toBeInViewport({ratio: 1});
    });
  }
});
