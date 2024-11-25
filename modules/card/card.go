// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package card

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"net/http"
	"strings"
	"time"

	_ "image/gif"  // for processing gif images
	_ "image/jpeg" // for processing jpeg images
	_ "image/png"  // for processing png images

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/goregular"

	_ "golang.org/x/image/webp" // for processing webp images
)

type Card struct {
	Img    *image.RGBA
	Font   *truetype.Font
	Margin int
}

// NewCard creates a new card with the given dimensions in pixels
func NewCard(width, height int) (*Card, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	return &Card{
		Img:    img,
		Font:   font,
		Margin: 0,
	}, nil
}

// Split splits the card horizontally or vertically by a given percentage; the first card returned has the percentage
// size, and the second card has the remainder.  Both cards draw to a subsection of the same image buffer.
func (c *Card) Split(vertical bool, percentage int) (*Card, *Card) {
	bounds := c.Img.Bounds()
	bounds = image.Rect(bounds.Min.X+c.Margin, bounds.Min.Y+c.Margin, bounds.Max.X-c.Margin, bounds.Max.Y-c.Margin)
	if vertical {
		mid := (bounds.Dx() * percentage / 100) + bounds.Min.X
		subleft := c.Img.SubImage(image.Rect(bounds.Min.X, bounds.Min.Y, mid, bounds.Max.Y)).(*image.RGBA)
		subright := c.Img.SubImage(image.Rect(mid, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)).(*image.RGBA)
		return &Card{Img: subleft, Font: c.Font},
			&Card{Img: subright, Font: c.Font}
	}
	mid := (bounds.Dy() * percentage / 100) + bounds.Min.Y
	subtop := c.Img.SubImage(image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, mid)).(*image.RGBA)
	subbottom := c.Img.SubImage(image.Rect(bounds.Min.X, mid, bounds.Max.X, bounds.Max.Y)).(*image.RGBA)
	return &Card{Img: subtop, Font: c.Font},
		&Card{Img: subbottom, Font: c.Font}
}

// SetMargin sets the margins for the card
func (c *Card) SetMargin(margin int) {
	c.Margin = margin
}

type VAlign int64
type HAlign int64

const (
	Top VAlign = iota
	Middle
	Bottom
)

const (
	Left HAlign = iota
	Center
	Right
)

// DrawText draws text within the card, respecting margins and alignment
func (c *Card) DrawText(text string, text_color color.Color, size_pt float64, valign VAlign, halign HAlign) ([]string, error) {
	ft := freetype.NewContext()
	ft.SetDPI(72)
	ft.SetFont(c.Font)
	ft.SetFontSize(size_pt)
	ft.SetClip(c.Img.Bounds())
	ft.SetDst(c.Img)
	ft.SetSrc(image.NewUniform(text_color))

	font_height := ft.PointToFixed(size_pt).Ceil()
	offscreenDraw := freetype.Pt(0, -1000)

	bounds := c.Img.Bounds()
	bounds = image.Rect(bounds.Min.X+c.Margin, bounds.Min.Y+c.Margin, bounds.Max.X-c.Margin, bounds.Max.Y-c.Margin)
	box_width, box_height := bounds.Size().X, bounds.Size().Y
	// draw.Draw(c.Img, bounds, image.NewUniform(color.Gray{128}), image.Point{}, draw.Src) // Debug draw box

	// Try to apply wrapping to this text; we'll find the most text that will fit into one line, record that line, move
	// on.  We precalculate each line before drawing so that we can support valign="middle" correctly which requires
	// knowing the total height, which is related to how many lines we'll have.
	lines := make([]string, 0)
	text_words := strings.Split(text, " ")
	current_line := ""
	height_total := 0

	for {
		if len(text_words) == 0 {
			// Ran out of words.
			if current_line != "" {
				height_total += font_height
				lines = append(lines, current_line)
			}
			break
		}

		next_word := text_words[0]
		proposed_line := current_line
		if proposed_line != "" {
			proposed_line += " "
		}
		proposed_line += next_word

		proposed_line_width, err := ft.DrawString(proposed_line, offscreenDraw)
		if err != nil {
			return nil, err
		}
		if proposed_line_width.X.Ceil() > box_width {
			// no, proposed line is too big; we'll use the last "current_line"
			height_total += font_height
			if current_line != "" {
				lines = append(lines, current_line)
				current_line = ""
				// leave next_word in text_words and keep going
			} else {
				// just next_word by itself doesn't fit on a line; well, we can't skip it, but we'll consume it
				// regardless as a line by itself.  It will be clipped by the drawing routine.  We'll ignore it for the
				// widest_line calc.
				lines = append(lines, next_word)
				text_words = text_words[1:]
			}
		} else {
			// yes, it will fit
			current_line = proposed_line
			text_words = text_words[1:]
		}
	}

	text_y := 0
	if valign == Top {
		text_y = font_height
	} else if valign == Bottom {
		text_y = box_height - height_total + font_height
	} else if valign == Middle {
		extra_space := box_height - height_total
		text_y = (extra_space / 2) + font_height
	}

	for _, line := range lines {
		line_width, err := ft.DrawString(line, offscreenDraw)
		if err != nil {
			return nil, err
		}

		text_x := 0
		if halign == Left {
			text_x = 0
		} else if halign == Right {
			text_x = box_width - line_width.X.Ceil()
		} else if halign == Center {
			text_x = (box_width - line_width.X.Ceil()) / 2
		}

		pt := freetype.Pt(bounds.Min.X+text_x, bounds.Min.Y+text_y)
		_, err = ft.DrawString(line, pt)
		if err != nil {
			return nil, err
		}

		text_y += font_height
	}

	return lines, nil
}

// DrawImage fills the card with an image, scaled to maintain the original aspect ratio and centered with respect to the non-filled dimension
func (c *Card) DrawImage(img image.Image) {
	bounds := c.Img.Bounds()
	targetRect := image.Rect(bounds.Min.X+c.Margin, bounds.Min.Y+c.Margin, bounds.Max.X-c.Margin, bounds.Max.Y-c.Margin)
	srcBounds := img.Bounds()
	srcAspect := float64(srcBounds.Dx()) / float64(srcBounds.Dy())
	targetAspect := float64(targetRect.Dx()) / float64(targetRect.Dy())

	scale := 0.0
	if srcAspect > targetAspect {
		// Image is wider than target, scale by width
		scale = float64(targetRect.Dx()) / float64(srcBounds.Dx())
	} else {
		// Image is taller or equal, scale by height
		scale = float64(targetRect.Dy()) / float64(srcBounds.Dy())
	}

	newWidth := int(float64(srcBounds.Dx()) * scale)
	newHeight := int(float64(srcBounds.Dy()) * scale)

	// Center the image within the target rectangle
	offsetX := (targetRect.Dx() - newWidth) / 2
	offsetY := (targetRect.Dy() - newHeight) / 2

	scaledRect := image.Rect(targetRect.Min.X+offsetX, targetRect.Min.Y+offsetY, targetRect.Min.X+offsetX+newWidth, targetRect.Min.Y+offsetY+newHeight)
	draw.CatmullRom.Scale(c.Img, scaledRect, img, srcBounds, draw.Over, nil)
}

func fallbackImage() image.Image {
	// can't usage image.Uniform(color.White) because it's infinitely sized causing a panic in the scaler in DrawImage
	rect := image.Rect(0, 0, 1, 1)
	img := image.NewRGBA(rect)
	white := color.RGBA{255, 255, 255, 255}
	img.Set(0, 0, white)
	return img
}

// As defensively as possible, attempt to load an image from a presumed external and untrusted URL
func (c *Card) fetchExternalImage(url string) image.Image {
	// Use a short timeout; in the event of any failure we'll be logging and returning a placeholder, but we don't want
	// this rendering process to be slowed down
	client := &http.Client{
		Timeout: 1 * time.Second, // 1 second timeout
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Warn("error when fetching external image from %s: %w", url, err)
		return fallbackImage()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn("non-OK error code when fetching external image from %s: %s", url, resp.Status)
		return fallbackImage()
	}

	contentType := resp.Header.Get("Content-Type")
	// Support content types are in-sync with the allowed custom avatar file types
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/gif" && contentType != "image/webp" {
		log.Warn("fetching external image returned unsupported Content-Type which was ignored: %s", contentType)
		return fallbackImage()
	}

	body := io.LimitReader(resp.Body, setting.Avatar.MaxFileSize)
	bodyBytes, err := io.ReadAll(body)
	if int64(len(bodyBytes)) >= setting.Avatar.MaxFileSize {
		log.Warn("while fetching external image response size hit MaxFileSize (%d) and was discarded from url %s", setting.Avatar.MaxFileSize, url)
		return fallbackImage()
	}

	bodyBuffer := bytes.NewReader(bodyBytes)
	imgCfg, imgType, err := image.DecodeConfig(bodyBuffer)
	if err != nil {
		log.Warn("error when decoding external image from %s: %w", url, err)
		return fallbackImage()
	}

	// Verify that we have a match between actual data understood in the image body and the reported Content-Type
	if (contentType == "image/png" && imgType != "png") ||
		(contentType == "image/jpeg" && imgType != "jpeg") ||
		(contentType == "image/gif" && imgType != "gif") ||
		(contentType == "image/webp" && imgType != "webp") {
		log.Warn("while fetching external image, mismatched image body (%s) and Content-Type (%s)", imgType, contentType)
		return fallbackImage()
	}

	// do not process image which is too large, it would consume too much memory
	if imgCfg.Width > setting.Avatar.MaxWidth {
		log.Warn("while fetching external image, width %d exceeds Avatar.MaxWidth %d", imgCfg.Width, setting.Avatar.MaxWidth)
		return fallbackImage()
	}
	if imgCfg.Height > setting.Avatar.MaxHeight {
		log.Warn("while fetching external image, height %d exceeds Avatar.MaxHeight %d", imgCfg.Height, setting.Avatar.MaxHeight)
		return fallbackImage()
	}

	bodyBuffer.Seek(0, io.SeekStart) // reset for actual decode
	img, imgType, err := image.Decode(bodyBuffer)
	if err != nil {
		log.Warn("error when decoding external image from %s: %w", url, err)
		return fallbackImage()
	}

	return img
}

func (c *Card) DrawExternalImage(url string) {
	image := c.fetchExternalImage(url)
	c.DrawImage(image)
}
