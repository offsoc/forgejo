import {expect} from '@playwright/test';
import {AxeBuilder} from '@axe-core/playwright';

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

  // check that multiple help text are correctly aligned to each other
  // used for example to separate read/write permissions in team permission matrix
  for (const l of await page.locator('label:has(.help + .help)').all()) {
    const helpLabels = await l.locator('.help').all();
    const boxes = await Promise.all(helpLabels.map((help) => help.boundingBox()));
    for (let i = 1; i < boxes.length; i++) {
      // help texts vertically aligned on top of each other
      await expect(boxes[i].x).toBe(boxes[0].x);
      // help texts don't horizontally intersect each other
      await expect(boxes[i].y + boxes[i].height).toBeGreaterThanOrEqual(boxes[i - 1].y + boxes[i - 1].height);
    }
  }
}
