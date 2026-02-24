// Package icon generates tray icons programmatically and encodes them as ICO.
// Does not require any SVG library.
package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	colorOn     = color.RGBA{R: 232, G: 112, B: 64, A: 255}  // #E87040 Claude orange
	colorOff    = color.RGBA{R: 107, G: 114, B: 128, A: 255} // #6b7280 gray
	colorUpdate = color.RGBA{R: 59, G: 130, B: 246, A: 255}  // #3b82f6 blue
	white       = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

const (
	StateOff = iota
	StateOn
	StateUpdate
)

// ICO returns an ICO-encoded icon (32×32) for use with systray on Windows.
// systray on Windows requires ICO format bytes.
func ICO(state int) []byte {
	img := generateImage(state, 32)
	return encodeICO(img)
}

// PNG returns a PNG-encoded icon (64×64) for platforms that accept PNG.
func PNG(state int) []byte {
	img := generateImage(state, 64)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func generateImage(state int, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	c := colorOff
	if state == StateOn {
		c = colorOn
	} else if state == StateUpdate {
		c = colorUpdate
	}

	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1.5

	// Anti-aliased circle
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Sample 4 sub-pixels for simple AA
			alpha := 0.0
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					px := float64(x) + float64(sx)*0.5 + 0.25
					py := float64(y) + float64(sy)*0.5 + 0.25
					dx, dy := px-cx, py-cy
					if math.Sqrt(dx*dx+dy*dy) <= r {
						alpha += 0.25
					}
				}
			}
			if alpha > 0 {
				img.SetRGBA(x, y, color.RGBA{
					R: uint8(float64(c.R) * alpha),
					G: uint8(float64(c.G) * alpha),
					B: uint8(float64(c.B) * alpha),
					A: uint8(255 * alpha),
				})
			}
		}
	}

	// Draw "VN" text centered
	drawTextCentered(img, "VN", size, white)
	return img
}

func drawTextCentered(img *image.RGBA, text string, size int, clr color.RGBA) {
	// Scale font size to ~45% of icon size, with minimum for small icons
	fontSize := float64(size) * 0.45
	if fontSize < 10 {
		fontSize = 10
	}

	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return
	}
	defer face.Close()

	// Measure advance
	adv := fixed.Int26_6(0)
	for _, r := range text {
		a, ok := face.GlyphAdvance(r)
		if ok {
			adv += a
		}
	}
	textW := adv.Round()
	ascent := face.Metrics().Ascent.Round()

	x := (size - textW) / 2
	y := (size+ascent)/2 - 1

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// encodeICO wraps an RGBA image into a single-image ICO binary.
// ICO format: ICONDIR (6 bytes) + ICONDIRENTRY (16 bytes) + PNG data
func encodeICO(img *image.RGBA) []byte {
	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, img)
	pngBytes := pngBuf.Bytes()

	size := img.Bounds().Dx()

	var buf bytes.Buffer
	// ICONDIR
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type=1 (icon)
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // count=1

	// ICONDIRENTRY (16 bytes)
	if size >= 256 {
		buf.WriteByte(0) // 0 = 256
	} else {
		buf.WriteByte(byte(size))
	}
	if size >= 256 {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(byte(size))
	}
	buf.WriteByte(0)                                               // color count (0=no palette)
	buf.WriteByte(0)                                               // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // planes
	binary.Write(&buf, binary.LittleEndian, uint16(32))            // bit count
	binary.Write(&buf, binary.LittleEndian, uint32(len(pngBytes))) // bytes in res
	binary.Write(&buf, binary.LittleEndian, uint32(6+16))          // offset = ICONDIR + ICONDIRENTRY

	// PNG data
	buf.Write(pngBytes)
	return buf.Bytes()
}
