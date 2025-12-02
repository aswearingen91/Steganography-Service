package steg

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
)

// EmbedBytes will embed data into img using LSB on R,G,B channels.
// Returns PNG bytes of the modified image.
func EmbedBytes(src image.Image, data []byte) ([]byte, error) {
	// Convert to RGBA for easier manipulation
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)

	// Prefix length: 8 bytes length (big-endian) + payload
	payload := prefixLen(data)

	capBits := (rgba.Bounds().Dx() * rgba.Bounds().Dy() * 3)
	if len(payload)*8 > capBits {
		return nil, errors.New("image doesn't have enough capacity to store payload")
	}

	bitIdx := 0
	for y := rgba.Rect.Min.Y; y < rgba.Rect.Max.Y; y++ {
		for x := rgba.Rect.Min.X; x < rgba.Rect.Max.X; x++ {
			offset := rgba.PixOffset(x, y)
			// channels: R,G,B,A
			for ch := 0; ch < 3; ch++ { // R,G,B only
				if bitIdx >= len(payload)*8 {
					break
				}
				byteIdx := bitIdx / 8
				bitInByte := 7 - (bitIdx % 8) // MSB first
				bit := (payload[byteIdx] >> bitInByte) & 1

				// set LSB of channel
				rgba.Pix[offset+ch] = (rgba.Pix[offset+ch] & 0xFE) | byte(bit)
				bitIdx++
			}
			if bitIdx >= len(payload)*8 {
				break
			}
		}
		if bitIdx >= len(payload)*8 {
			break
		}
	}

	// encode to PNG and return bytes
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, rgba); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExtractBytes reads length (first 8 bytes) and extracts that many payload bytes
// Returns the extracted payload (without the 8-byte length prefix).
func ExtractBytes(src image.Image) ([]byte, error) {
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)

	// Flatten all LSBs into a bit slice
	var bits []uint8
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			offset := rgba.PixOffset(x, y)
			for ch := 0; ch < 3; ch++ { // R,G,B only
				bits = append(bits, rgba.Pix[offset+ch]&1)
			}
		}
	}

	if len(bits) < 64 {
		return nil, errors.New("image too small to contain length header")
	}

	// Read first 64 bits as payload length
	var lenBytes [8]byte
	for i := 0; i < 8; i++ {
		var bval byte
		for j := 0; j < 8; j++ {
			bval = (bval << 1) | bits[i*8+j]
		}
		lenBytes[i] = bval
	}
	payloadLen := bytesToUint64(lenBytes[:])

	if int(payloadLen)*8 > len(bits)-64 {
		return nil, errors.New("declared payload longer than capacity")
	}

	// Read payload bits
	payload := make([]byte, payloadLen)
	for i := uint64(0); i < payloadLen; i++ {
		var bval byte
		for j := 0; j < 8; j++ {
			bval = (bval << 1) | bits[64+i*8+uint64(j)]
		}
		payload[i] = bval
	}

	return payload, nil
}

// helpers

func prefixLen(data []byte) []byte {
	lb := make([]byte, 8)
	putUint64(lb, uint64(len(data)))
	return append(lb, data...)
}

func putUint64(b []byte, v uint64) {
	_ = b[7] // bounds check
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

func bytesToUint64(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

// DecodeImageFromReader uses image.Decode (supports PNG, JPEG, GIF)
func DecodeImageFromReader(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	return img, format, err
}
