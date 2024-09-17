// @ts-check
import {expect} from '@playwright/test';
import {test} from './utils_e2e.js';

test.describe('Headers permalinks', () => {
  for (const {target, title} of [
    {title: 'Top level heading 1.20.6-', target: 'top-level-heading-1-20-6-1'},
    {title: 'level 2 heading', target: 'level-2-heading'},
    {title: 'level 3 heading face with', target: 'level-3-heading'},
    {title: 'level 4 heading …', target: 'level-4-heading'},
    {title: 'level 5 heading!!!', target: 'level-5-heading'},
    {title: 'level 6 höadüng with---', target: 'level-6-höadüng-with'},
    {title: '油山', target: '油山'},
    {title: '-1..2...5-14 → 42', target: '1-1-2-5-14-42'},
  ]) {
    test(`check that #${target} focuses "${title}"`, async ({page}, workerInfo) => {
      const viewport = page.viewportSize();
      await page.setViewportSize({
        height: 400,
        width: viewport.width,
      });
      await page.goto(`/user2/markdown/src/branch/main/headings.md#${target}`);
      // I got some flakiness locally where the page would not scroll at all
      await page.waitForLoadState('domcontentloaded');
      // unfortunately, this is JavaScript hackery and no proper HTML,
      // so focus is not set and the whole thing not actually accessible
      // see https://codeberg.org/forgejo/forgejo/pulls/5203#issuecomment-2318447
      // await expect(page.getByRole('heading', {name: title})).toBeFocused(true);

      // I couldn't get these browsers to scroll to the element on page load
      if (!['Mobile Safari', 'Mobile Chrome', 'webkit'].includes(workerInfo.project.name)) {
        await expect(page.getByRole('heading', {name: title})).toBeInViewport({ratio: 1});
      }
      await page.getByRole('heading', {name: title}).getByRole('link').click();
      await expect(page.getByRole('heading', {name: title})).toBeInViewport({ratio: 0.97});
    });
  }
});
