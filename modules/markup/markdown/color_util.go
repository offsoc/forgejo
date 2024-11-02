// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import "regexp"

var (
	hexRGB = regexp.MustCompile(`^#([0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$`)
	hsl    = regexp.MustCompile(`^hsl\([ ]*([012]?[0-9]{1,2}|3[0-5][0-9]|360),[ ]*([0-9]{0,2}|100)\%,[ ]*([0-9]{0,2}|100)\%\)$`)
	hsla   = regexp.MustCompile(`^hsla\(([ ]*[012]?[0-9]{1,2}|3[0-5][0-9]|360),[ ]*([0-9]{0,2}|100)\%,[ ]*([0-9]{0,2}|100)\%,[ ]*(1|1\.0|0|(0\.[0-9]+))\)$`)
	rgb    = regexp.MustCompile(`^rgb\(([ ]*((([0-9]{1,2}|100)\%)|(([01]?[0-9]{1,2})|(2[0-4][0-9])|(25[0-5]))),){2}([ ]*((([0-9]{1,2}|100)\%)|(([01]?[0-9]{1,2})|(2[0-4][0-9])|(25[0-5]))))\)$`)
	rgba   = regexp.MustCompile(`^rgba\(([ ]*((([0-9]{1,2}|100)\%)|(([01]?[0-9]{1,2})|(2[0-4][0-9])|(25[0-5]))),){3}[ ]*(1(\.0)?|0|(0\.[0-9]+))\)$`)
)

// matchColor return if color is in the form of hex RGB, HSL(A) or RGB(A).
func matchColor(color string) bool {
	return hexRGB.MatchString(color) || rgb.MatchString(color) || rgba.MatchString(color) || hsl.MatchString(color) || hsla.MatchString(color)
}
