// @watch start
// templates/repo/issue/view_content/**
// web_src/css/repo/issue-**
// web_src/js/features/repo-issue**
// @watch end

/* eslint playwright/expect-expect: ["error", { "assertFunctionNames": ["check_wip"] }] */

import {expect, type Page} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

test.describe('Pull: Toggle WIP', () => {
  const prTitle = 'pull5';

  async function toggle_wip_to({page}, should: boolean) {
    await page.waitForLoadState('domcontentloaded');
    if (should) {
      await page.getByText('Still in progress?').click();
    } else {
      await page.getByText('Ready for review?').click();
    }
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
    const response = await page.goto('/user2/repo1/pulls/5');
    expect(response?.status()).toBe(200); // Status OK
    // ensure original title
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(prTitle);
    await page.getByText('Save').click();
    await check_wip({page}, false);
  });

  test('simple toggle', async ({page}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
    await page.goto('/user2/repo1/pulls/5');
    // toggle to WIP
    await toggle_wip_to({page}, true);
    await check_wip({page}, true);
    // remove WIP
    await toggle_wip_to({page}, false);
    await check_wip({page}, false);
  });

  test('manual edit', async ({page}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
    await page.goto('/user2/repo1/pulls/5');
    // manually edit title to another prefix
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(`[WIP] ${prTitle}`);
    await page.getByText('Save').click();
    await check_wip({page}, true);
    // remove again
    await toggle_wip_to({page}, false);
    await check_wip({page}, false);
  });

  test('maximum title length', async ({page}, workerInfo) => {
    test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
    await page.goto('/user2/repo1/pulls/5');
    // check maximum title length is handled gracefully
    const maxLenStr = prTitle + 'a'.repeat(240);
    await page.locator('#issue-title-edit-show').click();
    await page.locator('#issue-title-editor input').fill(maxLenStr);
    await page.getByText('Save').click();
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

test('Issue: Labels', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');

  async function submitLabels({page}: { page: Page }) {
    const submitted = page.waitForResponse('/user2/repo1/issues/labels');
    await page.locator('textarea').first().click(); // close via unrelated element
    await submitted;
    await page.waitForLoadState();
  }

  // select label list in sidebar only
  const labelList = page.locator('.issue-content-right .labels-list a');
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  // restore initial state
  await page.locator('.select-label').click();
  const responsePromise = page.waitForResponse('/user2/repo1/issues/labels');
  await page.getByText('Clear labels').click();
  await responsePromise;
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

test('Issue: Assignees', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
  // select label list in sidebar only
  const assigneesList = page.locator('.issue-content-right .assignees.list .selected .item a');

  const response = await page.goto('/org3/repo3/issues/1');
  expect(response?.status()).toBe(200);
  // Clear all assignees
  await page.locator('.select-assignees-modify.dropdown').click();
  await page.locator('.select-assignees-modify.dropdown .no-select.item').click();
  await expect(assigneesList.filter({hasText: 'user2'})).toBeHidden();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(page.locator('.select-assign-me')).toBeVisible();

  // Assign other user (with searchbox)
  await page.locator('.select-assignees-modify.dropdown').click();
  await page.type('.select-assignees-modify .menu .search input', 'user4');
  await expect(page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user2'})).toBeHidden();
  await expect(page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'})).toBeVisible();
  await page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'}).click();
  await page.locator('.select-assignees-modify.dropdown').click();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeVisible();

  // remove user4
  await page.locator('.select-assignees-modify.dropdown').click();
  await page.locator('.select-assignees-modify .menu .item').filter({hasText: 'user4'}).click();
  await page.locator('.select-assignees-modify.dropdown').click();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeVisible();
  await expect(assigneesList.filter({hasText: 'user4'})).toBeHidden();

  // Test assign me
  await page.locator('.ui.assignees .select-assign-me').click();
  await expect(assigneesList.filter({hasText: 'user2'})).toBeVisible();
  await expect(page.locator('.ui.assignees.list .item.no-select')).toBeHidden();
});

test('New Issue: Assignees', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
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

  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  const selectedMilestone = page.locator('.issue-content-right .select-milestone.list');
  const milestoneDropdown = page.locator('.issue-content-right .select-milestone.dropdown');
  await expect(selectedMilestone).toContainText('No milestone');

  // Add milestone.
  await milestoneDropdown.click();
  await page.getByRole('option', {name: 'milestone1'}).click();
  await expect(selectedMilestone).toContainText('milestone1');
  await expect(page.locator('.timeline-item.event').last()).toContainText('user2 added this to the milestone1 milestone');

  // Clear milestone.
  await milestoneDropdown.click();
  await page.getByText('Clear milestone', {exact: true}).click();
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

test('Issue: Dependencies', async ({page}) => {
  const response = await page.goto('/user2/dependency-test/issues/103');
  expect(response?.status()).toBe(200);

  const depsBlock = page.locator('.issue-content-right .depending');
  await expect(depsBlock).toContainText('No dependencies set');

  const input = page.locator('.issue-content-right .depending .dropdown .search');
  const current = page.locator('.issue-content-right .depending input[name="newDependency"]');
  const menu = page.locator('.issue-content-right .depending .menu');
  const items = page.locator('.issue-content-right .depending .menu .item');
  await input.scrollIntoViewIfNeeded();
  await input.click();

  // Without query, it should show issues in the same repo, sorted by date, except current one.
  await expect(menu).toBeVisible();
  await expect(items).toHaveCount(2);
  await expect(items.first()).toContainText('second issue here');
  await expect(items.last()).toContainText('first issue here');
  await expect(items.last()).toHaveAttribute('data-value', '1001');

  // With query, it should search all repos, but show current repo issues first.
  await input.fill('seventh');
  await expect(items).toHaveCount(2); // there is an issue in user2/repo2 containing the word "seventh"
  await expect(items.first()).toHaveAttribute('data-value', '1002');

  // When entering an issue number, it should always show that one first, then all text matches.
  await input.fill('101');
  await expect(items.first()).toHaveAttribute('data-value', '1001');
  await expect(items.nth(1)).toHaveAttribute('data-value', '1002');

  // Should behave the same with a prefix
  await input.fill('#101');
  await expect(items.first()).toHaveAttribute('data-value', '1001');

  // Selecting an issue
  await items.first().click();
  await expect(current).toHaveValue('1001');

  // Dropdown filters out currently selected issue. Whether that's good UX is debatable, but test for now.
  await input.fill('#101');
  await expect(items.first()).toBeHidden();

  // Add dependency
  const link = page.locator('.issue-content-right .depending .dependency a.title');
  await page.locator('.issue-content-right .depending button').click();
  await expect(link).toHaveAttribute('href', '/user2/dependency-test/issues/101');

  // Remove dependency
  await page.locator('.issue-content-right .depending .delete-dependency-button').click();

  const modal = page.locator('.modal.remove-dependency');
  await expect(modal).toBeVisible();
  await expect(modal).toContainText('This will remove the dependency from this issue');
  await modal.locator('button.ok').click();

  await expect(depsBlock).toContainText('No dependencies set');
});
