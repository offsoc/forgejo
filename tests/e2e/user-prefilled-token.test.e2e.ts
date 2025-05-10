// @watch start
// templates/user/settings/**.tmpl
// web_src/css/{form,user}.css
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual, login_user, login} from './utils_e2e.ts';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('User: Pre-filled Token name only', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);

  const token_name = 'My Token';
  await page.goto(encodeURI(`/user/settings/applications?token_name=${token_name}`));

  await save_visual(page);

  await expect(page.getByLabel('Token name')).toHaveValue(token_name);

  await expect(page.getByText('Review the following preset scope carefully')).toBeHidden();
});

test('User: Pre-filled Token name and scope', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);

  const token_name = 'My Token';
  const scope_user = 'read';
  const scope_repository = 'read';
  const scope_issue = 'write';
  await page.goto(encodeURI(`/user/settings/applications?token_name=${token_name}&user=${scope_user}&repository=${scope_repository}&issue=${scope_issue}`));

  await save_visual(page);

  await expect(page.getByLabel('Token name')).toHaveValue(token_name);
  await expect(page.getByLabel('user')).toHaveValue(`${scope_user}:user`);
  await expect(page.getByLabel('user')).toBeVisible();
  await expect(page.getByLabel('repository')).toHaveValue(`${scope_repository}:repository`);
  await expect(page.getByLabel('repository')).toBeVisible();
  await expect(page.getByLabel('issue')).toHaveValue(`${scope_issue}:issue`);
  await expect(page.getByLabel('issue')).toBeVisible();

  await expect(page.getByText('Review the following preset scope carefully')).toBeVisible();
});
