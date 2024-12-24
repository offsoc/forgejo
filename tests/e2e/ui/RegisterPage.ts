import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class RegisterPage extends ForgejoUi {
  readonly inputEmail: Locator;
  readonly inputUsername: Locator;
  readonly inputPassword: Locator;
  readonly inputConfirmPassword: Locator;
  readonly formSubmitButton: Locator;

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);
    this.inputUsername = page.getByLabel('Username');
    this.inputEmail = page.getByLabel('Email address');
    this.inputPassword = page.getByLabel('Password', {exact: true});
    this.inputConfirmPassword = page.getByLabel('Confirm password');
    this.formSubmitButton = page.getByRole('button', {name: 'Register account'});
  }

  async goto() {
    const response = await this.page.goto('/user/sign_up');
    expect(response.status()).toBe(200);
    expect(this.page.url()).toBe(`${this.testInfo.project.use.baseURL}/user/sign_up`);
  }

  async fillUsername(username: string) {
    await this.fillAndValidate(this.inputUsername, username);
  }

  async fillEmail(email: string) {
    await this.fillAndValidate(this.inputEmail, email);
  }

  async fillPassword(password: string) {
    await this.fillAndValidate(this.inputPassword, password);
  }

  async fillConfirmPassword(confirmPassword: string) {
    await this.fillAndValidate(this.inputConfirmPassword, confirmPassword);
  }

  async submitForm() {
    const {baseURL} = this.testInfo.project.use;
    const match = new RegExp(`^${baseURL}(/|)$`, 'i');

    await Promise.all([
      // post form
      this.page.waitForResponse((response) => response?.url().endsWith('/user/sign_up') && response?.status() === 303),
      // redirect to landing page
      this.page.waitForResponse((response) => match.test(response?.url()) && response?.status() === 200),
      this.formSubmitButton.click(),
    ]);
  }
}
