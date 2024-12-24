import {ForgejoUi} from './ForgejoUi.ts';
import {expect, type Locator, type Page, type TestInfo} from '@playwright/test';

export class ActionsPage extends ForgejoUi {
  readonly actionHintContainer: Locator;
  readonly allWorkflowsContainer: Locator;
  readonly workflowTriggerButton: Locator;
  readonly workflowTriggerMenu: Locator;
  readonly workflowMenuDispatchSubmit: Locator;

  readonly search: Locator;

  readonly workflowTriggerNotificationText = 'This workflow has a workflow_dispatch event trigger.';

  constructor(page: Page, testInfo: TestInfo) {
    super(page, testInfo);
    this.actionHintContainer = page.locator('.container .empty-placeholder h2');
    this.workflowTriggerButton = page.locator('#workflow_dispatch_dropdown > button');
    this.workflowTriggerMenu = page.locator('#workflow_dispatch_dropdown > .menu');
    this.allWorkflowsContainer = page.getByTestId('actions-runs-all-workflows');
    this.workflowMenuDispatchSubmit = this.workflowTriggerMenu.locator('#workflow-dispatch-submit');
    this.search = page.getByPlaceholder('Search repos...');
  }

  async goto(owner: string, repo: string, query: string = null) {
    await this.page.goto(`/${owner}/${repo}/actions${query ? `?${query}` : ''}`);
  }

  async verifyNoWorkflowsYetMessage() {
    await expect(this.actionHintContainer).toBeVisible();
    await expect(this.actionHintContainer).toContainText('There are no workflows yet.');
  }

  async verifyNoWorkflowRunsMessage() {
    await expect(this.actionHintContainer).toBeVisible();
    await expect(this.actionHintContainer).toContainText('The workflow has no runs yet.');
  }

  async clickWorkflowByName(workflowName: string) {
    await this.clickAndWaitForNetworkResponse(
      this.allWorkflowsContainer.getByText(workflowName),
      `workflow=${workflowName}`,
    );
  }

  async hasWorkflowTriggerNotificationText() {
    await expect(this.page.getByText(this.workflowTriggerNotificationText)).toBeVisible();
  }

  async hasNoWorkflowTriggerNotificationText() {
    await expect(this.page.getByText(this.workflowTriggerNotificationText)).toBeHidden();
  }
}
