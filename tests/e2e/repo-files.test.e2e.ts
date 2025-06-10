// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// @watch start
// templates/repo/editor/**
// web_src/js/features/common-global.js
// routers/web/web.go
// services/repository/files/**
// @watch end

import {expect} from '@playwright/test';
import {test, dynamic_id} from './utils_e2e.ts';

test.use({user: 'user2'});

interface TestCase {
  description: string;
  files: string[];
}

async function doUpload({page}, testCase: TestCase) {
  await page.goto(`/user2/file-uploads/_upload/main/`);
  const testID = dynamic_id();
  const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

  // create the virtual files
  const dataTransfer = await page.evaluateHandle((testCase: TestCase) => {
    const dt = new DataTransfer();
    for (const filename of testCase.files) {
      dt.items.add(new File([`File content of ${filename}`], filename, {type: 'text/plain'}));
    }
    return dt;
  }, testCase);
  // and drop them to the upload area
  await dropzone.dispatchEvent('drop', {dataTransfer});

  await page.getByText('new branch').click();

  await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
  // ToDo: Potential race condition: We do not currently wait for the upload to complete.
  // See https://codeberg.org/forgejo/forgejo/pulls/6687#issuecomment-5068272 and
  // https://codeberg.org/forgejo/forgejo/issues/5893#issuecomment-5068266 for details.
  // Workaround is to wait (the uploads are just a few bytes and usually complete instantly)
  //
  // eslint-disable-next-line playwright/no-wait-for-timeout
  await page.waitForTimeout(100);

  await page.getByRole('button', {name: 'Propose file change'}).click();
}

test.describe('Drag and Drop upload', () => {
  const goodTestCases: TestCase[] = [
    {
      description: 'normal and special characters',
      files: [
        'dir1/file1.txt',
        'double/nested/file.txt',
        'special/äüöÄÜÖß.txt',
        'special/Ʉ₦ł₵ØĐɆ.txt',
      ],
    },
    {
      description: 'strange paths and spaces',
      files: [
        '..dots.txt',
        '.dots.preserved.txt',
        'special/S P  A   C   E    !.txt',
      ],
    },
  ];

  // actual good tests based on definition above
  for (const testCase of goodTestCases) {
    test(`good: ${testCase.description}`, async ({page}) => {
      await doUpload({page}, testCase);

      // check that nested file structure is preserved
      for (const filename of testCase.files) {
        await expect(page.locator('#diff-file-boxes').getByRole('link', {name: filename})).toBeVisible();
      }
    });
  }

  const badTestCases: TestCase[] = [
    {
      description: 'broken path slash in front',
      files: [
        '/special/badfirstslash.txt',
      ],
    },
    {
      description: 'broken path with traversal',
      files: [
        '../baddots.txt',
      ],
    },
  ];

  // actual bad tests based on definition above
  for (const testCase of badTestCases) {
    test(`bad: ${testCase.description}`, async ({page}) => {
      await doUpload({page}, testCase);
      await expect(page.getByText('Failed to upload files to')).toBeVisible();
    });
  }
});
