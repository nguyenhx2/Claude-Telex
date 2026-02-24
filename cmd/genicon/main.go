// Command genicon generates app icon files (PNG + ICO) for Claude Telex.
// Usage: go run ./cmd/genicon
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	colorOn = color.RGBA{R: 232, G: 112, B: 64, A: 255}
	white   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

func main() {
	dir := filepath.Join("assets", "icon")
	os.MkdirAll(dir, 0o755)

	sizes := []int{16, 32, 48, 64, 128, 256}
	for _, s := range sizes {
		img := generateIcon(s)
		savePNG(filepath.Join(dir, "icon-"+itoa(s)+".png"), img)
	}

	ico := generateICO([]int{16, 32, 48, 256})
	os.WriteFile(filepath.Join(dir, "app.ico"), ico, 0o644)

	println("Generated icons in", dir)
}

func generateIcon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1.5

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
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
					R: uint8(float64(colorOn.R) * alpha),
					G: uint8(float64(colorOn.G) * alpha),
					B: uint8(float64(colorOn.B) * alpha),
					A: uint8(255 * alpha),
				})
			}
		}
	}

	drawText(img, "VN", size, white)
	return img
}

func drawText(img *image.RGBA, text string, size int, clr color.RGBA) {
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

func savePNG(path string, img *image.RGBA) {
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}

func generateICO(sizes []int) []byte {
	var entries [][]byte
	for _, s := range sizes {
		img := generateIcon(s)
		var buf bytes.Buffer
		png.Encode(&buf, img)
		entries = append(entries, buf.Bytes())
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))

	offset := uint32(6 + 16*len(entries))
	for i, s := range sizes {
		sz := byte(s)
		if s >= 256 {
			sz = 0
		}
		buf.WriteByte(sz)
		buf.WriteByte(sz)
		buf.WriteByte(0)
		buf.WriteByte(0)
		binary.Write(&buf, binary.LittleEndian, uint16(1))
		binary.Write(&buf, binary.LittleEndian, uint16(32))
		binary.Write(&buf, binary.LittleEndian, uint32(len(entries[i])))
		binary.Write(&buf, binary.LittleEndian, offset)
		offset += uint32(len(entries[i]))
	}

	for _, e := range entries {
		buf.Write(e)
	}
	return buf.Bytes()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
