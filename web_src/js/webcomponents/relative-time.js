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
    object.textContent = GetPluralizedString('tool.ago_mins', minutes).replace('%d', minutes);
    return 60 * 1000;
  }

  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    // Datetime is one or more hours but less than a day ago.
    object.textContent = GetPluralizedString('tool.ago_hours', hours).replace('%d', hours);
    return 60 * 60 * 1000;
  }

  const days = Math.floor(hours / 24);
  if (days < 365) {
    if (days < 7) {
      if (days === 1) {
        // Datetime is one day ago.
        object.textContent = pageData.DATETIMESTRINGS.AGO_1DAY;
      } else if (days === 2) {
        // Datetime is two days ago.
        object.textContent = pageData.DATETIMESTRINGS.AGO_2DAYS;
      } else {
        // Datetime is several days but less than a week ago.
        object.textContent = GetPluralizedString('tool.ago_days', days).replace('%d', days);
      }
      return 24 * 60 * 60 * 1000;
    }

    if (days < 30) {
      // Datetime is at least one week but less than a month ago.
      const weeks = Math.floor(days / 7);
      if (weeks === 1) {
        // Datetime is one week ago.
        object.textContent = pageData.DATETIMESTRINGS.AGO_1WEEK;
      } else if (weeks === 2) {
        // Datetime is two week ago.
        object.textContent = pageData.DATETIMESTRINGS.AGO_2WEEKS;
      } else {
        // Datetime is several weeks ago (but less than a month).
        object.textContent = GetPluralizedString('tool.ago_weeks', weeks).replace('%d', weeks);
      }
      return 7 * 24 * 60 * 60 * 1000;
    }

    // Datetime is at least one month but less than a year ago.
    const months = Math.floor(days / 30);
    if (months === 1) {
      // Datetime is one month ago.
      object.textContent = pageData.DATETIMESTRINGS.AGO_1MONTH;
    } else if (months === 2) {
      // Datetime is two months ago.
      object.textContent = pageData.DATETIMESTRINGS.AGO_2MONTHS;
    } else {
      // Datetime is several months ago (but less than a year).
      object.textContent = GetPluralizedString('tool.ago_months', months).replace('%d', months);
    }
    return 30 * 24 * 60 * 60 * 1000;
  }

  const years = Math.floor(days / 365);
  if (years === 1) {
    // Datetime is one year ago.
    object.textContent = pageData.DATETIMESTRINGS.AGO_1YEAR;
  } else if (years === 2) {
    // Datetime is two year ago.
    object.textContent = pageData.DATETIMESTRINGS.AGO_2YEARS;
  } else {
    // Datetime is more than a year ago.
    object.textContent = GetPluralizedString('tool.ago_years', years).replace('%d', years);
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
