// @watch start
// web_src/js/features/comp/ReactionSelector.js
// routers/web/repo/issue.go
// @watch end

import {expect, type Locator} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test.use({user: 'user2'});

const assertReactionCounts = (comment: Locator, counts: unknown) =>
  expect(async () => {
    await expect(comment.locator('.reactions')).toBeVisible();

    const reactions = Object.fromEntries(
      await Promise.all(
        (
          await comment
            .locator(`.reactions [role=button][data-reaction-content]`)
            .all()
        ).map(async (button) => [
          await button.getAttribute('data-reaction-content'),
          parseInt(await button.locator('.reaction-count').textContent()),
        ]),
      ),
    );
    // eslint-disable-next-line playwright/no-standalone-expect
    return expect(reactions).toStrictEqual(counts);
  }).toPass();

async function toggleReaction(menu: Locator, reaction: string) {
  await menu.evaluateAll((menus) => menus[0].focus());
  await menu.locator('.add-reaction').click();
  await menu.locator(`[role=menuitem][data-reaction-content="${reaction}"]`).click();
}

test('Reaction Selectors', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  const comment = page.locator('.comment#issuecomment-2').first();

  const topPicker = comment.locator('.actions [role=menu].select-reaction');
  const bottomPicker = comment.locator('.reactions').getByRole('menu');

  await assertReactionCounts(comment, {'laugh': 2});

  await toggleReaction(topPicker, '+1');
  await assertReactionCounts(comment, {'laugh': 2, '+1': 1});

  await toggleReaction(bottomPicker, '+1');
  await assertReactionCounts(comment, {'laugh': 2});

  await toggleReaction(bottomPicker, '-1');
  await assertReactionCounts(comment, {'laugh': 2, '-1': 1});

  await toggleReaction(topPicker, '-1');
  await assertReactionCounts(comment, {'laugh': 2});

  await comment.locator('.reactions [role=button][data-reaction-content=laugh]').click();
  await assertReactionCounts(comment, {'laugh': 1});

  await toggleReaction(topPicker, 'laugh');
  await assertReactionCounts(comment, {'laugh': 2});
  await save_visual(page);
});
