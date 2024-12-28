import {
  type Browser,
  type BrowserContextOptions,
  expect,
  type Locator,
  type Page,
  test as baseTest,
  type WorkerInfo,
} from '@playwright/test';
import * as path from 'node:path';

const AUTH_PATH = 'tests/e2e/.auth';

type AuthScope = 'logout' | 'shared' | 'webauthn';

export type TestOptions = {
  forEachTest: void
  user: string | null;
  authScope: AuthScope;
};

export const test = baseTest.extend<TestOptions>({
  context: async ({browser, user, authScope, contextOptions}, use, {project}) => {
    if (user && authScope) {
      const browserName = project.name.toLowerCase().replace(' ', '-');
      contextOptions.storageState = path.join(AUTH_PATH, `state-${browserName}-${user}-${authScope}.json`);
    } else {
      // if no user is given, ensure to have clean state
      contextOptions.storageState = {cookies: [], origins: []};
    }

    return use(await test_context(browser, contextOptions));
  },
  user: null,
  authScope: 'shared',
  // see https://playwright.dev/docs/test-fixtures#adding-global-beforeeachaftereach-hooks
  forEachTest: [async ({page}, use) => {
    await use();
    // some tests create a new page which is not yet available here
    // only operate on tests that make the URL available
    if (page.url() !== 'about:blank') {
      await save_visual(page);
    }
  }, {auto: true}],
});

export async function test_context(browser: Browser, options?: BrowserContextOptions) {
  const context = await browser.newContext(options);

  context.on('page', (page) => {
    page.on('pageerror', (err) => expect(err).toBeUndefined());
  });

  return context;
}

export async function save_visual(page: Page) {
  // Optionally include visual testing
  if (process.env.VISUAL_TEST) {
    await page.waitForLoadState('domcontentloaded');
    // Mock/replace dynamic content which can have different size (and thus cannot simply be masked below)
    await page.locator('footer .left-links').evaluate((node) => node.innerHTML = 'MOCK');
    // replace timestamps in repos to mask them later down
    await page.locator('.flex-item-body > relative-time').filter({hasText: /now|minute/}).evaluateAll((nodes) => {
      for (const node of nodes) node.outerHTML = 'relative time in repo';
    });
    await page.locator('relative-time').evaluateAll((nodes) => {
      for (const node of nodes) node.outerHTML = 'time element';
    });
    // used for instance for security keys
    await page.locator('absolute-date').evaluateAll((nodes) => {
      for (const node of nodes) node.outerHTML = 'time element';
    });
    await expect(page).toHaveScreenshot({
      fullPage: true,
      timeout: 20000,
      mask: [
        page.locator('.ui.avatar'),
        page.locator('.sha'),
        page.locator('#repo_migrating'),
        // update order of recently created repos is not fully deterministic
        page.locator('.flex-item-main').filter({hasText: 'relative time in repo'}),
      ],
    });
  }
}

/**
 * Maps key inputs to specific platform and browser engine adjustments.
 * @param key - The name of the key to map (e.g., 'Tab', 'End').
 * @param workerInfo - Information about the current test worker, including project details like browser engine and platform.
 * @returns The adjusted key mapping based on the platform and browser engine.
 */
export const adjustKeyMapping = (key: string, workerInfo: WorkerInfo): string => {
  const isOsDarwin = process.platform === 'darwin';
  const isEngineWebKit = ['Mobile Safari', 'webkit'].includes(workerInfo.project.name);

  switch (key) {
    case 'Tab': {
      // Adjust Tab key mapping for macOS with WebKit browsers
      // when "Press tab to highlight each item on a webpage" is not enabled.
      // Ref: https://github.com/microsoft/playwright/issues/5609
      return isOsDarwin && isEngineWebKit ? 'Alt+Tab' : 'Tab';
    }
    case 'End': {
      // Adjust End key mapping for macOS.
      return isOsDarwin ? 'Alt+ArrowRight' : 'End';
    }
  }

  return key;
};

/**
 * Waits for a specific network response and ensures the page is idle before proceeding.
 * @param page - The Playwright Page instance.
 * @param locator - The locator for the element to click.
 * @param expectedUrlSubstring - Partial URL string to match the desired response.
 */
export const waitForClickAndResponse = async (page: Page, locator: Locator | string, expectedUrlSubstring: string) => {
  const element: Locator = typeof locator === 'string' ? page.locator(locator) : locator;

  await element.scrollIntoViewIfNeeded();

  await Promise.all([
    page.waitForResponse((response) => response.url().includes(expectedUrlSubstring) && response.ok()),
    element.click(),
  ]);

  await page.waitForLoadState('domcontentloaded');
};
