import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class ExplorePage extends ForgejoUi {
  readonly secondaryMenu: Locator;
  readonly tabReposButton: Locator;
  readonly tabUsersButton: Locator;
  readonly tabOrganizationsButton: Locator;
  readonly tabCodeButton: Locator;

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);
    this.secondaryMenu = page.getByRole('main');
    this.tabReposButton = this.secondaryMenu.locator('a.item[href="/explore/repos"]');
    this.tabUsersButton = this.secondaryMenu.locator('a.item[href="/explore/users"]');
    this.tabOrganizationsButton = this.secondaryMenu.locator('a.item[href="/explore/organizations"]');
    this.tabCodeButton = this.secondaryMenu.locator('a.item[href="/explore/code"]');
  }

  async goto() {
    const response = await this.page.goto('/explore');
    expect(response.status()).toBe(200);
  }
}
