import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class ForgejoUi {
  protected readonly page: Page;
  protected readonly testInfo: TestInfo;
  readonly flashMessage: Locator;

  constructor(page: Page, testInfo: TestInfo) {
    this.page = page;
    this.testInfo = testInfo;

    this.flashMessage = page.locator('#flash-message');
  }

  async flashMessageToContain(text: string) {
    await expect(this.flashMessage).toBeVisible();
    await expect(this.flashMessage).toContainText(text);
  }

  /**
   * Waits for a specific network response and ensures the page is idle before proceeding.
   * @param locator - The locator for the element to click.
   * @param expectedUrlSubstring - Partial URL string to match the desired response.
   */
  async clickAndWaitForNetworkResponse(locator: Locator, expectedUrlSubstring: string) {
    await locator.scrollIntoViewIfNeeded();

    await Promise.all([
      this.page.waitForResponse((response) => response.url().includes(expectedUrlSubstring) && response.ok()),
      locator.click(),
    ]);

    await this.page.waitForLoadState('domcontentloaded');
  }

  async fillAndValidate(locator: Locator, value: string, defaultValue: string = '') {
    await locator.scrollIntoViewIfNeeded();
    await expect(locator).toHaveValue(defaultValue);
    await locator.fill(value);
    await expect(locator).toHaveValue(value);
  }
}
