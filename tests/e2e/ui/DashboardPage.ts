import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class DashboardPage extends ForgejoUi {
  readonly repositoriesTabButton: Locator;
  readonly organizationsTabButton: Locator;
  readonly repositoryListEntries: Locator;
  readonly search: Locator;

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);
    this.repositoriesTabButton = page.getByText('Repositories');
    this.organizationsTabButton = page.getByText('Organizations');
    this.search = page.getByPlaceholder('Search repos...');
    this.repositoryListEntries = page.getByRole('listitem');
  }

  async goto() {
    const response = await this.page.goto('/');
    expect(response.status()).toBe(200);
  }

  async searchFor(text: string) {
    if (await this.search.inputValue() === text) {
      return;
    }

    await Promise.all([
      this.page.waitForResponse((response) => response.url().match(new RegExp(`/repo/search?.*&q=${text}(&|$)`)) && response.ok()),
      this.search.fill(text),
    ]);

    await expect(this.search).toHaveValue(text);
  }

  async repositoriesCount(expectedCount: number) {
    await expect(this.repositoryListEntries).toHaveCount(expectedCount);
  }
}
