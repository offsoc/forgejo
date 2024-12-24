// @watch start
// routers/web/user/**
// templates/shared/user/**
// web_src/js/features/common-global.js
// @watch end

import {ProfilePage} from '../ui/ProfilePage.ts';
import {test} from '../_test-setup.ts';

/* eslint playwright/expect-expect: ["error", { "assertFunctionNames": ["verifyCanFollow", "verifyCanUnfollow"] }] */
test.describe('Profile Actions', () => {
  test.use({user: 'user2'});

  // todo separate user for follow and block
  const testUser = 'user1';

  test('Follow actions', async ({page}, testInfo) => {
    const profile = new ProfilePage(page, testInfo, testUser);

    await profile.goto();

    await test.step(`Ensure test user can follow ${testUser}`, async () => {
      await profile.verifyCanFollow();
      await profile.followUser();
      await profile.verifyCanUnfollow();
    });

    await test.step(`Ensure test user can unfollow ${testUser}`, async () => {
      await profile.verifyCanUnfollow();
      await profile.unfollowUser();
      await profile.verifyCanFollow();
    });
  });

  test('Block actions', async ({page}, testInfo) => {
    const profile = new ProfilePage(page, testInfo, testUser);

    await profile.goto();

    await test.step(`Ensure test user can block ${testUser}`, async () => {
      await profile.verifyUserBlocked();
      await profile.blockUser();
      await profile.blockConfirm();
      await profile.verifyUserUnblocked();
    });

    await test.step(`Ensure test user can't follow blocked user ${testUser}`, async () => {
      await profile.verifyCanFollow();
      await profile.followUser();
      await profile.flashMessageToContain('You cannot follow this user because you have blocked this user or this user has blocked you.');
    });

    await test.step(`Ensure test user can unblock ${testUser}`, async () => {
      await profile.verifyUserUnblocked();
      await profile.unblockUser();
    });
  });
});
