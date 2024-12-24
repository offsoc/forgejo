import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class ProfilePage extends ForgejoUi {
  readonly followButton: Locator;
  readonly blockButton: Locator;
  readonly blockModal: Locator;
  readonly blockModalActionOk: Locator;
  readonly location: string;

  constructor(page: Page, testInfo: TestInfo, user: string) {
    super(page, testInfo);
    this.location = `/${user}`;
    this.followButton = page.locator('.follow button');
    this.blockButton = page.locator('.block button');
    this.blockModal = page.locator('#block-user');
    this.blockModalActionOk = this.blockModal.locator('.ok');
  }

  async goto() {
    await this.page.goto(this.location);
  }

  async followUser() {
    await this.clickAndWaitForNetworkResponse(this.followButton, `${this.location}?action`);
  }

  async verifyUserBlocked() {
    await expect(this.blockButton).toContainText('Block');
  }

  async verifyUserUnblocked() {
    await expect(this.blockButton).toContainText('Unblock');
  }

  async verifyCanFollow() {
    await expect(this.followButton).toContainText('Follow');
  }

  async verifyCanUnfollow() {
    await expect(this.followButton).toContainText('Unfollow');
  }

  async unfollowUser() {
    await expect(this.followButton).toContainText('Unfollow');
    await this.clickAndWaitForNetworkResponse(this.followButton, `${this.location}?action`);
  }

  async blockUser() {
    await this.blockButton.click();
  }

  async blockConfirm() {
    await expect(this.blockModal).toBeVisible();
    await this.clickAndWaitForNetworkResponse(this.blockModalActionOk, `${this.location}?action`);
  }

  async unblockUser() {
    await expect(this.blockButton).toContainText('Unblock');
    await this.clickAndWaitForNetworkResponse(this.blockButton, `${this.location}?action`);
  }
}
