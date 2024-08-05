// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

async function assertSelectedLines(page, nums) {
  const pageAssertions = async () => {
    expect(
      await Promise.all((await page.locator('tr.active [data-line-number]').all()).map((line) => line.getAttribute('data-line-number'))),
    )
      .toStrictEqual(nums);

    // the first line selected has an action button
    if (nums.length > 0) await expect(page.locator(`#L${nums[0]} .code-line-button`)).toBeVisible();
  };

  await pageAssertions();

  // URL has the expected state
  expect(new URL(page.url()).hash)
    .toEqual(nums.length === 0 ? '' : nums.length === 1 ? `#L${nums[0]}` : `#L${nums[0]}-L${nums.at(-1)}`);

  // test selection restored from URL hash
  await page.reload();
  return pageAssertions();
}

test('Line Range Selection', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();

  const filePath = '/user2/repo1/src/branch/master/README.md?display=source';

  const response = await page.goto(filePath);
  await expect(response?.status()).toBe(200);

  await assertSelectedLines(page, []);
  await page.locator('span#L1').click();
  await assertSelectedLines(page, ['1']);
  await page.locator('span#L3').click({modifiers: ['Shift']});
  await assertSelectedLines(page, ['1', '2', '3']);
  await page.locator('span#L2').click();
  await assertSelectedLines(page, ['2']);
  await page.locator('span#L1').click({modifiers: ['Shift']});
  await assertSelectedLines(page, ['1', '2']);

  // out-of-bounds end line
  await page.goto(`${filePath}#L1-L100`);
  await assertSelectedLines(page, ['1', '2', '3']);
});
