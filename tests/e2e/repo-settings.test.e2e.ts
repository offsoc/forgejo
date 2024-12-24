// @watch start
// templates/webhook/shared-settings.tmpl
// templates/repo/settings/**
// web_src/css/{form,repo}.css
// web_src/css/modules/grid.css
// web_src/js/features/comp/WebHookEditor.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test, waitForClickAndResponse} from './_test-setup.ts';
import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test('repo webhook settings', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Please read before starting to fixing it https://codeberg.org/forgejo/forgejo/pulls/6362');
  const response = await page.goto('/user2/repo1/settings/hooks/forgejo/new');
  expect(response?.status()).toBe(200);

  await page.locator('input[name="events"][value="choose_events"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeVisible();

  // check accessibility including the custom events (now visible) part
  await validate_form({page}, 'fieldset');
  await save_visual(page);

  await page.locator('input[name="events"][value="push_only"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await page.locator('input[name="events"][value="send_everything"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await save_visual(page);
});

test.describe('repo branch protection settings', () => {
  test('form', async ({page}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Please read before starting to fixing it https://codeberg.org/forgejo/forgejo/pulls/6362');
    const response = await page.goto('/user2/repo1/settings/branches/edit');
    expect(response?.status()).toBe(200);

    await validate_form({page}, 'fieldset');

    // verify header is new
    await expect(page.locator('h4')).toContainText('new');
    await page.locator('input[name="rule_name"]').fill('testrule');
    await save_visual(page);
    await waitForClickAndResponse(page, page.getByText('Save rule'), '/user2/repo1/settings/branches');
    await save_visual(page);
    const editBtn = page.getByRole('link', {name: 'Edit'});
    await editBtn.scrollIntoViewIfNeeded();
    await waitForClickAndResponse(page, editBtn, '/user2/repo1/settings/branches/edit');
    await expect(page.locator('h4')).toContainText('Protection rules for branch');
    await save_visual(page);
  });

  // good first reason to split up the e2e into its own suites
  // this ensures a clean state as tests share same data state
  test.beforeEach(async ({page}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Please read before starting to fixing it https://codeberg.org/forgejo/forgejo/pulls/6362');

    await page.goto('/user2/repo1/settings/branches/', {waitUntil: 'domcontentloaded'});

    const exitingRule = page.getByText('Delete rule');
    if (await exitingRule.isVisible()) {
      await exitingRule.scrollIntoViewIfNeeded();
      await exitingRule.click();
      await waitForClickAndResponse(page, '.modals .actions .ok', '/user2/repo1/settings/branches');
    }
  });
});
