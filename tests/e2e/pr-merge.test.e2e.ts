// @watch start
// //todo
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual} from './utils_e2e.ts';
// import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test.describe('PR: merge', () => {
  test('with merge commit', async ({page}) => {
    // test.skip(project.name === 'Mobile Safari', 'Cannot get it to work - as usual');
    const response = await page.goto('/user2/pr-def-merge-style-merge');
    expect(response?.status()).toBe(200);

    // await validate_form({page}, 'fieldset');

    // verify header is new
    await expect(page.locator('h4')).toContainText('new'); //
    await page.locator('input[name="rule_name"]').fill('testrule'); //
    await save_visual(page);
  });
});
