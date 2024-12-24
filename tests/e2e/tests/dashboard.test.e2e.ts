// @watch start
// routers/web/user/**
// templates/shared/user/**
// web_src/js/features/common-global.js
// @watch end

import {save_visual, test} from '../_test-setup.ts';
import {DashboardPage} from '../ui/DashboardPage.ts';
import {expect} from '@playwright/test';
import {RepositoryOverviewPage} from '../ui/RepositoryOverviewPage.ts';

/* eslint playwright/expect-expect: ["error", { "assertFunctionNames": ["repositoriesCount", "verifyCanUnfollow"] }] */

test.describe('Interactive with repository searxh', () => {
  test.use({user: 'user2'});

  for (const {query, count} of [
    {query: 'test', count: 7},
    {query: 'test_workflow', count: 1},
    {query: 'thisStringWillNeverMatch', count: 0},
    {query: '', count: 15},
  ]) {
    test(`Search for "${query}" with expected count of ${count} entries in list`, async ({page}, testInfo) => {
      const dashboard = new DashboardPage(page, testInfo);
      await dashboard.goto();
      await dashboard.searchFor(query);
      await dashboard.repositoriesCount(count);
    });
  }

  test(`Search query length is limit to 255 chars`, async ({page}, testInfo) => {
    const longString = 'a'.repeat(300);
    const maxString = longString.substring(0, 255);

    const dashboard = new DashboardPage(page, testInfo);
    await dashboard.goto();

    await dashboard.search.fill(longString);

    await expect(dashboard.search).toHaveValue(maxString);
  });

  test('Navigate to git_hooks_test repository', async ({page}, testInfo) => {
    const repoName = 'git_hooks_test';

    await test.step(`Open dashboard and search for ${repoName}`, async () => {
      const dashboard = new DashboardPage(page, testInfo);
      await dashboard.goto();
      await dashboard.searchFor(repoName);

      await Promise.all([
        page.waitForResponse((response) => response.url().includes(repoName) && response.ok()),
        dashboard.repositoryListEntries.getByText(repoName).click(),
      ]);
    });

    await test.step(`Navigate to ${repoName}`, async () => {
      const repo = new RepositoryOverviewPage(page, testInfo);
      await expect(repo.repoHeader.getByRole('link', {name: repoName})).toBeVisible();
    });
  });
});

test.describe('Dashboard', () => {
  test.use({user: 'user2'});

  test.describe.configure({retries: 1});

  test('Dashboard has ci status', async ({page}, testInfo) => {
    // TODO: optimize fixtures
    if (testInfo.retry) {
      await page.goto('/user2/test_workflows/actions');
    }

    const repoName = 'test_workflows';
    const dashboard = new DashboardPage(page, testInfo);
    await dashboard.goto();
    await dashboard.searchFor(repoName);

    const repoStatus = page.locator('.dashboard-repos .repo-owner-name-list > li:nth-child(1) > a:nth-child(2)');
    await expect(repoStatus).toHaveAttribute('href', '/user2/test_workflows/actions');
    await expect(repoStatus).toHaveAttribute('data-tooltip-content', /^(Error|Failure)$/);
    await save_visual(page);
  });
});

test.describe('Dashboard as anonymous', () => {
  // eslint-disable-next-line playwright/no-skipped-test
  test.describe.skip('example with different viewports (not actually run)', () => {
    // only necessary when the default web / mobile devices are not enough.
    // If you need to use a single fixed viewport, you can also use:
    // test.use({viewport: {width: 400, height: 800}});
    // also see https://playwright.dev/docs/test-parameterize
    for (const width of [400, 1000]) {
      // do not actually run (skip) this test
      test(`Do x on width: ${width}px`, async ({page}) => {
        await page.setViewportSize({
          width,
          height: 800,
        });
        // do something, then check that an element is fully in viewport
        // (i.e. not overflowing)
        await expect(page.locator('#my-element')).toBeInViewport({ratio: 1});
      });
    }
  });

  test('Landing Page', async ({page}, testInfo) => {
    const dashboard = new DashboardPage(page, testInfo);
    await dashboard.goto();
    await expect(page).toHaveTitle(/^Forgejo: Beyond coding. We Forge.\s*$/);
    await expect(page.locator('.logo')).toHaveAttribute('src', '/assets/img/logo.svg');
  });
});
