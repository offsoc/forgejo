import {GET} from '../modules/fetch.js';

// English default values, will be overwritten with translated texts from the server.
/* eslint-disable-next-line prefer-const */
let DATETIMESTRINGS = {
  'future': 'in the future',
  'now': 'now',
  '1min': '1 minute ago',
  'mins': (minutes) => `${minutes} minutes ago`,
  '1hour': '1 hour ago',
  'hour': (hours) => `${hours} hours ago`,
  '1day': 'yesterday',
  'days': (days) => `${days} days ago`,
  '1week': 'last week',
  'weeks': (weeks) => `${weeks} weeks ago`,
  '1month': 'last month',
  'months': (months) => `${months} months ago`,
  '1year': 'last year',
  'years': (years) => `${years} years ago`,
};

const ABSOLUTE_DATETIME_FORMAT = new Intl.DateTimeFormat(navigator.language, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZoneName: 'short',
});

/** Update the displayed text of the given relative-time DOM element with its human-readable, localized relative time string. */
function UpdateRelativeTime(object) {
  if (!(object && object.attributes.datetime && object.attributes.datetime.nodeValue)) {
    return;  // Object does not contain a datetime.
  }

  const then = Date.parse(object.attributes.datetime.nodeValue);
  const now = Date.now();
  const milliseconds = now - then;

  if (Number.isNaN(milliseconds)) {
    return;  // Datetime is invalid.
  }

  object.setAttribute('data-tooltip-content', ABSOLUTE_DATETIME_FORMAT.format(then));

  if (milliseconds < 0) {
    // Datetime is in the future.
    object.textContent = DATETIMESTRINGS['future'];
    return;
  }

  const minutes = Math.floor(milliseconds / 60000);
  if (minutes < 1) {
    // Datetime is very recent.
    object.textContent = DATETIMESTRINGS['now'];
    return;
  }
  if (minutes === 1) {
    // Datetime is one minute ago.
    object.textContent = DATETIMESTRINGS['1min'];
    return;
  }
  if (minutes < 60) {
    // Datetime is several minutes but less than an hour ago.
    object.textContent = DATETIMESTRINGS['mins'](minutes);
    return;
  }

  const hours = Math.floor(minutes / 60);
  if (hours === 1) {
    // Datetime is one hour ago.
    object.textContent = DATETIMESTRINGS['1hour'];
    return;
  }
  if (hours < 24) {
    // Datetime is several hours but less than a day ago.
    object.textContent = DATETIMESTRINGS['hours'](hours);
    return;
  }

  const days = Math.floor(hours / 24);
  if (days === 1) {
    // Datetime is one day ago.
    object.textContent = DATETIMESTRINGS['1day'];
    return;
  }
  if (days < 7) {
    // Datetime is several days but less than a week ago.
    object.textContent = DATETIMESTRINGS['days'](days);
    return;
  }
  if (days < 30) {
    // Datetime is at least one week but less than a month ago.
    const weeks = Math.floor(days / 7);
    if (weeks === 1) {
      // Datetime is one week ago.
      object.textContent = DATETIMESTRINGS['1week'];
      return;
    }
    // Datetime is several weeks ago (but less than a month).
    object.textContent = DATETIMESTRINGS['weeks'](weeks);
    return;
  }

  if (days < 365) {
    // Datetime is at least one month but less than a year ago.
    const months = Math.floor(days / 30);
    if (months === 1) {
      // Datetime is one month ago.
      object.textContent = DATETIMESTRINGS['1month'];
      return;
    }
    // Datetime is several months ago (but less than a year).
    object.textContent = DATETIMESTRINGS['months'](months);
    return;
  }

  const years = Math.floor(days / 365);
  if (years === 1) {
    // Datetime is one year ago.
    object.textContent = DATETIMESTRINGS['1year'];
    return;
  }
  // Datetime is more than a year ago.
  object.textContent = DATETIMESTRINGS['years'](years);
}

/** Update the displayed text of all relative-time DOM elements with their respective human-readable, localized relative time string. */
function UpdateAllRelativeTimes() {
  for (const object of document.querySelectorAll('relative-time')) UpdateRelativeTime(object);
}

// Immediately update all relative-time elements and refresh them every 60 seconds.
async function UpdateAllRelativeTimesFirstTime() {
  try {
    const response = await GET('/relative-time-constants');
    if (response.ok) {
      const run = await response.text();
      /* eslint-disable-next-line no-eval */
      eval(run);
    } else {
      console.error('Failed to query relative datetime string, HTTP status code', response.status);
    }
  } catch (error) {
    console.error('Failed to query relative datetime string; error:', error);
  }

  UpdateAllRelativeTimes();
  setInterval(UpdateAllRelativeTimes, 60 * 1000);
}

UpdateAllRelativeTimesFirstTime();
