// Package gridimage draws the weekly availability grid as a PNG image, for the "image" style
// of the weekly message (colored rounded cells, like a heatmap).
package gridimage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"sync"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Sizes in 1x pixels; the image is drawn at 2x so Discord's downscaling stays crisp.
const (
	scale      = 2
	padding    = 4
	rowLabelW  = 44
	headerH    = 24
	hourCellW  = 52
	wideCellW  = 120 // half-day layout: few, long column labels
	cellH      = 32
	gap        = 6
	radius     = 7
	countSize  = 15
	headerSize = 12
	rowSize    = 14
)

// Transparent background: labels and cells read on both light and dark Discord themes.
var (
	colorLabel = color.RGBA{0x94, 0x9b, 0xa4, 0xff}
	colorText  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	colorEmpty = color.RGBA{0x80, 0x84, 0x8e, 0x26}
	colorLow   = color.RGBA{0x4e, 0x50, 0x58, 0xff}
	colorMid   = color.RGBA{0xb0, 0x7a, 0x12, 0xff}
	colorHigh  = color.RGBA{0x24, 0x80, 0x46, 0xff}
)

// parseFonts parses the fonts once; they can be shared, unlike the faces drawn with them.
var parseFonts = sync.OnceValues(func() (fonts [2]*opentype.Font, err error) {
	if fonts[0], err = opentype.Parse(gobold.TTF); err != nil {
		return fonts, err
	}
	fonts[1], err = opentype.Parse(goregular.TTF)
	return fonts, err
})

type faces struct {
	count, header, row font.Face
}

// newFaces returns the faces of one image: a font.Face is not safe for concurrent use,
// and guilds render their grid in parallel.
func newFaces() (faces, error) {
	fonts, err := parseFonts()
	if err != nil {
		return faces{}, err
	}
	bold, regular := fonts[0], fonts[1]
	face := func(f *opentype.Font, size float64) font.Face {
		fc, faceErr := opentype.NewFace(f, &opentype.FaceOptions{Size: size * scale, DPI: 72, Hinting: font.HintingFull})
		if err == nil {
			err = faceErr
		}
		return fc
	}
	fs := faces{count: face(bold, countSize), header: face(regular, headerSize), row: face(bold, rowSize)}
	return fs, err
}

// Render draws view.Counts as a PNG: one row per day, one rounded cell per slot,
// colored by headcount relative to the best slot of the week.
func Render(view appavailability.WeeklyView) ([]byte, error) {
	img, err := drawGrid(view)
	if err != nil {
		return nil, err
	}
	// PNG at the fastest level: a few milliseconds where the pure Go WebP encoder took
	// ~300 ms, for a file a few dozen KB heavier that Discord converts for display anyway.
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func drawGrid(view appavailability.WeeklyView) (*image.RGBA, error) {
	fs, err := newFaces()
	if err != nil {
		return nil, fmt.Errorf("load fonts: %w", err)
	}

	cellW := hourCellW
	if len(view.Columns) <= 4 {
		cellW = wideCellW
	}
	width := padding*2 + rowLabelW + len(view.Columns)*(cellW+gap) - gap
	height := padding*2 + headerH + len(view.Rows)*(cellH+gap) - gap
	img := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))

	for col, label := range view.Columns {
		x := padding + rowLabelW + col*(cellW+gap)
		drawText(img, fs.header, label, colorLabel, (x+cellW/2)*scale, (padding+headerH/2-2)*scale)
	}
	for row, label := range view.Rows {
		y := padding + headerH + row*(cellH+gap)
		drawTextLeft(img, fs.row, label, colorLabel, padding*scale, (y+cellH/2)*scale)
		for col := range view.Columns {
			count := view.Counts[row][col]
			x := padding + rowLabelW + col*(cellW+gap)
			cell := image.Rect(x*scale, y*scale, (x+cellW)*scale, (y+cellH)*scale)
			fillRoundRect(img, cell, radius*scale, cellColor(count, view.Peak))
			if count > 0 {
				drawText(img, fs.count, fmt.Sprint(count), colorText, (x+cellW/2)*scale, (y+cellH/2)*scale)
			}
		}
	}

	return img, nil
}

// cellColor uses the same thresholds as the text styles: >= 75% of the best slot is
// green, >= 40% yellow, anything else grey.
func cellColor(count, peak int) color.RGBA {
	switch {
	case count == 0:
		return colorEmpty
	case count*4 >= peak*3:
		return colorHigh
	case count*5 >= peak*2:
		return colorMid
	default:
		return colorLow
	}
}

// fillRoundRect paints r with rounded corners of the given radius, blending over dst.
// Corner pixels are anti-aliased: their opacity is the part of the pixel inside the arc.
func fillRoundRect(dst *image.RGBA, r image.Rectangle, rad int, c color.RGBA) {
	w, h := r.Dx(), r.Dy()
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	radius := float64(rad)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Pixel center relative to the nearest corner's arc center; zero outside corners.
			dx := max(radius-(float64(x)+0.5), (float64(x)+0.5)-float64(w-rad), 0)
			dy := max(radius-(float64(y)+0.5), (float64(y)+0.5)-float64(h-rad), 0)
			coverage := min(max(radius-math.Hypot(dx, dy)+0.5, 0), 1)
			mask.SetAlpha(x, y, color.Alpha{A: uint8(coverage * 0xff)})
		}
	}
	draw.DrawMask(dst, r, image.NewUniform(c), image.Point{}, mask, image.Point{}, draw.Over)
}

// drawText draws s centered on (cx, cy).
func drawText(dst *image.RGBA, face font.Face, s string, c color.RGBA, cx, cy int) {
	w := font.MeasureString(face, s).Ceil()
	drawTextLeft(dst, face, s, c, cx-w/2, cy)
}

// drawTextLeft draws s starting at x, vertically centered on cy.
func drawTextLeft(dst *image.RGBA, face font.Face, s string, c color.RGBA, x, cy int) {
	m := face.Metrics()
	baseline := cy + (m.Ascent-m.Descent).Ceil()/2
	d := font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(s)
}
