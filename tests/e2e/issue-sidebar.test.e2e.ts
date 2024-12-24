// @watch start
// templates/repo/issue/view_content/**
// web_src/css/repo/issue-**
// web_src/js/features/repo-issue**
// @watch end

/* eslint playwright/expect-expect: ["error", { "assertFunctionNames": ["check_wip"] }] */

import {expect, type Page} from '@playwright/test';
import {save_visual, test, waitForClickAndResponse} from './_test-setup.ts';

const USR1_REPO_PULLS = '/user2/repo1/pulls/5';
const ORG3_REPO3_IS = '/org3/repo3/issues/1';
const USR2_REPO1_IS = '/user2/repo1/issues';
const USR2_REPO1_IS1 = `${USR2_REPO1_IS}/1`;

const SELECTOR_MILESTONE = '#milestone-section';
const SELECTOR_MILESTONE_DROPDOWN = `${SELECTOR_MILESTONE} .dropdown`;
const SELECTOR_MILESTONE_LIST = `${SELECTOR_MILESTONE} .list`;

test.use({user: 'user2'});

test.describe('Pull: Toggle WIP', () => {
  const prTitle = 'pull5';

  async function toggle_wip_to({page}, should: boolean) {
    const text: string = should ? 'Still in progress?' : 'Ready for review?';

    await page.waitForLoadState('domcontentloaded');
    await waitForClickAndResponse(page, page.getByText(text), USR1_REPO_PULLS);
  }

  async function check_wip({page}, is: boolean) {
    const elemTitle = 'h1';
    const stateLabel = '.issue-state-label';
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator(elemTitle)).toContainText(prTitle);
    await expect(page.locator(elemTitle)).toContainText('#5');
    if (is) {
      await expect(page.locator(elemTitle)).toContainText('WIP');
      await expect(page.locator(stateLabel)).toContainText('Draft');
    } else {
      await expect(page.locator(elemTitle)).not.toContainText('WIP');
      await expect(page.locator(stateLabel)).toContainText('Open');
    }
  }

  test.beforeEach(async ({page}) => {
    const response = await page.goto(USR1_REPO_PULLS);
    expect(response?.status()).toBe(200); // Status OK
    // ensure original title
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(prTitle);
    await waitForClickAndResponse(page, page.getByText('Save'), USR1_REPO_PULLS);
    await check_wip({page}, false);
  });

  test('simple toggle', async ({page}) => {
    await page.goto(USR1_REPO_PULLS);
    // toggle to WIP
    await toggle_wip_to({page}, true);
    await check_wip({page}, true);
    // remove WIP
    await toggle_wip_to({page}, false);
    await check_wip({page}, false);
  });

  test('manual edit', async ({page}) => {
    await page.goto(USR1_REPO_PULLS);
    // manually edit title to another prefix
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(`[WIP] ${prTitle}`);
    await waitForClickAndResponse(page, page.getByText('Save'), USR1_REPO_PULLS);
    await check_wip({page}, true);
    // remove again
    await toggle_wip_to({page}, false);
    await check_wip({page}, false);
  });

  test('maximum title length', async ({page}) => {
    await page.goto(USR1_REPO_PULLS);
    // check maximum title length is handled gracefully
    const maxLenStr = prTitle + 'a'.repeat(240);
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(maxLenStr);
    await waitForClickAndResponse(page, page.getByText('Save'), USR1_REPO_PULLS);
    await expect(page.locator('h1')).toContainText(maxLenStr);
    await check_wip({page}, false);
    await toggle_wip_to({page}, true);
    await check_wip({page}, true);
    await expect(page.locator('h1')).toContainText(maxLenStr);
    await toggle_wip_to({page}, false);
    await check_wip({page}, false);
    await expect(page.locator('h1')).toContainText(maxLenStr);
  });
});
test('Issue: Labels', async ({page}) => {
  async function submitLabels({page}: { page: Page }) {
    await waitForClickAndResponse(page, page.locator('textarea').first(), USR2_REPO1_IS1);
  }

  // select label list in sidebar only
  const labelList = page.locator('.issue-content-right .labels-list a');
  const response = await page.goto(USR2_REPO1_IS1);
  expect(response?.status()).toBe(200);

  // restore initial state
  await page.locator('.select-label').click();
  await waitForClickAndResponse(page, page.locator('.select-label .menu .no-select.item'), '/user2/repo1/issues/labels');
  await expect(labelList.filter({hasText: 'label1'})).toBeHidden();
  await expect(labelList.filter({hasText: 'label2'})).toBeHidden();

  // add both labels
  await page.locator('.select-label').click();
  // label search could be tested this way:
  // await page.locator('.select-label input').fill('label2');
  await page.locator('.select-label .item').filter({hasText: 'label2'}).click();
  await page.locator('.select-label .item').filter({hasText: 'label1'}).click();
  await submitLabels({page});
  await expect(labelList.filter({hasText: 'label2'})).toBeVisible();
  await expect(labelList.filter({hasText: 'label1'})).toBeVisible();

  // test removing label2 again
  // due to a race condition, the page could still be "reloading",
  // closing the dropdown after it was clicked.
  // Retry the interaction as a group
  // also see https://playwright.dev/docs/test-assertions#expecttopass
  await expect(async () => {
    await page.locator('.select-label').click();
    await page.locator('.select-label .item').filter({hasText: 'label2'}).click();
  }).toPass();
  await submitLabels({page});
  await expect(labelList.filter({hasText: 'label2'})).toBeHidden();
  await expect(labelList.filter({hasText: 'label1'})).toBeVisible();
});

test('Issue: Assignees', async ({page}) => {
  // select label list in sidebar only
  const assigneesList = page.locator('.issue-content-right .assignees.list .selected .item a');

  const response = await page.goto(ORG3_REPO3_IS);
  expect(response?.status()).toBe(200);
  // Clear all assignees
  await page.locator('.select-assignees-modify.dropdown').click();
  await waitForClickAndResponse(page, '.select-assignees-modify.dropdown .no-select.item', ORG3_REPO3_IS);
  await expect(assigneesList.filter({hasText: 'user2'})).toBeHidden();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(page.locator('.select-assign-me')).toBeVisible();

  // Assign other user (with searchbox)
  await page.locator('.select-assignees-modify.dropdown').click();
  await page.fill('.select-assignees-modify .menu .search input', 'user4');
  await expect(page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user2'})).toBeHidden();
  await expect(page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'})).toBeVisible();
  await page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'}).click();
  await waitForClickAndResponse(page, '.select-assignees-modify.dropdown', ORG3_REPO3_IS);
  await expect(assigneesList.filter({hasText: 'user4'})).toBeVisible();

  // remove user4
  await page.locator('.select-assignees-modify.dropdown').click();
  await page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'}).click();
  await waitForClickAndResponse(page, '.select-assignees-modify.dropdown', ORG3_REPO3_IS);
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();

  // Test assign me
  await waitForClickAndResponse(page, '.ui.assignees .select-assign-me', ORG3_REPO3_IS);
  await expect(assigneesList.filter({hasText: 'user2'})).toBeVisible();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeHidden();
});

test('New Issue: Assignees', async ({page}) => {
  // select label list in sidebar only
  const assigneesList = page.locator('.issue-content-right .assignees.list .selected .item');

  const response = await page.goto('/org3/repo3/issues/new');
  expect(response?.status()).toBe(200);
  // preconditions
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(assigneesList.filter({hasText: 'user2'})).toBeHidden();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();

  // Assign other user (with searchbox)
  await page.locator('.select-assignees.dropdown').click();
  await page.fill('.select-assignees .menu .search input', 'user4');
  await expect(page.locator('.select-assignees .menu .item').filter({hasText: 'user2'})).toBeHidden();
  await expect(page.locator('.select-assignees .menu .item').filter({hasText: 'user4'})).toBeVisible();
  await page.locator('.select-assignees .menu .item').filter({hasText: 'user4'}).click();
  await page.locator('.select-assignees.dropdown').click();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeVisible();
  await save_visual(page);

  // remove user4
  await page.locator('.select-assignees.dropdown').click();
  await page.locator('.select-assignees .menu .item').filter({hasText: 'user4'}).click();
  await page.locator('.select-assignees.dropdown').click();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();

  // Test assign me
  await page.locator('.ui.assignees .select-assign-me').click();
  await expect(assigneesList.filter({hasText: 'user2'})).toBeVisible();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeHidden();

  await page.locator('.select-assignees.dropdown').click();
  await page.fill('.select-assignees .menu .search input', '');
  await page.locator('.select-assignees.dropdown .no-select.item').click();
  await expect(page.locator('.select-assign-me')).toBeVisible();
  await save_visual(page);
});

test('Issue: Milestone', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
  const response = await page.goto(USR2_REPO1_IS1);
  expect(response?.status()).toBe(200);

  const selectedMilestone = page.locator(SELECTOR_MILESTONE_LIST);
  const milestoneDropdown = page.locator(SELECTOR_MILESTONE_DROPDOWN);
  await expect(selectedMilestone.locator('.no-select')).toBeVisible();
  await expect(selectedMilestone.locator('.sidebar-item-link')).toHaveCount(0);

  // Add milestone.
  await expect(milestoneDropdown.getByRole('listbox')).toBeHidden();
  await milestoneDropdown.click();
  await expect(milestoneDropdown.getByRole('listbox')).toBeVisible();
  await waitForClickAndResponse(page, milestoneDropdown.getByRole('option', {name: 'milestone1'}), USR2_REPO1_IS);
  await expect(selectedMilestone.locator('.no-select')).toBeHidden();
  await expect(selectedMilestone.locator('.sidebar-item-link')).toHaveCount(1);
  await expect(selectedMilestone.locator('.sidebar-item-link')).toContainText('milestone1');
  await expect(page.locator('.timeline-item.event').last()).toContainText('user2 added this to the milestone1 milestone');

  // Clear milestone.
  await expect(milestoneDropdown.locator('.menu')).toBeHidden();
  await milestoneDropdown.scrollIntoViewIfNeeded();
  await milestoneDropdown.click();

  const menu = milestoneDropdown.locator('.menu');
  await expect(menu).toBeVisible();

  await waitForClickAndResponse(page, milestoneDropdown.locator('div.no-select'), USR2_REPO1_IS);
  await expect(selectedMilestone).toContainText('No milestone');
  await expect(page.locator('.timeline-item.event').last()).toContainText('user2 removed this from the milestone1 milestone');
});

test('New Issue: Milestone', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');

  const response = await page.goto('/user2/repo1/issues/new');
  expect(response?.status()).toBe(200);

  const selectedMilestone = page.locator('.issue-content-right .select-milestone.list');
  const milestoneDropdown = page.locator('.issue-content-right .select-milestone.dropdown');
  await expect(selectedMilestone).toContainText('No milestone');
  await save_visual(page);

  // Add milestone.
  await milestoneDropdown.click();
  await page.getByRole('option', {name: 'milestone1'}).click();
  await expect(selectedMilestone).toContainText('milestone1');
  await save_visual(page);

  // Clear milestone.
  await milestoneDropdown.click();
  await page.getByText('Clear milestone', {exact: true}).click();
  await expect(selectedMilestone).toContainText('No milestone');
  await save_visual(page);
});
