import {mount, flushPromises} from '@vue/test-utils';
import ContextPopup from './ContextPopup.vue';

test('renders a issue info popup', async () => {
  const owner = 'user2';
  const repo = 'repo1';
  const index = 1;
  vi.spyOn(global, 'fetch').mockResolvedValue({
    json: vi.fn().mockResolvedValue({
      ok: true,
      created_at: '2023-09-30T19:00:00Z',
      repository: {full_name: owner},
      pull_request: null,
      state: 'open',
      title: 'Normal issue',
      body: 'Lorem ipsum...',
      number: index,
      labels: [{color: 'ee0701', name: "Bug :+1: <script class='evil'>alert('Oh no!');</script>"}],
    }),
    ok: true,
  });

  const wrapper = mount(ContextPopup);
  wrapper.vm.$el.dispatchEvent(new CustomEvent('ce-load-context-popup', {detail: {owner, repo, index}}));
  await flushPromises();

  // Header
  expect(wrapper.get('p:nth-of-type(1)').text()).toEqual('user2 on Sep 30, 2023');
  // Title
  expect(wrapper.get('p:nth-of-type(2)').text()).toEqual('Normal issue #1');
  // Body
  expect(wrapper.get('p:nth-of-type(3)').text()).toEqual('Lorem ipsum...');
  // Check that the state is correct.
  expect(wrapper.get('svg').classes()).toContain('octicon-issue-opened');
  // Ensure that script is not an element.
  expect(() => wrapper.get('.evil')).toThrowError();
  // Check content of label
  expect(wrapper.get('.ui.label').text()).toContain("Bug 👍 <script class='evil'>alert('Oh no!');</script>");
});
