import {DoUpdateRelativeTime, HALF_MINUTE, ONE_MINUTE, ONE_HOUR, ONE_DAY, ONE_WEEK, ONE_MONTH, ONE_YEAR} from './relative-time.js';

test('CalculateRelativeTimes', () => {
  window.config.pageData.PLURAL_RULE_LANG = 0;
  window.config.pageData.DATETIMESTRINGS = {
    'FUTURE': 'future',
    'NOW': 'now',
    'tool.ago_1day': 'yesterday',
    'tool.ago_1week': 'last week',
    'tool.ago_1month': 'last month',
    'tool.ago_1year': 'last year',
    'tool.ago_2days': 'two days ago',
    'tool.ago_2weeks': 'two weeks ago',
    'tool.ago_2months': 'two months ago',
    'tool.ago_2years': 'two years ago',
  };
  window.config.pageData.PLURALSTRINGS_LANG = {
    'tool.ago_mins': ['%d minute ago', '%d minutes ago'],
    'tool.ago_hours': ['%d hour ago', '%d hours ago'],
    'tool.ago_days': ['%d day ago', '%d days ago'],
    'tool.ago_weeks': ['%d week ago', '%d weeks ago'],
    'tool.ago_months': ['%d month ago', '%d months ago'],
    'tool.ago_years': ['%d year ago', '%d years ago'],
  };
  const mock = document.createElement('relative-time');

  const now = Date.parse('2024-10-27T04:05:30+01:00');  // One hour after DST switchover, CET.

  expect(DoUpdateRelativeTime(mock, now)).toEqual(null);
  expect(mock.textContent).toEqual('');

  mock.setAttribute('datetime', '2024-10-27T04:05:40+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('future');

  mock.setAttribute('datetime', '2024-10-27T04:05:10+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(HALF_MINUTE);
  expect(mock.textContent).toEqual('now');

  mock.setAttribute('datetime', '2024-10-27T04:04:30+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('1 minute ago');

  mock.setAttribute('datetime', '2024-10-27T04:04:00+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('1 minute ago');

  mock.setAttribute('datetime', '2024-10-27T04:03:20+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('2 minutes ago');

  mock.setAttribute('datetime', '2024-10-27T04:00:00+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('5 minutes ago');

  mock.setAttribute('datetime', '2024-10-27T03:59:30+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('6 minutes ago');

  mock.setAttribute('datetime', '2024-10-27T03:01:00+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('1 hour ago');

  mock.setAttribute('datetime', '2024-10-27T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('4 hours ago');  // This tests DST switchover

  mock.setAttribute('datetime', '2024-10-27T00:01:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('5 hours ago');

  mock.setAttribute('datetime', '2024-10-26T22:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('7 hours ago');

  mock.setAttribute('datetime', '2024-10-26T05:08:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('23 hours ago');

  mock.setAttribute('datetime', '2024-10-26T04:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_DAY);
  expect(mock.textContent).toEqual('yesterday');

  mock.setAttribute('datetime', '2024-10-25T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_DAY);
  expect(mock.textContent).toEqual('two days ago');

  mock.setAttribute('datetime', '2024-10-21T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_DAY);
  expect(mock.textContent).toEqual('6 days ago');

  mock.setAttribute('datetime', '2024-10-20T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_WEEK);
  expect(mock.textContent).toEqual('last week');

  mock.setAttribute('datetime', '2024-10-14T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_WEEK);
  expect(mock.textContent).toEqual('last week');

  mock.setAttribute('datetime', '2024-10-13T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_WEEK);
  expect(mock.textContent).toEqual('two weeks ago');

  mock.setAttribute('datetime', '2024-10-06T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_WEEK);
  expect(mock.textContent).toEqual('3 weeks ago');

  mock.setAttribute('datetime', '2024-09-25T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('last month');

  mock.setAttribute('datetime', '2024-08-30T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('last month');

  mock.setAttribute('datetime', '2024-07-30T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('two months ago');

  mock.setAttribute('datetime', '2024-05-30T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('5 months ago');

  mock.setAttribute('datetime', '2024-03-01T01:00:00+01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('8 months ago');

  mock.setAttribute('datetime', '2024-02-29T01:00:00+01:00');  // Leap day handling
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('8 months ago');

  mock.setAttribute('datetime', '2024-02-27T01:00:00-03:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('8 months ago');

  mock.setAttribute('datetime', '2023-11-15T01:00:00+03:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MONTH);
  expect(mock.textContent).toEqual('11 months ago');

  mock.setAttribute('datetime', '2023-10-20T01:00:00+08:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_YEAR);
  expect(mock.textContent).toEqual('last year');

  mock.setAttribute('datetime', '2022-10-30T01:00:00-05:30');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_YEAR);
  expect(mock.textContent).toEqual('last year');

  mock.setAttribute('datetime', '2022-10-20T01:00:00+10:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_YEAR);
  expect(mock.textContent).toEqual('two years ago');

  mock.setAttribute('datetime', '2021-10-20T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_YEAR);
  expect(mock.textContent).toEqual('3 years ago');

  mock.setAttribute('datetime', '2014-10-20T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_YEAR);
  expect(mock.textContent).toEqual('10 years ago');

  // Timezone tests
  mock.setAttribute('datetime', '2024-10-27T01:01:00-05:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('future');

  mock.setAttribute('datetime', '2024-10-27T05:05:25+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(HALF_MINUTE);
  expect(mock.textContent).toEqual('now');

  mock.setAttribute('datetime', '2024-10-27T05:04:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('1 minute ago');

  mock.setAttribute('datetime', '2024-10-27T05:02:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('3 minutes ago');

  mock.setAttribute('datetime', '2024-10-27T04:06:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_MINUTE);
  expect(mock.textContent).toEqual('59 minutes ago');

  mock.setAttribute('datetime', '2024-10-27T04:05:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('1 hour ago');

  mock.setAttribute('datetime', '2024-10-27T01:00:00+02:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('4 hours ago');

  mock.setAttribute('datetime', '2024-10-27T01:00:00+04:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('6 hours ago');

  mock.setAttribute('datetime', '2024-10-27T01:00:00+10:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('12 hours ago');

  mock.setAttribute('datetime', '2024-10-27T01:00:00Z');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('2 hours ago');

  mock.setAttribute('datetime', '2024-10-26T15:00:00-01:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('11 hours ago');

  mock.setAttribute('datetime', '2024-10-25T19:00:00-11:00');
  expect(DoUpdateRelativeTime(mock, now)).toEqual(ONE_HOUR);
  expect(mock.textContent).toEqual('21 hours ago');
});
