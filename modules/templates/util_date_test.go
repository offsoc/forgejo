// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"html/template"
	"testing"
	"time"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateTime(t *testing.T) {
	testTz, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	defer test.MockVariableValue(&setting.DefaultUILocation, testTz)()
	defer test.MockVariableValue(&setting.IsInTesting, false)()

	du := NewDateUtils()

	refTimeStr := "2018-01-01T00:00:00Z"
	refDateStr := "2018-01-01"
	refTime, _ := time.Parse(time.RFC3339, refTimeStr)
	refTimeStamp := timeutil.TimeStamp(refTime.Unix())

	for _, val := range []any{nil, 0, time.Time{}, timeutil.TimeStamp(0)} {
		for _, fun := range []func(val any) template.HTML{du.AbsoluteLong, du.AbsoluteShort, du.FullTime} {
			assert.EqualValues(t, "-", fun(val))
		}
	}

	actual := dateTimeLegacy("short", "invalid")
	assert.EqualValues(t, `-`, actual)

	actual = dateTimeLegacy("short", refTimeStr)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="short" day="numeric" date="2018-01-01T00:00:00Z">2018-01-01</absolute-date>`, actual)

	actual = du.AbsoluteShort(refTime)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="short" day="numeric" date="2018-01-01T00:00:00Z">2018-01-01</absolute-date>`, actual)

	actual = du.AbsoluteLong(refTime)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="long" day="numeric" date="2018-01-01T00:00:00Z">2018-01-01</absolute-date>`, actual)

	actual = dateTimeLegacy("short", refDateStr)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="short" day="numeric" date="2018-01-01T00:00:00-05:00">2018-01-01</absolute-date>`, actual)

	actual = du.AbsoluteShort(refTimeStamp)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="short" day="numeric" date="2017-12-31T19:00:00-05:00">2017-12-31</absolute-date>`, actual)

	actual = du.AbsoluteLong(refTimeStamp)
	assert.EqualValues(t, `<absolute-date weekday="" year="numeric" month="long" day="numeric" date="2017-12-31T19:00:00-05:00">2017-12-31</absolute-date>`, actual)

	actual = du.FullTime(refTimeStamp)
	assert.EqualValues(t, `<relative-time weekday="" year="numeric" format="datetime" month="short" day="numeric" hour="numeric" minute="numeric" second="numeric" data-tooltip-content data-tooltip-interactive="true" datetime="2017-12-31T19:00:00-05:00">2017-12-31 19:00:00 -05:00</relative-time>`, actual)
}

func TestTimeSince(t *testing.T) {
	testTz, _ := time.LoadLocation("America/New_York")
	defer test.MockVariableValue(&setting.DefaultUILocation, testTz)()
	defer test.MockVariableValue(&setting.IsInTesting, false)()

	du := NewDateUtils()
	assert.EqualValues(t, "-", du.TimeSince(nil))

	refTimeStr := "2018-01-01T00:00:00Z"
	refTime, _ := time.Parse(time.RFC3339, refTimeStr)

	actual := du.TimeSince(refTime)
	assert.EqualValues(t, `<relative-time prefix="" tense="past" datetime="2018-01-01T00:00:00Z" data-tooltip-content data-tooltip-interactive="true">2018-01-01 00:00:00 +00:00</relative-time>`, actual)

	actual = timeSinceTo(&refTime, time.Time{})
	assert.EqualValues(t, `<relative-time prefix="" tense="future" datetime="2018-01-01T00:00:00Z" data-tooltip-content data-tooltip-interactive="true">2018-01-01 00:00:00 +00:00</relative-time>`, actual)

	actual = timeSinceLegacy(timeutil.TimeStampNano(refTime.UnixNano()), nil)
	assert.EqualValues(t, `<relative-time prefix="" tense="past" datetime="2017-12-31T19:00:00-05:00" data-tooltip-content data-tooltip-interactive="true">2017-12-31 19:00:00 -05:00</relative-time>`, actual)
}
