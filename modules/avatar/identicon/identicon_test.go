// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build test_avatar_identicon

package identicon

import (
	"image/color"
	"image/png"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	dir, _ := os.Getwd()
	dir = dir + "/testdata"
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Errorf("can not save generated images to %s", dir)
	}

	backColor := color.White
	imgMaker, err := New(64, backColor, DarkColors...)
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		s := strconv.Itoa(i)
		img := imgMaker.Make([]byte(s))

		f, err := os.Create(dir + "/" + s + ".png")
		require.NoError(t, err)

		defer f.Close()
		err = png.Encode(f, img)
		require.NoError(t, err)
	}
}
