package pix

import (
	"encoding/binary"
	"errors"
	"os"
)

// Pictures are stored in two layouts. The embedded art has a width byte, a
// height byte and one terminal palette index per pixel. Pictures the image
// model draws start with "PX2", then the cell width and height, the width
// and height as big-endian 16-bit numbers, the number of palette entries
// (0 meaning the terminal palette), that many RGB triples and the pixels.
var magic = []byte("PX2")

const headerSize = 2

var errFormat = errors.New("pix: not a picture")

// Decode reads a picture in either layout.
func Decode(data []byte) (Image, error) {
	if len(data) >= len(magic)+7 && string(data[:len(magic)]) == string(magic) {
		return decodeV2(data[len(magic):])
	}

	if len(data) < headerSize || len(data) != headerSize+int(data[0])*int(data[1]) {
		return Image{}, errFormat
	}

	return Image{W: int(data[0]), H: int(data[1]), Pix: data[headerSize:]}, nil
}

func decodeV2(data []byte) (Image, error) {
	img := Image{CellW: int(data[0]), CellH: int(data[1])}
	img.W, img.H = int(binary.BigEndian.Uint16(data[2:])), int(binary.BigEndian.Uint16(data[4:]))
	entries := int(data[6])
	data = data[7:]

	if len(data) != entries*3+img.W*img.H {
		return Image{}, errFormat
	}

	if entries > 0 {
		img.Palette = make([][3]uint8, entries)
		for index := range img.Palette {
			copy(img.Palette[index][:], data[index*3:])
		}
	}

	img.Pix = data[entries*3:]

	return img, nil
}

// Encode writes a picture in the second layout.
func Encode(img Image) []byte {
	cellW, cellH := img.cell()
	out := append([]byte(nil), magic...)
	out = append(out, byte(cellW), byte(cellH))
	out = binary.BigEndian.AppendUint16(out, uint16(img.W))
	out = binary.BigEndian.AppendUint16(out, uint16(img.H))
	out = append(out, byte(len(img.Palette)))

	for _, rgb := range img.Palette {
		out = append(out, rgb[:]...)
	}

	return append(out, img.Pix...)
}

// ReadFile loads a picture from disk.
func ReadFile(path string) (Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Image{}, err
	}

	return Decode(data)
}

// WriteFile stores a picture on disk.
func WriteFile(path string, img Image) error {
	return os.WriteFile(path, Encode(img), 0o644)
}
