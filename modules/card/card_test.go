// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package card

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/test"

	"github.com/golang/freetype/truetype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/goregular"
)

func TestNewCard(t *testing.T) {
	width, height := 100, 50
	card, err := NewCard(width, height)
	require.NoError(t, err, "No error should occur when creating a new card")
	assert.NotNil(t, card, "Card should not be nil")
	assert.Equal(t, width, card.Img.Bounds().Dx(), "Width should match the provided width")
	assert.Equal(t, height, card.Img.Bounds().Dy(), "Height should match the provided height")

	// Checking default margin
	assert.Equal(t, 0, card.Margin, "Default margin should be 0")

	// Checking font parsing
	originalFont, _ := truetype.Parse(goregular.TTF)
	assert.Equal(t, originalFont, card.Font, "Fonts should be equivalent")
}

func TestSplit(t *testing.T) {
	// Note: you normally wouldn't split the same card twice as draw operations would start to overlap each other; but
	// it's fine for this limited scope test
	card, _ := NewCard(200, 100)

	// Test vertical split
	leftCard, rightCard := card.Split(true, 50)
	assert.Equal(t, 100, leftCard.Img.Bounds().Dx(), "Left card should have half the width of original")
	assert.Equal(t, 100, leftCard.Img.Bounds().Dy(), "Left card height unchanged by split")
	assert.Equal(t, 100, rightCard.Img.Bounds().Dx(), "Right card should have half the width of original")
	assert.Equal(t, 100, rightCard.Img.Bounds().Dy(), "Right card height unchanged by split")

	// Test horizontal split
	topCard, bottomCard := card.Split(false, 50)
	assert.Equal(t, 200, topCard.Img.Bounds().Dx(), "Top card width unchanged by split")
	assert.Equal(t, 50, topCard.Img.Bounds().Dy(), "Top card should have half the height of original")
	assert.Equal(t, 200, bottomCard.Img.Bounds().Dx(), "Bottom width unchanged by split")
	assert.Equal(t, 50, bottomCard.Img.Bounds().Dy(), "Bottom card should have half the height of original")
}

func TestDrawTextSingleLine(t *testing.T) {
	card, _ := NewCard(300, 100)
	lines, err := card.DrawText("This is a single line", color.Black, 12, Middle, Center)
	require.NoError(t, err, "No error should occur when drawing text")
	assert.Len(t, lines, 1, "Should be exactly one line")
	assert.Equal(t, "This is a single line", lines[0], "Text should match the input")
}

func TestDrawTextLongLine(t *testing.T) {
	card, _ := NewCard(300, 100)
	text := "This text is definitely too long to fit in three hundred pixels width without wrapping"
	lines, err := card.DrawText(text, color.Black, 12, Middle, Center)
	require.NoError(t, err, "No error should occur when drawing text")
	assert.Len(t, lines, 2, "Text should wrap into multiple lines")
	assert.Equal(t, "This text is definitely too long to fit in three hundred", lines[0], "Text should match the input")
	assert.Equal(t, "pixels width without wrapping", lines[1], "Text should match the input")
}

func TestDrawTextWordTooLong(t *testing.T) {
	card, _ := NewCard(300, 100)
	text := "Line 1 Superduperlongwordthatcannotbewrappedbutshouldenduponitsownsingleline Line 3"
	lines, err := card.DrawText(text, color.Black, 12, Middle, Center)
	require.NoError(t, err, "No error should occur when drawing text")
	assert.Len(t, lines, 3, "Text should create two lines despite long word")
	assert.Equal(t, "Line 1", lines[0], "First line should contain text before the long word")
	assert.Equal(t, "Superduperlongwordthatcannotbewrappedbutshouldenduponitsownsingleline", lines[1], "Second line couldn't wrap the word so it just overflowed")
	assert.Equal(t, "Line 3", lines[2], "Third line continued with wrapping")
}

func TestFetchExternalImageServer(t *testing.T) {
	blackPng, err := base64.URLEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQAAAAA3bvkkAAAACklEQVR4AWNgAAAAAgABc3UBGAAAAABJRU5ErkJggg==")
	if err != nil {
		t.Error(err)
		return
	}

	var tooWideBuf bytes.Buffer
	imgTooWide := image.NewGray(image.Rect(0, 0, 16001, 10))
	err = png.Encode(&tooWideBuf, imgTooWide)
	if err != nil {
		t.Error(err)
		return
	}
	imgTooWidePng := tooWideBuf.Bytes()

	var tooTallBuf bytes.Buffer
	imgTooTall := image.NewGray(image.Rect(0, 0, 10, 16002))
	err = png.Encode(&tooTallBuf, imgTooTall)
	if err != nil {
		t.Error(err)
		return
	}
	imgTooTallPng := tooTallBuf.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/timeout":
			// Simulate a timeout by taking a long time to respond
			time.Sleep(8 * time.Second)
			w.Header().Set("Content-Type", "image/png")
			w.Write(blackPng)
		case "/notfound":
			http.NotFound(w, r)
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(blackPng)
		case "/weird-content":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html></html>"))
		case "/giant-response":
			w.Header().Set("Content-Type", "image/png")
			w.Write(make([]byte, 10485760))
		case "/invalid.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(make([]byte, 100))
		case "/mismatched.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(blackPng) // valid png, but wrong content-type
		case "/too-wide.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(imgTooWidePng)
		case "/too-tall.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(imgTooTallPng)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	tests := []struct {
		name            string
		url             string
		expectedSuccess bool
		expectedLog     string
	}{
		{
			name:            "timeout error",
			url:             "/timeout",
			expectedSuccess: false,
			expectedLog:     "error when fetching external image from",
		},
		{
			name:            "external fetch success",
			url:             "/image.png",
			expectedSuccess: true,
			expectedLog:     "",
		},
		{
			name:            "404 fallback",
			url:             "/notfound",
			expectedSuccess: false,
			expectedLog:     "non-OK error code when fetching external image",
		},
		{
			name:            "unsupported content type",
			url:             "/weird-content",
			expectedSuccess: false,
			expectedLog:     "fetching external image returned unsupported Content-Type",
		},
		{
			name:            "response too large",
			url:             "/giant-response",
			expectedSuccess: false,
			expectedLog:     "while fetching external image response size hit MaxFileSize",
		},
		{
			name:            "invalid png",
			url:             "/invalid.png",
			expectedSuccess: false,
			expectedLog:     "error when decoding external image",
		},
		{
			name:            "mismatched content type",
			url:             "/mismatched.jpg",
			expectedSuccess: false,
			expectedLog:     "while fetching external image, mismatched image body",
		},
		{
			name:            "too wide",
			url:             "/too-wide.png",
			expectedSuccess: false,
			expectedLog:     "while fetching external image, width 16001 exceeds Avatar.MaxWidth",
		},
		{
			name:            "too tall",
			url:             "/too-tall.png",
			expectedSuccess: false,
			expectedLog:     "while fetching external image, height 16002 exceeds Avatar.MaxHeight",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// stopMark is used as a logging boundary to verify that the expected message (testCase.expectedLog) is
			// logged during the `fetchExternalImage` operation.  This is verified by a combination of checking that the
			// stopMark message was received, and that the filtered log (logFiltered[0]) was received.
			stopMark := fmt.Sprintf(">>>>>>>>>>>>>STOP: %s<<<<<<<<<<<<<<<", testCase.name)

			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
			logChecker.Filter(testCase.expectedLog).StopMark(stopMark)
			defer cleanup()

			card, _ := NewCard(100, 100)
			img, ok := card.fetchExternalImage(server.URL + testCase.url)

			if testCase.expectedSuccess {
				assert.True(t, ok, "expected success from fetchExternalImage")
				assert.NotNil(t, img)
			} else {
				assert.False(t, ok, "expected failure from fetchExternalImage")
				assert.Nil(t, img)
			}

			log.Info(stopMark)

			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped, "failed to find log stop mark")
			assert.True(t, logFiltered[0], "failed to find in log: '%s'", testCase.expectedLog)
		})
	}
}
