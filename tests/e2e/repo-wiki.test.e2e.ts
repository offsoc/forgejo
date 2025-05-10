// @watch start
// templates/repo/wiki/**
// web_src/css/repo**
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

for (const searchTerm of ['space', 'consectetur']) {
  for (const width of [null, 2560, 4000]) {
    test(`Search for '${searchTerm}' and test for no overflow ${width && `on ${width}-wide viewport` || ''}`, async ({page, viewport}, workerInfo) => {
      test.skip(workerInfo.project.name === 'Mobile Safari', 'Fails as always, see https://codeberg.org/forgejo/forgejo/pulls/5326#issuecomment-2313275');

      await page.setViewportSize({
        width: width ?? viewport.width,
        height: 1440, // We're testing that we fit horizontally - vertical scrolling is fine.
      });
      await page.goto('/user2/repo1/wiki');
      await page.getByPlaceholder('Search wiki').fill(searchTerm);
      await page.getByPlaceholder('Search wiki').click();
      // workaround: HTMX listens on keyup events, playwright's fill only triggers the input event
      // so we manually "type" the last letter
      await page.getByPlaceholder('Search wiki').dispatchEvent('keyup');

      await expect(page.locator('#wiki-search a[href]')).toBeInViewport({
        ratio: workerInfo.project.name === 'webkit' ? 0.9 : 1,
      });
      await save_visual(page);
    });
  }
}

test(`Search results show titles (and not file names)`, async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Fails as always, see https://codeberg.org/forgejo/forgejo/pulls/5326#issuecomment-2313275');
  await page.goto('/user2/repo1/wiki');
  await page.getByPlaceholder('Search wiki').fill('spaces');
  await page.getByPlaceholder('Search wiki').click();
  // workaround: HTMX listens on keyup events, playwright's fill only triggers the input event
  // so we manually "type" the last letter
  await page.getByPlaceholder('Search wiki').dispatchEvent('keyup');
  await expect(page.locator('#wiki-search a[href] b')).toHaveText('Page With Spaced Name');
  await save_visual(page);
});
