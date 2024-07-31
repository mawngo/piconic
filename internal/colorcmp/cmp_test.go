package colorcmp

import (
	"image/color"
	"math"
	"testing"
)

// epsilon is a maximum permissible error.
const epsilon = 0.00000001

func TestColorToLAB(t *testing.T) {
	tests := []struct {
		color            color.Color
		expL, expA, expB float64
		gotL, gotA, gotB float64
	}{
		{color: color.RGBA{A: 255}, expL: 0.0, expA: 0.0, expB: 0.0},
		{color: color.RGBA{B: 255, A: 255}, expL: 32.30258667, expA: 79.19666179, expB: -107.86368104},
		{color: color.RGBA{G: 255, A: 255}, expL: 87.73703347, expA: -86.18463650, expB: 83.18116475},
		{color: color.RGBA{R: 255, A: 255}, expL: 53.23288179, expA: 80.10930953, expB: 67.22006831},
		{color: color.RGBA{R: 255, G: 255, B: 255, A: 255}, expL: 100.00000000, expA: 0.00526050, expB: -0.01040818},
	}

	for _, test := range tests {
		test.gotL, test.gotA, test.gotB = colorToLAB(test.color)
		if math.Abs(test.gotL-test.expL) > epsilon || math.Abs(test.gotA-test.expA) > epsilon ||
			math.Abs(test.gotB-test.expB) > epsilon {
			t.Errorf("%v: expected {%.8f, %.8f, %.8f}, got {%.8f, %.8f, %.8f}",
				test.color, test.expL, test.expA, test.expB, test.gotL, test.gotA, test.gotB)
		}
	}
}

func TestLinearComparators(t *testing.T) {
	comparators := []comparator{CmpEuclidean, CmpRGBComponents}

	tests := []struct {
		color1 color.Color
		color2 color.Color
		exp    float64
		got    float64
	}{
		{color1: color.RGBA{A: 255}, color2: color.RGBA{A: 255}, exp: 0.00},                                                 // same black colors
		{color1: color.RGBA{R: 255, G: 255, B: 255, A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 0.00}, // same white colors
		{color1: color.RGBA{A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 1.00},                         // different (black and white) colors
		{color1: color.RGBA{R: 255, G: 255, B: 255, A: 255}, color2: color.RGBA{A: 255}, exp: 1.00},                         // different (white and black) colors
		{color1: color.RGBA{R: 255, G: 255, B: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 0.00},         // must ignore alpha channel
	}

	for _, comparator := range comparators {
		for _, test := range tests {
			test.got = comparator(test.color1, test.color2)
			if math.Abs(test.got-test.exp) > epsilon {
				t.Errorf("%v %v: expected %.8f, got %.8f",
					test.color1, test.color2, test.exp, test.got)
			}
		}
	}
}

func TestCmpCIE76(t *testing.T) {
	type test struct {
		color1 color.Color
		color2 color.Color
		exp    float64
		got    float64
	}

	tests := []test{
		{color1: color.RGBA{A: 255}, color2: color.RGBA{A: 255}, exp: 0.00000000 / 149.95514755},                                                 // same black colors
		{color1: color.RGBA{R: 255, G: 255, B: 255, A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 0.00000000 / 149.95514755}, // same white colors
		{color1: color.RGBA{A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 100.00000068 / 149.95514755},                       // different (black and white) colors
		{color1: color.RGBA{R: 255, G: 255, B: 255, A: 255}, color2: color.RGBA{A: 255}, exp: 100.00000068 / 149.95514755},                       // different (white and black) colors
		{color1: color.RGBA{R: 255, A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 114.55897602 / 149.95514755},               // different (red and white) colors
		{color1: color.RGBA{G: 255, A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 120.41559907 / 149.95514755},               // different (green and white) colors
		{color1: color.RGBA{B: 255, A: 255}, color2: color.RGBA{R: 255, G: 255, B: 255, A: 255}, exp: 1},                                         // different (blue and white) colors
	}

	for _, test := range tests {
		test.got = CmpCIE76(test.color1, test.color2)
		if math.Abs(test.got-test.exp) > epsilon {
			t.Errorf("%v %v: expected %.8f, got %.8f",
				test.color1, test.color2, test.exp, test.got)
		}
	}
}
