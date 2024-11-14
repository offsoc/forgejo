import {expect, type Page} from '@playwright/test';
import {AxeBuilder} from '@axe-core/playwright';

export async function accessibilityCheck({page}: {page: Page}, includes: string[], excludes: string[], disabledRules: string[]) {
  // contrast of inline links is still a global issue in Forgejo
  disabledRules += 'link-in-text-block';

  let accessibilityScanner = await new AxeBuilder({page})
    .disableRules(disabledRules);
  // passing the whole array seems to be not supported,
  // iterating has the nice side-effectof skipping this if the array is empty
  for (const incl of includes) {
    // passing the whole array seems to be not supported
    accessibilityScanner = accessibilityScanner.include(incl);
  }
  for (const excl of excludes) {
    accessibilityScanner = accessibilityScanner.exclude(excl);
  }

  // scan the page both in dark and light theme
  let accessibilityScanResults = await accessibilityScanner.analyze();
  expect(accessibilityScanResults.violations).toEqual([]);
  await page.emulateMedia({colorScheme: 'dark'});
  // in https://codeberg.org/forgejo/forgejo/pulls/5899 there have been
  // some weird failures related to contrast scanning,
  // reporting for colours that haven't been used and no trace in the
  // screenshots.
  // Since this was only happening with some browsers and not always,
  // my bet is on a transition effect on dark/light mode switch.
  // Waiting a little seems to work around this.
  await page.waitForTimeout(100); // eslint-disable-line playwright/no-wait-for-timeout
  accessibilityScanResults = await accessibilityScanner.analyze();
  expect(accessibilityScanResults.violations).toEqual([]);
  await page.emulateMedia({colorScheme: 'light'});
}
