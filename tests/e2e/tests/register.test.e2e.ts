// @watch start
// templates/user/auth/**
// web_src/js/features/user-**
// modules/{user,auth}/**
// @watch end

import {expect} from '@playwright/test';
import {test} from '../_test-setup.ts';
import {RegisterPage} from '../ui/RegisterPage.ts';

test.describe('Register Form', () => {
  test.use({user: null});

  test('Register Form', async ({page}, testInfo) => {
    const register = new RegisterPage(page, testInfo);
    await register.goto();

    await register.fillUsername(`e2e-test-${testInfo.workerIndex}-${process.pid}`);
    await register.fillEmail(`e2e-test-${testInfo.workerIndex}-${process.pid}@test.com`);
    await register.fillPassword('test123test123');
    await register.fillConfirmPassword('test123test123');

    await register.submitForm();

    // Make sure we routed to the home page. Else login failed.
    expect(page.url()).toBe(`${testInfo.project.use.baseURL}/`);
    await expect(page.locator('.secondary-nav span>img.ui.avatar')).toBeVisible();
    await expect(page.locator('.ui.positive.message.flash-success')).toHaveText('Account was successfully created. Welcome!');
  });
});
