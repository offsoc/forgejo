// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

const workflow_trigger_notification_text = 'This workflow has a workflow_dispatch event trigger.';

test('workflow dispatch present', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  /** @type {import('@playwright/test').Page} */
  const page = await context.newPage();

  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

  await expect(page.getByText(workflow_trigger_notification_text)).toBeVisible();

  const run_workflow_btn = page.locator('#workflow_dispatch_dropdown>button');
  await expect(run_workflow_btn).toBeVisible();

  const menu = page.locator('#workflow_dispatch_dropdown>.menu');
  await expect(menu).toBeHidden();
  await run_workflow_btn.click();
  await expect(menu).toBeVisible();
});

test('workflow dispatch error: missing inputs', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari; see https://codeberg.org/forgejo/forgejo/pulls/3334#issuecomment-2033383');

  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  /** @type {import('@playwright/test').Page} */
  const page = await context.newPage();

  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');
  await page.waitForLoadState('networkidle');

  await page.locator('#workflow_dispatch_dropdown>button').click();

  // Remove the required attribute so we can trigger the error message!
  await page.evaluate(() => {
    const elem = document.querySelector('input[name="inputs[string2]"]');
    elem?.removeAttribute('required');
  });

  await page.locator('#workflow-dispatch-submit').click();
  await page.waitForLoadState('networkidle');

  await expect(page.getByText('Require value for input "String w/o. default".')).toBeVisible();
});

test('workflow dispatch success', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari; see https://codeberg.org/forgejo/forgejo/pulls/3334#issuecomment-2033383');

  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  /** @type {import('@playwright/test').Page} */
  const page = await context.newPage();

  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');
  await page.waitForLoadState('networkidle');

  await page.locator('#workflow_dispatch_dropdown>button').click();

  await page.type('input[name="inputs[string2]"]', 'abc');
  await page.locator('#workflow-dispatch-submit').click();
  await page.waitForLoadState('networkidle');

  await expect(page.getByText('Workflow run was successfully requested.')).toBeVisible();

  await expect(page.locator('.run-list>:first-child .run-list-meta', {hasText: 'now'})).toBeVisible();
});

test('workflow dispatch box not available for unauthenticated users', async ({page}) => {
  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('body')).not.toContainText(workflow_trigger_notification_text);
});
