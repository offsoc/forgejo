import {mount, flushPromises} from '@vue/test-utils';
import RepoActionView from './RepoActionView.vue';

test('processes ##[group] and ##[endgroup]', async () => {
  Object.defineProperty(document.documentElement, 'lang', {value: 'en'});
  vi.spyOn(global, 'fetch').mockImplementation((url, opts) => {
    const artifacts_value = {
      artifacts: [],
    };
    const stepsLog_value = [
      {
        step: 0,
        cursor: 0,
        lines: [
          {index: 1, message: '##[group]Test group', timestamp: 0},
          {index: 2, message: 'A test line', timestamp: 0},
          {index: 3, message: '##[endgroup]', timestamp: 0},
          {index: 4, message: 'A line outside the group', timestamp: 0},
        ],
      },
    ];
    const jobs_value = {
      state: {
        run: {
          status: 'success',
          commit: {
            pusher: {},
          },
        },
        currentJob: {
          steps: [
            {
              summary: 'Test Job',
              duration: '1s',
              status: 'success',
            },
          ],
        },
      },
      logs: {
        stepsLog: opts.body?.includes('"cursor":null') ? stepsLog_value : [],
      },
    };

    return Promise.resolve({
      ok: true,
      json: vi.fn().mockResolvedValue(
        url.endsWith('/artifacts') ? artifacts_value : jobs_value,
      ),
    });
  });

  const wrapper = mount(RepoActionView, {
    props: {
      jobIndex: '1',
      locale: {
        approve: '',
        cancel: '',
        rerun: '',
        artifactsTitle: '',
        areYouSure: '',
        confirmDeleteArtifact: '',
        rerun_all: '',
        showTimeStamps: '',
        showLogSeconds: '',
        showFullScreen: '',
        downloadLogs: '',
        status: {
          unknown: '',
          waiting: '',
          running: '',
          success: '',
          failure: '',
          cancelled: '',
          skipped: '',
          blocked: '',
        },
      },
    },
  });
  await flushPromises();
  await wrapper.get('.job-step-summary').trigger('click');
  await flushPromises();

  // Test if header was loaded correctly
  expect(wrapper.get('.step-summary-msg').text()).toEqual('Test Job');

  // Check if 3 lines where rendered
  expect(wrapper.findAll('.job-log-line').length).toEqual(3);

  // Check if line 1 contains the group header
  expect(wrapper.get('.job-log-line:nth-of-type(1) > details.log-msg').text()).toEqual('Test group');

  // Check if right after the header line exists a log list
  expect(wrapper.find('.job-log-line:nth-of-type(1) + .job-log-list.hidden').exists()).toBe(true);

  // Check if inside the loglist exist exactly one log line
  expect(wrapper.findAll('.job-log-list > .job-log-line').length).toEqual(1);

  // Check if inside the loglist is an logline with our second logline
  expect(wrapper.get('.job-log-list > .job-log-line > .log-msg').text()).toEqual('A test line');

  // Check if after the log list exists another log line
  expect(wrapper.get('.job-log-list + .job-log-line > .log-msg').text()).toEqual('A line outside the group');
});
