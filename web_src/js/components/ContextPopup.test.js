// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

import {flushPromises, mount} from '@vue/test-utils';
import ContextPopup from './ContextPopup.vue';

async function assertPopup(popupData, expectedIconColor, expectedIcon) {
  const date = new Date('2024-07-13T22:00:00Z');

  vi.spyOn(global, 'fetch').mockResolvedValue({
    json: vi.fn().mockResolvedValue({
      ok: true,
      created_at: date.toISOString(),
      repository: {full_name: 'user2/repo1'},
      ...popupData,
    }),
    ok: true,
  });

  const popup = mount(ContextPopup);
  popup.vm.$el.dispatchEvent(new CustomEvent('ce-load-context-popup', {
    detail: {owner: 'user2', repo: 'repo1', index: popupData.number},
  }));
  await flushPromises();

  expect(popup.get('p:nth-of-type(1)').text()).toEqual(`user2/repo1 on ${date.toLocaleDateString(undefined, {year: 'numeric', month: 'short', day: 'numeric'})}`);
  expect(popup.get('p:nth-of-type(2)').text()).toEqual(`${popupData.title} #${popupData.number}`);
  expect(popup.get('p:nth-of-type(3)').text()).toEqual(popupData.body);

  expect(popup.get('svg').classes()).toContain(expectedIcon);
  expect(popup.get('svg').classes()).toContain(expectedIconColor);

  for (const l of popupData.labels) {
    expect(popup.findAll('.ui.label').map((x) => x.text())).toContain(l.name);
  }
}

test('renders an open issue popup', async () => {
  await assertPopup({
    title: 'Open Issue',
    body: 'Open Issue Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'open',
    pull_request: null,
  }, 'green', 'octicon-issue-opened');
});

test('renders a closed issue popup', async () => {
  await assertPopup({
    title: 'Closed Issue',
    body: 'Closed Issue Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'closed',
    pull_request: null,
  }, 'red', 'octicon-issue-closed');
});

test('renders an open PR popup', async () => {
  await assertPopup({
    title: 'Open PR',
    body: 'Open PR Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'open',
    pull_request: {merged: false, draft: false},
  }, 'green', 'octicon-git-pull-request');
});

test('renders an open WIP PR popup', async () => {
  await assertPopup({
    title: 'WIP: Open PR',
    body: 'WIP Open PR Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'open',
    pull_request: {merged: false, draft: true},
  }, 'grey', 'octicon-git-pull-request-draft');
});

test('renders a closed PR popup', async () => {
  await assertPopup({
    title: 'Closed PR',
    body: 'Closed PR Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'closed',
    pull_request: {merged: false, draft: false},
  }, 'red', 'octicon-git-pull-request-closed');
});

test('renders a closed WIP PR popup', async () => {
  await assertPopup({
    title: 'WIP: Closed PR',
    body: 'WIP Closed PR Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'closed',
    pull_request: {merged: false, draft: true},
  }, 'red', 'octicon-git-pull-request-closed');
});

test('renders a merged PR popup', async () => {
  await assertPopup({
    title: 'Merged PR',
    body: 'Merged PR Body',
    number: 1,
    labels: [{color: 'd21b1fff', name: 'Bug'}, {color: 'aaff00', name: 'Confirmed'}],
    state: 'closed',
    pull_request: {merged: true, draft: false},
  }, 'purple', 'octicon-git-merge');
});

test('renders an issue popup with escaped HTML', async () => {
  const evil = '<a class="evil">evil link</a>';

  vi.spyOn(global, 'fetch').mockResolvedValue({
    json: vi.fn().mockResolvedValue({
      ok: true,
      created_at: '2024-07-13T22:00:00Z',
      repository: {full_name: evil},
      title: evil,
      body: evil,
      labels: [{color: '000666', name: evil}],
      state: 'open',
      pull_request: null,
    }),
    ok: true,
  });

  const popup = mount(ContextPopup);
  popup.vm.$el.dispatchEvent(new CustomEvent('ce-load-context-popup', {
    detail: {owner: evil, repo: evil, index: 1},
  }));
  await flushPromises();

  expect(() => popup.get('.evil')).toThrowError();
  expect(popup.get('p:nth-of-type(1)').text()).toContain(evil);
  expect(popup.get('p:nth-of-type(2)').text()).toContain(evil);
  expect(popup.get('p:nth-of-type(3)').text()).toContain(evil);
});

test('renders an issue popup with emojis', async () => {
  vi.spyOn(global, 'fetch').mockResolvedValue({
    json: vi.fn().mockResolvedValue({
      ok: true,
      created_at: '2024-07-13T22:00:00Z',
      repository: {full_name: 'user2/repo1'},
      title: 'Title',
      body: 'Body',
      labels: [{color: '000666', name: 'Tag :+1:'}],
      state: 'open',
      pull_request: null,
    }),
    ok: true,
  });

  const popup = mount(ContextPopup);
  popup.vm.$el.dispatchEvent(new CustomEvent('ce-load-context-popup', {
    detail: {owner: 'user2', repo: 'repo1', index: 1},
  }));
  await flushPromises();

  expect(popup.get('.ui.label').text()).toEqual('Tag 👍');
});
