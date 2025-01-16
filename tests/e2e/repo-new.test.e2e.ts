// @watch start
// templates/repo/create**.tmpl
// web_src/css/{form,repo}.css
// @watch end

import {expect} from '@playwright/test';
import {test, dynamic_id, save_visual} from './utils_e2e.ts';
import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test('New repo: invalid', async ({page}) => {
  const response = await page.goto('/repo/create');
  expect(response?.status()).toBe(200);
  // check that relevant form content is hidden or available
  await expect(page.getByRole('group', {name: 'Use a template You can select'}).getByRole('combobox')).toBeVisible();
  await expect(page.getByText('.gitignore Select .gitignore')).toBeHidden();
  await expect(page.getByText('Labels Select a label set')).toBeHidden();
  await validate_form({page}, 'fieldset');
  await save_visual(page);

  await page.getByLabel('Repository name').fill('*invalid');
  await page.getByRole('button', {name: 'Create repository'}).click();
  await expect(page.getByText('Repository name should contain only alphanumeric')).toBeVisible();
  await save_visual(page);
});

test('New repo: initialize', async ({page}, workerInfo) => {
  const response = await page.goto('/repo/create');
  expect(response?.status()).toBe(200);
  // check that relevant form content is hidden or available
  await expect(page.getByRole('group', {name: 'Use a template You can select'}).getByRole('combobox')).toBeVisible();
  await expect(page.getByText('.gitignore Select .gitignore')).toBeHidden();
  // fill initialization section
  await page.getByText('Start the Git history with').click();
  await page.getByText('Select .gitignore templates').click();
  await page.getByLabel('.gitignore Select .gitignore').fill('Go');
  await page.getByRole('option', {name: 'Go', exact: true}).click();
  await page.keyboard.press('Escape');
  await page.getByLabel('License Select a license file').click();
  await page.getByRole('option', {name: 'MIT', exact: true}).click();
  await page.keyboard.press('Escape');
  // add advanced settings
  await page.getByText('Click to expand').click();
  await page.getByPlaceholder('master').fill('main');
  await page.getByLabel('Make repository a template').check();

  await validate_form({page}, 'fieldset');
  await save_visual(page);
  const reponame = dynamic_id();
  await page.getByLabel('Repository name').fill(reponame);
  await page.getByRole('button', {name: 'Create repository'}).click();
  await expect(page.getByRole('link', {name: '.gitignore'})).toBeVisible();
  await expect(page.getByRole('link', {name: 'LICENSE', exact: true})).toBeVisible();
  if (!workerInfo.project.name.includes('Mobile')) {
    await expect(page.getByText('Template', {exact: true})).toBeVisible();
  }
  await save_visual(page);
});

test('New repo: initialize later', async ({page}) => {
  const response = await page.goto('/repo/create');
  expect(response?.status()).toBe(200);

  const reponame = dynamic_id();
  await page.getByLabel('Repository name').fill(reponame);
  await page.getByPlaceholder('Enter short description').fill(`Description for repo ${reponame}`);
  await page.getByText('Click to expand').click();
  await page.getByPlaceholder('master').fill('devbranch');
  await validate_form({page}, 'fieldset');
  await page.getByRole('button', {name: 'Create repository'}).click();
  expect(page.url()).toBe(`http://localhost:3003/user2/${reponame}`);
  await expect(page.getByRole('link', {name: 'New file'})).toBeVisible();
  await expect(page.getByRole('heading', {name: 'Creating a new repository on'})).toBeVisible();
  await save_visual(page);

  // add a README
  await page.getByRole('link', {name: 'New file'}).click();
  // wait for loading spinner to disappear
  // Otherwise, filling the filename might not populate the tree_path form field or preview tab
  // The editor has race conditions, likely related to https://codeberg.org/forgejo/forgejo/issues/3371
  await expect(page.locator('.is-loading')).toBeHidden();
  await page.locator('.view-lines').click();
  await page.keyboard.type('# Heading\n\nHello Forgejo!');
  await page.getByPlaceholder('Name your file…').fill('README.md');
  await expect(page.getByText('Preview')).toBeVisible();
  await page.getByPlaceholder('Add "<filename>"').fill('My first commit message');
  await page.getByRole('button', {name: 'Commit changes'}).click();
  expect(page.url()).toBe(`http://localhost:3003/user2/${reponame}/src/branch/devbranch/README.md`);
  await expect(page.getByRole('link', {name: 'My first commit message'})).toBeVisible();
  await expect(page.getByText('Hello Forgejo!')).toBeVisible();
  await save_visual(page);
});

test('New repo: from template', async ({page}, workerInfo) => {
  test.skip(['Mobile Safari', 'webkit'].includes(workerInfo.project.name), 'WebKit browsers seem to have CORS issues with localhost here.');
  const response = await page.goto('/repo/create');
  expect(response?.status()).toBe(200);

  const reponame = dynamic_id();
  await page.getByRole('group', {name: 'Use a template You can select'}).getByRole('combobox').click();
  await page.getByRole('option', {name: 'user27/template1'}).click();
  await page.getByText('Git content (Default branch)').click();
  await save_visual(page);
  await page.getByLabel('Repository name').fill(reponame);
  await page.getByRole('button', {name: 'Create repository'}).click();
  await expect(page.getByRole('link', {name: `${reponame}.log`})).toBeVisible();
  await save_visual(page);
});

test('New repo: label set', async ({page}) => {
  await page.goto('/repo/create');

  const reponame = dynamic_id();
  await page.getByText('Click to expand').click();
  await page.getByLabel('Labels Select a label set').click();
  await page.getByRole('option', {name: 'Advanced (Kind/Bug, Kind/'}).click();
  // close dropdown via unrelated click
  await page.getByText('You can select an existing').click();
  await save_visual(page);
  await page.getByLabel('Repository name').fill(reponame);
  await page.getByRole('button', {name: 'Create repository'}).click();
  await page.goto(`/user2/${reponame}/issues`);
  await page.getByRole('link', {name: 'Labels'}).click();
  await expect(page.getByText('Kind/Bug Something is not')).toBeVisible();
  await save_visual(page);
});
