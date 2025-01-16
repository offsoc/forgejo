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

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

const workflow_trigger_notification_text = 'This workflow has a workflow_dispatch event trigger.';
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

  test('dispatch success', async ({page}, testInfo) => {
    test.skip(testInfo.project.name === 'Mobile Safari', 'Flaky behaviour on mobile safari; see https://codeberg.org/forgejo/forgejo/pulls/3334#issuecomment-2033383');
    await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

    await page.locator('#workflow_dispatch_dropdown>button').click();

    await page.fill('input[name="inputs[string2]"]', 'abc');
    await save_visual(page);
    await page.locator('#workflow-dispatch-submit').click();

    await expect(page.getByText('Workflow run was successfully requested.')).toBeVisible();

    await expect(page.locator('.run-list>:first-child .run-list-meta', {hasText: 'now'})).toBeVisible();
    await save_visual(page);
  });
});

test('workflow dispatch box not available for unauthenticated users', async ({page}) => {
  await page.goto('/user2/test_workflows/actions?workflow=test-dispatch.yml&actor=0&status=0');

  await expect(page.locator('body')).not.toContainText(workflow_trigger_notification_text);
});
