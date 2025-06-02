// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// @watch start
// templates/repo/editor/**
// web_src/js/features/common-global.js
// routers/web/web.go
// services/repository/files/upload.go
// @watch end

import {expect} from '@playwright/test';
import {test, dynamic_id, save_visual} from './utils_e2e.ts';

test.use({user: 'user2'});
test.describe('Drag and Drop upload', () => {
  test('normal and special characters', async ({page}) => {
    const response = await page.goto(`/user2/file-uploads/_upload/main/`);
    expect(response?.status()).toBe(200); // Status OK

    const testID = dynamic_id();
    const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

    // create the virtual files
    const dataTransferA = await page.evaluateHandle(() => {
      const dt = new DataTransfer();
      // add items in different folders
      dt.items.add(new File(['Filecontent (dir1/file1.txt)'], 'dir1/file1.txt', {type: 'text/plain'}));
      dt.items.add(new File(["Another file's content(double / nested / file.txt)"], 'double / nested / file.txt', {type: 'text / plain'}));
      dt.items.add(new File(['Root file (root_file.txt)'], 'root_file.txt', {type: 'text/plain'}));
      dt.items.add(new File(['Umlaut test'], 'special/äüöÄÜÖß.txt', {type: 'text/plain'}));
      dt.items.add(new File(['Unicode test'], 'special/Ʉ₦ł₵ØĐɆ.txt', {type: 'text/plain'}));
      return dt;
    });
    // and drop them to the upload area
    await dropzone.dispatchEvent('drop', {dataTransfer: dataTransferA});

    await page.getByText('new branch').click();

    await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
    await page.getByRole('button', {name: 'Propose file change'}).click();

    // check that nested file structure is preserved
    await expect(page.getByRole('link', {name: 'dir1/file1.txt'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'double/nested/file.txt'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'special/äüöÄÜÖß.txt'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'special/Ʉ₦ł₵ØĐɆ.txt'})).toBeVisible();
    // Since this is a file in root, there two links with the same label
    // we take the on in #diff-file-tree
    await expect(page.locator('#diff-file-boxes').getByRole('link', {name: 'root_file.txt'})).toBeVisible();
  });

  test('strange paths and spaces', async ({page}) => {
    const response = await page.goto(`/user2/file-uploads/_upload/main/`);
    expect(response?.status()).toBe(200); // Status OK

    const testID = dynamic_id();
    const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

    // create the virtual files
    const dataTransferA = await page.evaluateHandle(() => {
      const dt = new DataTransfer();
      // add items in different folders
      dt.items.add(new File(['1'], '..dots.txt', {type: 'text/plain'}));
      dt.items.add(new File(['2'], '.dots.vanish.txt', {type: 'text/plain'}));
      dt.items.add(new File(['3'], 'special/S P  A   C   E    !.txt', {type: 'text/plain'}));
      return dt;
    });
    // and drop them to the upload area
    await dropzone.dispatchEvent('drop', {dataTransfer: dataTransferA});

    await page.getByText('new branch').click();

    await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
    await page.getByRole('button', {name: 'Propose file change'}).click();

    // check that nested file structure is preserved
    // Since this is a file in root, there two links with the same label
    // we take the on in #diff-file-tree
    await expect(page.locator('#diff-file-boxes').getByRole('link', {name: '.dots.vanish.txt'})).toBeVisible();
    await expect(page.getByRole('link', {name: 'special/S P  A   C   E    !.txt'})).toBeVisible();
    // Since this is a file in root, there two links with the same label
    // we take the on in #diff-file-tree
    await expect(page.locator('#diff-file-boxes').getByRole('link', {name: '..dots.txt'})).toBeVisible();
  });

  test('broken path slash in front', async ({page}) => {
    const response = await page.goto(`/user2/file-uploads/_upload/main/`);
    expect(response?.status()).toBe(200); // Status OK

    const testID = dynamic_id();
    const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

    // create the virtual files
    const dataTransferA = await page.evaluateHandle(() => {
      const dt = new DataTransfer();
      // add items in different folders
      dt.items.add(new File(['1'], '/special/badfirstslash.txt', {type: 'text/plain'}));
      return dt;
    });
    // and drop them to the upload area
    await dropzone.dispatchEvent('drop', {dataTransfer: dataTransferA});

    await page.getByText('new branch').click();

    await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
    await page.getByRole('button', {name: 'Propose file change'}).click();

    await expect(page.getByText('Failed to upload files to')).toBeVisible();

    await save_visual(page);
  });

  test('broken path with traversal', async ({page}) => {
    const response = await page.goto(`/user2/file-uploads/_upload/main/`);
    expect(response?.status()).toBe(200); // Status OK

    const testID = dynamic_id();
    const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

    // create the virtual files
    const dataTransferA = await page.evaluateHandle(() => {
      const dt = new DataTransfer();
      // add items in different folders
      dt.items.add(new File(['1'], '../baddots.txt', {type: 'text/plain'}));
      return dt;
    });
    // and drop them to the upload area
    await dropzone.dispatchEvent('drop', {dataTransfer: dataTransferA});

    await page.getByText('new branch').click();

    await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
    await page.getByRole('button', {name: 'Propose file change'}).click();

    await expect(page.getByText('Failed to upload files to')).toBeVisible();

    await save_visual(page);
  });
});
