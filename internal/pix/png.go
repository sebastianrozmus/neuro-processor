package pix

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// PNG encodes the picture as a PNG, every pixel scaled up to a square of
// the given size; transparent pixels stay transparent.
func (img Image) PNG(scale int) []byte {
	scale = max(scale, 1)
	out := image.NewNRGBA(image.Rect(0, 0, img.W*scale, img.H*scale))

	for y := range img.H {
		for x := range img.W {
			index := img.Pix[y*img.W+x]
			if index == 0 {
				continue
			}

			rgb := img.rgb(index)
			pixel := color.NRGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}

			for dy := range scale {
				for dx := range scale {
					out.SetNRGBA(x*scale+dx, y*scale+dy, pixel)
				}
			}
		}
	}

	var buffer bytes.Buffer
	_ = png.Encode(&buffer, out)

	return buffer.Bytes()
}
