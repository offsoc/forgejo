import {ForgejoUi} from './ForgejoUi.ts';
import {type Locator, type Page, type TestInfo} from '@playwright/test';

export class RepositoryOverviewPage extends ForgejoUi {
  readonly repoHeader: Locator;
  readonly location: string;

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);
    this.repoHeader = page.locator('.repo-header');
  }
}
