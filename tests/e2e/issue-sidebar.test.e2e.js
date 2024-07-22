// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

async function login({browser}, workerInfo) {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  return await context.newPage();
}

// belongs to test: Pull: Toggle WIP
const prTitle = 'pull5';

async function click_toggle_wip({page}) {
  await page.locator('.toggle-wip>a').click();
  await page.waitForLoadState('networkidle');
}

async function check_wip({page}, is) {
  const elemTitle = '#issue-title-display';
  const stateLabel = '.issue-state-label';
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

test('Pull: Toggle WIP', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
  const page = await login({browser}, workerInfo);
  const response = await page.goto('/user2/repo1/pulls/5');
  await expect(response?.status()).toBe(200); // Status OK
  // initial state
  await check_wip({page}, false);
  // toggle to WIP
  await click_toggle_wip({page});
  await check_wip({page}, true);
  // remove WIP
  await click_toggle_wip({page});
  await check_wip({page}, false);

  // manually edit title to another prefix
  await page.locator('#issue-title-edit-show').click();
  await page.locator('#issue-title-editor input').fill(`[WIP] ${prTitle}`);
  await page.getByText('Save').click();
  await page.waitForLoadState('networkidle');
  await check_wip({page}, true);
  // remove again
  await click_toggle_wip({page});
  await check_wip({page}, false);
});

test('Issue: Labels', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
  const page = await login({browser}, workerInfo);
  // select label list in sidebar only
  const labelList = page.locator('.issue-content-right .labels-list a');
  const response = await page.goto('/user2/repo1/issues/1');
  await expect(response?.status()).toBe(200);
  // preconditions
  await expect(labelList.filter({hasText: 'label1'})).toBeVisible();
  await expect(labelList.filter({hasText: 'label2'})).toBeHidden();
  // add label2
  await page.locator('.select-label').click();
  // label search could be tested this way:
  // await page.locator('.select-label input').fill('label2');
  await page.locator('.select-label .item').filter({hasText: 'label2'}).click();
  await page.locator('.select-label').click();
  await page.waitForLoadState('networkidle');
  await expect(labelList.filter({hasText: 'label2'})).toBeVisible();
  // test removing label again
  await page.locator('.select-label').click();
  await page.locator('.select-label .item').filter({hasText: 'label2'}).click();
  await page.locator('.select-label').click();
  await page.waitForLoadState('networkidle');
  await expect(labelList.filter({hasText: 'label2'})).toBeHidden();
  await expect(labelList.filter({hasText: 'label1'})).toBeVisible();
});

test('Issue: Milestone', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Unable to get tests working on Safari Mobile, see https://codeberg.org/forgejo/forgejo/pulls/3445#issuecomment-1789636');
  const page = await login({browser}, workerInfo);

  const response = await page.goto('/user2/repo1/issues/1');
  await expect(response?.status()).toBe(200);

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
