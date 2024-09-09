import {expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

export async function validate_form({page}, scope) {
  scope ??= 'form';
  const accessibilityScanResults = await new AxeBuilder({page})
    // disable checking for link style - should be fixed, but not now
    .disableRules('link-in-text-block')
    .include(scope)
    // exclude automated tooltips from accessibility scan, remove when fixed
    .exclude('span[data-tooltip-content')
    // exclude weird non-semantic HTML disabled content
    .exclude('.disabled')
    .analyze();
  expect(accessibilityScanResults.violations).toEqual([]);

  // assert CSS properties that needed to be overriden for forms (ensure they remain active)
  const boxes = page.getByRole('checkbox').or(page.getByRole('radio'));
  for (const b of await boxes.all()) {
    await expect(b).toHaveCSS('margin-left', '0px');
    await expect(b).toHaveCSS('margin-top', '0px');
    await expect(b).toHaveCSS('vertical-align', 'baseline');
  }
  // assert no (trailing) colon is used in labels
  // might be necessary to adjust in case colons are strictly necessary in help text
  for (const l of await page.locator('label').all()) {
    const str = await l.textContent();
    await expect(str.split('\n')[0]).not.toContain(':');
  }
}
