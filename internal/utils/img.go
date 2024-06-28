package utils

import (
	"errors"
	"image"
	"image/color"
	"math"
)

var ErrInvalidHexColor = errors.New("invalid hex color format")
var ErrFormatNotSupported = errors.New("format not supported")

func ParseHexColor(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, ErrInvalidHexColor
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = ErrInvalidHexColor
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = ErrInvalidHexColor
	}
	return
}

type settable interface {
	Set(x, y int, c color.Color)
}

var EmptyColor = color.RGBA{R: 255, G: 255, B: 255}

func RoundImage(m image.Image, rate float64) error {
	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	r := (float64(min(w, h)) / 2) * rate
	sm, ok := m.(settable)
	if !ok {
		// Check if image is YCbCr format.
		ym, ok := m.(*image.YCbCr)
		if !ok {
			return ErrFormatNotSupported
		}
		m = yCbCrToRGBA(ym)
		sm = m.(settable)
	}
	// Parallelize?
	for y := 0.0; y <= r; y++ {
		l := math.Round(r - math.Sqrt(2*y*r-y*y))
		for x := 0; x <= int(l); x++ {
			sm.Set(x-1, int(y)-1, EmptyColor)
		}
		for x := 0; x <= int(l); x++ {
			sm.Set(w-x, int(y)-1, EmptyColor)
		}
		for x := 0; x <= int(l); x++ {
			sm.Set(x-1, h-int(y), EmptyColor)
		}
		for x := 0; x <= int(l); x++ {
			sm.Set(w-x, h-int(y), EmptyColor)
		}
	}
	return nil
}

func yCbCrToRGBA(m image.Image) image.Image {
	b := m.Bounds()
	nm := image.NewRGBA(b)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			nm.Set(x, y, m.At(x, y))
		}
	}
	return nm
}
