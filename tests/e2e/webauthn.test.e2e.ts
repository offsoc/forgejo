// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// @watch start
// templates/user/auth/**
// templates/user/settings/**
// web_src/js/features/user-**
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './_test-setup.ts';
import {LoginPage} from './ui/LoginPage.ts';

const user = 'user40';

test.use({user});

test('WebAuthn register & login flow', async ({page, browser}, testInfo) => {
  test.skip(testInfo.project.name !== 'chromium', 'Uses Chrome protocol');

  // Register a security key.
  let response = await page.goto('/user/settings/security');
  expect(response?.status()).toBe(200);

  // https://github.com/microsoft/playwright/issues/7276#issuecomment-1516768428
  const cdpSession = await page.context().newCDPSession(page);
  await cdpSession.send('WebAuthn.enable');
  await cdpSession.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      ctap2Version: 'ctap2_1',
      hasUserVerification: true,
      transport: 'usb',
      automaticPresenceSimulation: true,
      isUserVerified: true,
    },
  });

  await page.locator('input#nickname').fill('Testing Security Key');
  await save_visual(page);
  await page.getByText('Add security key').click();

  await test.step('logout', async () => {
    await expect(async () => {
      await page.locator('div[aria-label="Profile and settings…"]').click();
      await page.getByText('Sign Out').click();
    }).toPass();
    await page.waitForURL(`${testInfo.project.use.baseURL}/`);
  });

  await test.step('login', async () => {
    response = await page.goto('/user/login');
    expect(response?.status()).toBe(200);
    await page.getByLabel('Username or email address').fill(user);
    await page.getByLabel('Password').fill('password');
    await page.getByRole('button', {name: 'Sign in'}).click();
    await page.waitForURL(`${testInfo.project.use.baseURL}/user/webauthn`);
    await page.waitForURL(`${testInfo.project.use.baseURL}/`);
  });

  await test.step('remove passkey', async () => {
    response = await page.goto('/user/settings/security');
    expect(response?.status()).toBe(200);
    await page.getByRole('button', {name: 'Remove'}).click();
    await save_visual(page);
    await page.getByRole('button', {name: 'Yes'}).click();
    await page.waitForLoadState();
  });

  // verify the user can login without a key
  await test.step('Use can login without a passkey', async () => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    const login = new LoginPage(page, testInfo);
    login.login(user, 'password');
  });
});
