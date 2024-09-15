// @ts-check
import {expect} from '@playwright/test';
import {test} from './utils_e2e.js';

test(`Search for long titles and test for no overflow`, async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Fails as always, see https://codeberg.org/forgejo/forgejo/pulls/5326#issuecomment-2313275');
  await page.goto('/user2/repo1/wiki');
  await page.waitForLoadState('networkidle');
  await page.getByPlaceholder('Search wiki').fill('spaces');
  await page.getByPlaceholder('Search wiki').click();
  // workaround: HTMX listens on keyup events, playwright's fill only triggers the input event
  // so we manually "type" the last letter
  await page.getByPlaceholder('Search wiki').dispatchEvent('keyup');
  // timeout is necessary because HTMX search could be slow
  await expect(page.locator('#wiki-search a[href]')).toBeInViewport({ratio: 1});
});
