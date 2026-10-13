// Package gridimage draws the weekly availability grid as a WebP image, for the "image" style
// of the weekly message (colored rounded cells, like a heatmap).
package gridimage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/HugoSmits86/nativewebp"
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

var (
	fontsOnce                      sync.Once
	fontsErr                       error
	countFace, headerFace, rowFace font.Face
)

func loadFonts() {
	bold, err := opentype.Parse(gobold.TTF)
	if err != nil {
		fontsErr = err
		return
	}
	regular, err := opentype.Parse(goregular.TTF)
	if err != nil {
		fontsErr = err
		return
	}
	face := func(f *opentype.Font, size float64) font.Face {
		fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size * scale, DPI: 72, Hinting: font.HintingFull})
		if err != nil && fontsErr == nil {
			fontsErr = err
		}
		return fc
	}
	countFace = face(bold, countSize)
	headerFace = face(regular, headerSize)
	rowFace = face(bold, rowSize)
}

// Render draws view.Counts as a lossless WebP: one row per day, one rounded cell per slot,
// colored by headcount relative to the best slot of the week.
func Render(view appavailability.WeeklyView) ([]byte, error) {
	img, err := drawGrid(view)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, &nativewebp.Options{CompressionLevel: nativewebp.BestCompression}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}

func drawGrid(view appavailability.WeeklyView) (*image.RGBA, error) {
	fontsOnce.Do(loadFonts)
	if fontsErr != nil {
		return nil, fmt.Errorf("load fonts: %w", fontsErr)
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
		drawText(img, headerFace, label, colorLabel, (x+cellW/2)*scale, (padding+headerH/2-2)*scale)
	}
	for row, label := range view.Rows {
		y := padding + headerH + row*(cellH+gap)
		drawTextLeft(img, rowFace, label, colorLabel, padding*scale, (y+cellH/2)*scale)
		for col := range view.Columns {
			count := view.Counts[row][col]
			x := padding + rowLabelW + col*(cellW+gap)
			cell := image.Rect(x*scale, y*scale, (x+cellW)*scale, (y+cellH)*scale)
			fillRoundRect(img, cell, radius*scale, cellColor(count, view.Peak))
			if count > 0 {
				drawText(img, countFace, fmt.Sprint(count), colorText, (x+cellW/2)*scale, (y+cellH/2)*scale)
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
func fillRoundRect(dst *image.RGBA, r image.Rectangle, rad int, c color.RGBA) {
	mask := image.NewAlpha(image.Rect(0, 0, r.Dx(), r.Dy()))
	w, h := r.Dx(), r.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Distance to the nearest corner center, only relevant inside the corner squares.
			cx, cy := x, y
			switch {
			case x < rad:
				cx = rad
			case x >= w-rad:
				cx = w - rad - 1
			}
			switch {
			case y < rad:
				cy = rad
			case y >= h-rad:
				cy = h - rad - 1
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= rad*rad {
				mask.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
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
