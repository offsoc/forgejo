import {GetPluralizedString} from './i18n.js';
const {pageData} = window.config;

const ABSOLUTE_DATETIME_FORMAT = new Intl.DateTimeFormat(navigator.language, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZoneName: 'short',
});
const FALLBACK_DATETIME_FORMAT = new Intl.RelativeTimeFormat(navigator.language, {style: 'long'});

function GetPluralizedStringOrFallback(key, n, unit) {
  const translation = GetPluralizedString(key, n, true);
  if (translation) return translation.replace('%d', n);
  return FALLBACK_DATETIME_FORMAT.format(-n, unit);
}

/** Update the displayed text of the given relative-time DOM element with its human-readable, localized relative time string. */
function DoUpdateRelativeTime(object) {
  if (!(object?.attributes?.datetime?.nodeValue)) {
    return null;  // Object does not contain a datetime.
  }

  const then = Date.parse(object.attributes.datetime.nodeValue);
  const now = Date.now();
  const milliseconds = now - then;

  if (Number.isNaN(milliseconds)) {
    return null;  // Datetime is invalid.
  }

  object.setAttribute('data-tooltip-content', ABSOLUTE_DATETIME_FORMAT.format(then));

  if (milliseconds < 0) {
    // Datetime is in the future.
    object.textContent = pageData.DATETIMESTRINGS.FUTURE;
    return 60 * 1000;
  }

  const minutes = Math.floor(milliseconds / 60000);
  if (minutes < 1) {
    // Datetime is very recent.
    object.textContent = pageData.DATETIMESTRINGS.NOW;
    return 30 * 1000;
  }
  if (minutes < 60) {
    // Datetime is one or more minutes but less than an hour ago.
    object.textContent = GetPluralizedStringOrFallback('tool.ago_mins', minutes, 'minute');
    return 60 * 1000;
  }

  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    // Datetime is one or more hours but less than a day ago.
    object.textContent = GetPluralizedStringOrFallback('tool.ago_hours', hours, 'hour');
    return 60 * 60 * 1000;
  }

  const days = Math.floor(hours / 24);
  if (days < 365) {
    if (days < 7) {
      if (days === 1 && pageData.DATETIMESTRINGS['tool.ago_1day']) {
        // Datetime is one day ago.
        object.textContent = pageData.DATETIMESTRINGS['tool.ago_1day'];
      } else if (days === 2 && pageData.DATETIMESTRINGS['tool.ago_2days']) {
        // Datetime is two days ago.
        object.textContent = pageData.DATETIMESTRINGS['tool.ago_2days'];
      } else {
        // Datetime is several days but less than a week ago.
        object.textContent = GetPluralizedStringOrFallback('tool.ago_days', days, 'day');
      }
      return 24 * 60 * 60 * 1000;
    }

    if (days < 30) {
      // Datetime is at least one week but less than a month ago.
      const weeks = Math.floor(days / 7);
      if (weeks === 1 && pageData.DATETIMESTRINGS['tool.ago_1week']) {
        // Datetime is one week ago.
        object.textContent = pageData.DATETIMESTRINGS['tool.ago_1week'];
      } else if (weeks === 2 && pageData.DATETIMESTRINGS['tool.ago_2weeks']) {
        // Datetime is two week ago.
        object.textContent = pageData.DATETIMESTRINGS['tool.ago_2weeks'];
      } else {
        // Datetime is several weeks ago (but less than a month).
        object.textContent = GetPluralizedStringOrFallback('tool.ago_weeks', weeks, 'week');
      }
      return 7 * 24 * 60 * 60 * 1000;
    }

    // Datetime is at least one month but less than a year ago.
    const months = Math.floor(days / 30);
    if (months === 1 && pageData.DATETIMESTRINGS['tool.ago_1month']) {
      // Datetime is one month ago.
      object.textContent = pageData.DATETIMESTRINGS['tool.ago_1month'];
    } else if (months === 2 && pageData.DATETIMESTRINGS['tool.ago_2months']) {
      // Datetime is two months ago.
      object.textContent = pageData.DATETIMESTRINGS['tool.ago_2months'];
    } else {
      // Datetime is several months ago (but less than a year).
      object.textContent = GetPluralizedStringOrFallback('tool.ago_months', months, 'month');
    }
    return 30 * 24 * 60 * 60 * 1000;
  }

  const years = Math.floor(days / 365);
  if (years === 1 && pageData.DATETIMESTRINGS['tool.ago_1year']) {
    // Datetime is one year ago.
    object.textContent = pageData.DATETIMESTRINGS['tool.ago_1year'];
  } else if (years === 2 && pageData.DATETIMESTRINGS['tool.ago_2years']) {
    // Datetime is two year ago.
    object.textContent = pageData.DATETIMESTRINGS['tool.ago_2years'];
  } else {
    // Datetime is more than a year ago.
    object.textContent = GetPluralizedStringOrFallback('tool.ago_years', years, 'year');
  }
  return 365 * 24 * 60 * 60 * 1000;
}

/** Update the displayed text of one relative-time DOM element with its human-readable, localized relative time string. */
function UpdateRelativeTime(object) {
  const next = DoUpdateRelativeTime(object);
  if (next !== null) setTimeout(() => { UpdateRelativeTime(object) }, next);
}

/** Update the displayed text of all relative-time DOM elements with their respective human-readable, localized relative time string. */
function UpdateAllRelativeTimes() {
  for (const object of document.querySelectorAll('relative-time')) UpdateRelativeTime(object);
}

document.addEventListener('DOMContentLoaded', UpdateAllRelativeTimes);
