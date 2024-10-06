// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
// @ts-check

// @watch start
// templates/user/auth/**
// templates/user/settings/**
// web_src/js/features/user-**
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user40');
});

test('WebAuthn register & login flow', async ({browser}, workerInfo) => {
  test.skip(workerInfo.project.name !== 'chromium', 'Uses Chrome protocol');
  const context = await load_logged_in_context(browser, workerInfo, 'user40');
  const page = await context.newPage();

  // Register a security key.
  let response = await page.goto('/user/settings/security');
  await expect(response?.status()).toBe(200);

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
      backupEligibility: true,
    },
  });

  await page.locator('input#nickname').fill('Testing Security Key');
  await page.getByText('Add security key').click();

  // Logout.
  await page.locator('div[aria-label="Profile and settings…"]').click();
  await page.getByText('Sign Out').click();
  await page.waitForURL(`${workerInfo.project.use.baseURL}/`);

  // Login.
  response = await page.goto('/user/login');
  await expect(response?.status()).toBe(200);

  await page.getByLabel('Username or email address').fill('user40');
  await page.getByLabel('Password').fill('password');
  await page.getByRole('button', {name: 'Sign in'}).click();
  await page.waitForURL(`${workerInfo.project.use.baseURL}/user/webauthn`);
  await page.waitForURL(`${workerInfo.project.use.baseURL}/`);

  // Cleanup.
  response = await page.goto('/user/settings/security');
  await expect(response?.status()).toBe(200);
  await page.getByRole('button', {name: 'Remove'}).click();
  await page.getByRole('button', {name: 'Yes'}).click();
  await page.waitForURL(`${workerInfo.project.use.baseURL}/user/settings/security`);
});
