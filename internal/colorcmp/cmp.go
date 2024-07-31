package colorcmp

import (
	"image/color"
	"math"
)

// comparator is a function that returns a difference between two colors in
// range 0.0..1.0 (0.0 - same colors, 1.0 - totally different colors).
type comparator func(color.Color, color.Color) float64

// CmpEuclidean returns Euclidean difference of two colors.
//
// https://en.wikipedia.org/wiki/Color_difference#Euclidean
func CmpEuclidean(color1 color.Color, color2 color.Color) float64 {
	const maxDiff = 113509.94967402637 // Difference between black and white colors

	r1, g1, b1, _ := color1.RGBA()
	r2, g2, b2, _ := color2.RGBA()

	return math.Sqrt(distance(float64(r2), float64(r1))+
		distance(float64(g2), float64(g1))+
		distance(float64(b2), float64(b1))) / maxDiff
}

// CmpRGBComponents returns RGB components difference of two colors.
func CmpRGBComponents(color1 color.Color, color2 color.Color) float64 {
	const maxDiff = 765.0 // Difference between black and white colors

	r1, g1, b1, _ := color1.RGBA()
	r2, g2, b2, _ := color2.RGBA()

	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8

	return float64((max(r1, r2)-min(r1, r2))+
		(max(g1, g2)-min(g1, g2))+
		(max(b1, b2)-min(b1, b2))) / maxDiff
}

// CmpCIE76 returns difference of two colors defined in CIE76 standart.
//
// https://en.wikipedia.org/wiki/Color_difference#CIE76
func CmpCIE76(color1 color.Color, color2 color.Color) float64 {
	const maxDiff = 149.95514755 // Difference between blue and white colors

	cl1, ca1, cb1 := colorToLAB(color1)
	cl2, ca2, cb2 := colorToLAB(color2)

	return math.Sqrt(distance(cl2, cl1)+distance(ca2, ca1)+distance(cb2, cb1)) / maxDiff
}

func distance(x, y float64) float64 {
	return (x - y) * (x - y)
}

// colorToLAB returns LAB representation of any color (without aplha)
// https://en.wikipedia.org/wiki/Lab_color_space
func colorToLAB(color color.Color) (l, a, b float64) {
	return xyztoLAB(colorToXYZ(color))
}

// xyztoLAB converts CIE XYZ color space to CIE LAB color space
// https://en.wikipedia.org/wiki/Lab_color_space#CIELAB-CIEXYZ_conversions
func xyztoLAB(x, y, z float64) (l, a, b float64) {
	refX, refY, refZ := 95.047, 100.000, 108.883 // Daylight, sRGB, Adobe-RGB, Observer D65, 2°

	varX := x / refX
	varY := y / refY
	varZ := z / refZ

	if varX > 0.008856 {
		varX = math.Pow(varX, 1.0/3.0)
	} else {
		varX = (7.787 * varX) + (16.0 / 116.0)
	}
	if varY > 0.008856 {
		varY = math.Pow(varY, 1.0/3.0)
	} else {
		varY = (7.787 * varY) + (16.0 / 116.0)
	}
	if varZ > 0.008856 {
		varZ = math.Pow(varZ, 1.0/3.0)
	} else {
		varZ = (7.787 * varZ) + (16.0 / 116.0)
	}

	l = (116 * varY) - 16
	a = 500 * (varX - varY)
	b = 200 * (varY - varZ)

	return l, a, b
}

// colorToXYZ returns CIE XYZ representation of color.
// https://en.wikipedia.org/wiki/Color_model#CIE_XYZ_color_space
func colorToXYZ(color color.Color) (x, y, z float64) {
	r, g, b, _ := color.RGBA()
	varR := float64(r>>8) / 255
	varG := float64(g>>8) / 255
	varB := float64(b>>8) / 255

	if varR > 0.04045 {
		varR = math.Pow((varR+0.055)/1.055, 2.4)
	} else {
		varR /= 12.92
	}

	if varG > 0.04045 {
		varG = math.Pow((varG+0.055)/1.055, 2.4)
	} else {
		varG /= 12.92
	}

	if varB > 0.04045 {
		varB = math.Pow((varB+0.055)/1.055, 2.4)
	} else {
		varB /= 12.92
	}

	x = varR*41.24 + varG*35.76 + varB*18.05
	y = varR*21.26 + varG*71.52 + varB*7.22
	z = varR*1.93 + varG*11.92 + varB*95.05

	return x, y, z
}
