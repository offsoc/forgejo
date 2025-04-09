// @watch start
// templates/repo/actions/**
// web_src/css/actions.css
// web_src/js/components/ActionRunStatus.vue
// web_src/js/components/RepoActionView.vue
// modules/actions/**
// modules/structs/workflow.go
// routers/api/v1/repo/action.go
// routers/web/repo/actions/**
// @watch end

import {expect, type Page, type TestInfo} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

const workflow_trigger_notification_text = 'This workflow has a workflow_dispatch event trigger.';

async function dispatchSuccess(page: Page, testInfo: TestInfo) {
  test.skip(testInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari; see https://codeberg.org/forgejo/forgejo/pulls/3334#issuecomment-2033383');
  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

  await page.locator('#workflow_dispatch_dropdown>button').click();

  await page.fill('input[name="inputs[string2]"]', 'abc');
  await save_visual(page);
  await page.locator('#workflow-dispatch-submit').click();

  await expect(page.getByText('Workflow run was successfully requested.')).toBeVisible();

  await expect(page.locator('.run-list>:first-child .run-list-meta', {hasText: 'now'})).toBeVisible();
  await save_visual(page);
}

test.describe('Workflow Authenticated user2', () => {
  test.use({user: 'user2'});

  test('workflow dispatch present', async ({page}) => {
    await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    await expect(page.getByText(workflow_trigger_notification_text)).toBeVisible();

    const run_workflow_btn = page.locator('#workflow_dispatch_dropdown>button');
    await expect(run_workflow_btn).toBeVisible();

    const menu = page.locator('#workflow_dispatch_dropdown>.menu');
    await expect(menu).toBeHidden();
    await run_workflow_btn.click();
    await expect(menu).toBeVisible();
    await save_visual(page);
  });

  test('dispatch error: missing inputs', async ({page}, testInfo) => {
    test.skip(testInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari; see https://codeberg.org/forgejo/forgejo/pulls/3334#issuecomment-2033383');

    await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    await page.locator('#workflow_dispatch_dropdown>button').click();

    // Remove the required attribute so we can trigger the error message!
    await page.evaluate(() => {
      const elem = document.querySelector('input[name="inputs[string2]"]');
      elem?.removeAttribute('required');
    });

    await page.locator('#workflow-dispatch-submit').click();

    await expect(page.getByText('Require value for input "String w/o. default".')).toBeVisible();
    await save_visual(page);
  });

  // no assertions as the login in this test case is extracted for reuse
  // eslint-disable-next-line playwright/expect-expect
  test('dispatch success', async ({page}, testInfo) => {
    await dispatchSuccess(page, testInfo);
  });
});

test('workflow dispatch box not available for unauthenticated users', async ({page}) => {
  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

  await expect(page.locator('body')).not.toContainText(workflow_trigger_notification_text);
  await save_visual(page);
});

async function completeDynamicRefresh(page: Page) {
  // Ensure that the reloading indicator isn't active, indicating that dynamic refresh is done.
  await expect(page.locator('#reloading-indicator')).not.toHaveClass(/(^|\s)is-loading(\s|$)/);
}

async function simulatePollingInterval(page: Page) {
  // In order to simulate the background page sitting around for > 30s, a custom event `simulate-polling-interval` is
  // fired into the document to mimic the polling interval expiring -- although this isn't a perfectly great E2E test
  // with this kind of mimicry, it's better than having multiple >30s execution-time tests.
  await page.evaluate(() => {
    document.dispatchEvent(new Event('simulate-polling-interval'));
  });
  await completeDynamicRefresh(page);
}

test.describe('workflow list dynamic refresh', () => {
  test.use({user: 'user2'});

  test('refreshes on visibility change', async ({page}, testInfo) => {
    // Test operates by creating two pages; one which is sitting idle on the workflows list (backgroundPage), and one
    // which triggers a workflow dispatch.  Then a document visibilitychange event is fired on the background page to
    // mimic a user returning to the tab on their browser, which should trigger the workflow list to refresh and display
    // the newly dispatched workflow from the other page.

    const backgroundPage = await page.context().newPage();
    await backgroundPage.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    // Mirror the `Workflow Authenticated user2 > dispatch success` test:
    await dispatchSuccess(page, testInfo);
    const latestDispatchedRun = await page.locator('.run-list>:first-child .flex-item-body>b').textContent();
    expect(latestDispatchedRun).toMatch(/^#/); // workflow ID, eg. "#53"

    // Synthetically trigger a visibilitychange event, as if we were returning to backgroundPage:
    await backgroundPage.evaluate(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });
    await completeDynamicRefresh(page);
    await expect(backgroundPage.locator('.run-list>:first-child .flex-item-body>b', {hasText: latestDispatchedRun})).toBeVisible();
    await save_visual(backgroundPage);
  });

  test('refreshes on interval', async ({page}, testInfo) => {
    // Test operates by creating two pages; one which is sitting idle on the workflows list (backgroundPage), and one
    // which triggers a workflow dispatch.  After the polling, the page should refresh and show the newly dispatched
    // workflow from the other page.

    const backgroundPage = await page.context().newPage();
    await backgroundPage.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    // Mirror the `Workflow Authenticated user2 > dispatch success` test:
    await dispatchSuccess(page, testInfo);
    const latestDispatchedRun = await page.locator('.run-list>:first-child .flex-item-body>b').textContent();
    expect(latestDispatchedRun).toMatch(/^#/); // workflow ID, eg. "#53"

    await simulatePollingInterval(backgroundPage);
    await expect(backgroundPage.locator('.run-list>:first-child .flex-item-body>b', {hasText: latestDispatchedRun})).toBeVisible();
    await save_visual(backgroundPage);
  });

  test('post-refresh the dropdowns continue to operate', async ({page}, testInfo) => {
    // Verify that after the page is dynamically refreshed, the 'Actor', 'Status', and 'Run workflow' dropdowns work
    // correctly -- that the htmx morph hasn't messed up any JS event handlers.
    await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    // Mirror the `Workflow Authenticated user2 > dispatch success` test -- this creates data for the 'Actor' dropdown
    await dispatchSuccess(page, testInfo);

    // Perform a dynamic refresh before checking the functionality of each dropdown.
    await simulatePollingInterval(page);

    // Workflow run dialog
    await expect(page.locator('input[name="inputs[string2]"]')).toBeHidden();
    await page.locator('#workflow_dispatch_dropdown>button').click();
    await expect(page.locator('input[name="inputs[string2]"]')).toBeVisible();
    await page.locator('#workflow_dispatch_dropdown>button').click();

    // Status dropdown
    await expect(page.getByText('Waiting')).toBeHidden();
    await expect(page.getByText('Failure')).toBeHidden();
    await page.locator('#status_dropdown').click();
    await expect(page.getByText('Waiting')).toBeVisible();
    await expect(page.getByText('Failure')).toBeVisible();

    // Actor dropdown
    await expect(page.getByText('All actors')).toBeHidden();
    await page.locator('#actor_dropdown').click();
    await expect(page.getByText('All Actors')).toBeVisible();
  });

  test('refresh does not break interacting with open drop-downs', async ({page}, testInfo) => {
    // Verify that if the polling refresh occurs while interacting with any multi-step dropdown on the page, the
    // multi-step interaction continues to be visible and functional.  This is implemented by preventing the refresh,
    // but that isn't the subject of the test here -- as long as the dropdown isn't broken by the refresh, that's fine.
    await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    // Mirror the `Workflow Authenticated user2 > dispatch success` test -- this creates data for the 'Actor' dropdown
    await dispatchSuccess(page, testInfo);

    // Workflow run dialog
    await expect(page.locator('input[name="inputs[string2]"]')).toBeHidden();
    await page.locator('#workflow_dispatch_dropdown>button').click();
    await expect(page.locator('input[name="inputs[string2]"]')).toBeVisible();
    await simulatePollingInterval(page);
    await expect(page.locator('input[name="inputs[string2]"]')).toBeVisible();

    // Status dropdown
    await expect(page.getByText('Waiting')).toBeHidden();
    await expect(page.getByText('Failure')).toBeHidden();
    await page.locator('#status_dropdown').click();
    await expect(page.getByText('Waiting')).toBeVisible();
    await expect(page.getByText('Failure')).toBeVisible();
    await simulatePollingInterval(page);
    await expect(page.getByText('Waiting')).toBeVisible();
    await expect(page.getByText('Failure')).toBeVisible();

    // Actor dropdown
    await expect(page.getByText('All actors')).toBeHidden();
    await page.locator('#actor_dropdown').click();
    await expect(page.getByText('All Actors')).toBeVisible();
    await simulatePollingInterval(page);
    await expect(page.getByText('All Actors')).toBeVisible();
  });
});
