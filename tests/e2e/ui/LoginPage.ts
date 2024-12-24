import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class LoginPage extends ForgejoUi {
  readonly userNameInput: Locator;
  readonly passwordInput: Locator;
  readonly loginButton: Locator;

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);

    // Bind all selectors
    this.userNameInput = page.locator('input[name="user_name"]');
    this.passwordInput = page.locator('input[name="password"]');
    this.loginButton = page.locator('form button.ui.primary.button:visible');
  }

  async goto() {
    const response = await this.page.goto('/user/login');
    expect(response?.status()).toBe(200);
  }

  async login(user: string, password: string) {
    await this.userNameInput.fill(user);
    await this.passwordInput.fill(password);
    await this.loginButton.click();
    await this.page.waitForLoadState();
  }
}
