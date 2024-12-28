// @watch start
// routers/web/user/**
// templates/shared/user/**
// web_src/js/features/common-global.js
// @watch end

/* eslint playwright/expect-expect: ["error", { "assertFunctionNames": ["hasNoWorkflowTriggerNotificationText", "verifyNoWorkflowsYetMessage"] }] */
import {save_visual, test} from '../_test-setup.ts';
import {DashboardPage} from '../ui/DashboardPage.ts';
import {expect} from '@playwright/test';
import {ActionsPage} from '../ui/ActionsPage.ts';

test.describe('Actions: authenticated user', () => {
  test.use({user: 'user2'});

  test('No ci status is present after server start', async ({page}, testInfo) => {
    test.skip(testInfo.project.name !== '', 'This test can only run once, has other tests ');

    const dashboard = new DashboardPage(page, testInfo);
    await dashboard.goto();
    await dashboard.searchFor('test_workflows');
    await expect(page.locator('.dashboard-repos .repo-owner-name-list > li:nth-child(1) > a:nth-child(2)')).toHaveCount(0);
  });

  test('workflow dispatch present', async ({page}, testInfo) => {
    const action = new ActionsPage(page, testInfo);
    await action.goto('user2', 'test_workflows');
    await action.clickWorkflowByName('test-dispatch.yml');
    await action.hasWorkflowTriggerNotificationText();
    await expect(action.workflowTriggerButton).toBeVisible();
    await expect(action.workflowTriggerMenu).toBeHidden();

    await test.step('Open workflow trigger menu', async () => {
      await action.workflowTriggerButton.click();
      await expect(action.workflowTriggerMenu).toBeVisible();
    });

    await save_visual(page);
  });

  test('workflow dispatch error: missing inputs', async ({page}, testInfo) => {
    const action = new ActionsPage(page, testInfo);
    await action.goto('user2', 'test_workflows', 'workflow=test-dispatch.yml&actor=0&status=0');
    await action.workflowTriggerButton.click();

    // Remove the required attribute so we can trigger the error message!
    await page.evaluate(() => {
      const elem = document.querySelector('input[name="inputs[string2]"]');
      elem?.removeAttribute('required');
    });

    await action.workflowMenuDispatchSubmit.click();
    await expect(page.getByText('Require value for input "String w/o. default".')).toBeVisible();
    await save_visual(page);
  });

  test('workflow dispatch success', async ({page}, testInfo) => {
    const action = new ActionsPage(page, testInfo);
    await action.goto('user2', 'test_workflows', 'workflow=test-dispatch.yml&actor=0&status=0');

    await action.workflowTriggerButton.click();
    await page.fill('input[name="inputs[string2]"]', testInfo.project.name);
    await save_visual(page);
    await action.workflowMenuDispatchSubmit.click();

    await expect(page.getByText('Workflow run was successfully requested.')).toBeVisible();

    await expect(page.locator('.run-list>:first-child .run-list-meta', {hasText: 'now'})).toBeVisible();
    await save_visual(page);
  });

  test('repo has no actions defined', async ({page}, testInfo) => {
    const action = new ActionsPage(page, testInfo);
    await action.goto('user2', 'diff-test');
    await action.verifyNoWorkflowsYetMessage();
  });
});

test.describe('Actions: anonymous user', () => {
  test.use({user: null});

  test('workflow dispatch box not available for unauthenticated users', async ({page}, testInfo) => {
    const action = new ActionsPage(page, testInfo);
    await action.goto('user2', 'test_workflows', 'workflow=test-dispatch.yml&actor=0&status=0');
    await action.hasNoWorkflowTriggerNotificationText();
  });
});
