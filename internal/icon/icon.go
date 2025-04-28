package icon

import (
	"fmt"
	"github.com/mawngo/piconic/internal/colorcmp"
	"github.com/mawngo/piconic/internal/scan"
	"github.com/mawngo/piconic/internal/utils"
	matcolornames "golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/image/colornames"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	BackgroundDefaultColor = "#f1f5f9"
	AutoColor              = "auto"
	TransparentColor       = "transparent"
)

type OutputFlags struct {
	Output     string
	Padding    uint
	Round      uint
	Overwrite  bool
	Background string
	Trim       string
	PadX       int
	PadY       int
}

type Flags struct {
	OutputFlags
	Size     uint
	SrcRound uint
}

func WriteIcon(f Flags, img scan.DecodedImage) {
	slog.Info("Processing",
		slog.String("img", filepath.Base(img.Path)),
		slog.String("dimension", fmt.Sprintf("%dx%d", img.Width, img.Height)),
		slog.String("bg", f.Background),
		slog.Any("size", f.Size),
	)

	outName := fmt.Sprintf("%s.%dpc%d.png", strings.TrimSuffix(filepath.Base(img.Path), filepath.Ext(img.Path)), f.Size, f.Padding)
	outfile, ok := canWriteOutImage(f.OutputFlags, outName)
	if !ok {
		return
	}

	bgColor, rect := calculateTargetRect(f, img)
	img = resize(f, img, rect)
	if f.SrcRound > 0 {
		err := utils.RoundImage(img.Image, float64(f.SrcRound)/100)
		if err != nil {
			slog.Warn("Source format does not support rounding", slog.String("out", outfile))
		}
	}

	bgImg := image.NewRGBA(image.Rect(0, 0, int(f.Size), int(f.Size)))
	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	offset := image.Pt((int(f.Size)-img.Width)/2, (int(f.Size)-img.Height)/2)
	offset = offset.Add(image.Pt(int(math.RoundToEven((float64(f.PadX)/100)*float64(f.Size))), int(math.RoundToEven((float64(f.PadY)/100)*float64(f.Size)))))
	slog.Debug("Padding", slog.Int("x", offset.X), slog.Int("y", offset.Y))
	draw.Draw(bgImg, bgImg.Bounds().Add(offset), img.Image, image.Point{}, draw.Over)
	writeOutImage(f.OutputFlags, outName, bgImg)
}

func resize(f Flags, img scan.DecodedImage, rect image.Rectangle) scan.DecodedImage {
	imgSize := rect.Dx()
	if imgSize < rect.Dy() {
		imgSize = rect.Dy()
	}
	targetSize := float64(f.Size) - float64(f.Size)*(float64(f.Padding)/100)*2
	ratio := targetSize / float64(imgSize)
	slog.Debug("Resize ratio", slog.String("path", img.Path), slog.Float64("ratio", ratio))

	width := int(math.RoundToEven(float64(rect.Dx()) * ratio))
	height := int(math.RoundToEven(float64(rect.Dy()) * ratio))

	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resized, resized.Bounds(),
		img.Image,
		rect,
		draw.Src,
		nil)
	slog.Debug("Resized image", slog.String("path", img.Path), slog.String("dimension", fmt.Sprintf("%dx%d", width, height)))
	return scan.DecodedImage{
		Image:  resized,
		Path:   img.Path,
		Width:  width,
		Height: height,
	}
}

func calculateTargetRect(f Flags, img scan.DecodedImage) (color.Color, image.Rectangle) {
	if f.Trim == "" {
		return calculateColor(img, f.Background, BackgroundDefaultColor), img.Bounds()
	}
	colors := strings.Split(f.Trim, ",")
	trim := make([]color.Color, 0, len(colors))
	for _, s := range colors {
		trim = append(trim, calculateColor(img, strings.TrimSpace(s), TransparentColor))
	}
	trim = utils.Uniq(trim)

	// Trim colors by finding a new bound.
	minPt := img.Bounds().Min
	maxPt := img.Bounds().Max

MINX:
	for x := range img.Bounds().Max.X {
		for y := range img.Bounds().Max.Y {
			if isContainAnyColors(trim, img, x, y) {
				continue
			}
			minPt.X = x
			break MINX
		}
	}

MINY:
	for y := range img.Bounds().Max.Y {
		for x := range img.Bounds().Max.X {
			if isContainAnyColors(trim, img, x, y) {
				continue
			}
			minPt.Y = y
			break MINY
		}
	}

MAXX:
	for x := img.Bounds().Max.X - 1; x >= 0; x-- {
		for y := img.Bounds().Max.Y - 1; y >= 0; y-- {
			if isContainAnyColors(trim, img, x, y) {
				continue
			}
			maxPt.X = x
			break MAXX
		}
	}
MAXY:
	for y := img.Bounds().Max.Y - 1; y >= 0; y-- {
		for x := img.Bounds().Max.X - 1; x >= 0; x-- {
			if isContainAnyColors(trim, img, x, y) {
				continue
			}
			maxPt.Y = y
			break MAXY
		}
	}
	return calculateColor(img, f.Background, BackgroundDefaultColor), image.Rectangle{Min: minPt, Max: maxPt}
}

func isContainAnyColors(colors []color.Color, img image.Image, x int, y int) bool {
	r, g, b, a := img.At(x, y).RGBA()
	for _, rgba := range colors {
		cr, cg, cb, ca := rgba.RGBA()
		if cr == r && cg == g && cb == b && ca == a {
			return true
		}
	}
	return false
}

func calculateColor(img scan.DecodedImage, bg string, fallback string) color.Color {
	if strings.HasPrefix(bg, AutoColor) {
		c, ok := calculateAutoBackgroundColor(img)
		if ok {
			return c
		}
		// Does not specify auto fallback color.
		if !strings.Contains(bg, ",") {
			var err error
			c, err = utils.ParseHexColor(fallback)
			if err != nil {
				panic(err)
			}
			return c
		}
		// Fallback color specified, parse fallback color instead.
		bg = strings.TrimSpace(strings.Split(bg, ",")[1])
	}

	if bg == TransparentColor {
		return color.Transparent
	}

	if !strings.HasPrefix(bg, "#") {
		// SVG color names.
		c, ok := colornames.Map[bg]
		if ok {
			return c
		}
		// Material design color names.
		c, ok = matcolornames.Map[bg]
		if ok {
			return c
		}
		slog.Warn("Unsupported color, fallback to default hex",
			slog.String("color", bg),
			slog.String("default", fallback),
		)
		bg = fallback
	}

	c, err := utils.ParseHexColor(bg)
	if err != nil {
		slog.Warn("Invalid hex color, fallback to default",
			slog.String("hex", bg),
			slog.String("default", fallback))
		if fallback == TransparentColor {
			return color.Transparent
		}
		c, err = utils.ParseHexColor(fallback)
		if err != nil {
			panic(err)
		}
	}
	return c
}

func calculateAutoBackgroundColor(img scan.DecodedImage) (color.Color, bool) {
	c := img.At(0, 0)
	diffCnt := 0
	if img.Bounds().Max.X <= 8 || img.Bounds().Max.Y <= 8 {
		// Require need at least 8x8 image to auto calculate color
		return c, false
	}

	// The bg color will be set to the 2px border color if all pixels of the border have the same color.
	// Checking the left and right border.
	for y := 2; y < img.Bounds().Max.Y-2; y++ {
		for x := range []int{0, 1, img.Bounds().Max.X - 2, img.Bounds().Max.X - 1} {
			border := img.At(x, y)
			if colorcmp.CmpCIE76(c, border) > 0.02 {
				diffCnt++
			}
		}
	}

	// Checking the top and bottom border.
	for x := 0; x < img.Bounds().Max.X; x++ {
		for y := range []int{0, 1, img.Bounds().Max.Y - 2, img.Bounds().Max.Y - 1} {
			border := img.At(x, y)
			if colorcmp.CmpCIE76(c, border) > 0.02 {
				diffCnt++
			}
		}
	}
	diffRatio := float64(diffCnt) / float64(img.Bounds().Max.X*4+img.Bounds().Max.Y*4)
	// We can ignore if the ratio of different pixel is small enough.
	if diffRatio > 0.01 {
		return c, false
	}
	if _, _, _, a := c.RGBA(); a == 0 {
		// Ignore transparent image.
		return c, false
	}
	return c, true
}

func writeOutImage(f OutputFlags, outName string, img image.Image) {
	if f.Round > 0 {
		err := utils.RoundImage(img, float64(f.Round)/100)
		if err != nil {
			slog.Warn("Output format does not support rounding", slog.String("out", filepath.Join(f.Output, outName)))
		}
	}

	outfile, ok := canWriteOutImage(f, outName)
	if !ok {
		return
	}
	o, err := os.Create(outfile)
	if err == nil {
		err = png.Encode(o, img)
	}
	if err != nil {
		slog.Error("Error writing image", slog.String("out", outfile), slog.Any("err", err))
		return
	}
}

func canWriteOutImage(f OutputFlags, outName string) (string, bool) {
	outfile := filepath.Join(f.Output, outName)
	if _, err := os.Stat(outfile); err == nil {
		if !f.Overwrite {
			slog.Warn("File existed",
				slog.Any("path", outfile),
				slog.Bool("override", f.Overwrite),
			)
			return outfile, false
		}
	}
	return outfile, true
}
