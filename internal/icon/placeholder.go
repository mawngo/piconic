package icon

import (
	"fmt"
	"github.com/goki/freetype"
	"github.com/goki/freetype/truetype"
	"github.com/mawngo/piconic/internal/utils"
	matcolornames "golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/image/colornames"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"image"
	"image/color"
	"log/slog"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

var tff *truetype.Font

const noneText = "<none>"

var filenameNormalizer = strings.NewReplacer(
	" ", "-",
	"?", "",
	"\\", "",
	":", "",
	"/", "",
	"<", "",
	">", "",
	"%", "",
	"*", "",
	"\"", "",
)

var (
	placeholderSizeRegex      = regexp.MustCompile(`^[1-9][0-9]*x[1-9][0-9]*$`)
	placeholderTextColorRegex = regexp.MustCompile(`(<.+>)$`)
)

func InitFont(ttf []byte) {
	var err error
	if tff, err = freetype.ParseFont(ttf); err != nil {
		panic(err)
	}
}

func ParsePlaceholderSize(size string) (w int, h int, valid bool) {
	if !placeholderSizeRegex.MatchString(size) {
		return 0, 0, false
	}

	wh := strings.Split(size, "x")
	if len(wh) != 2 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(wh[0])
	if err != nil {
		panic(err)
	}
	h, err = strconv.Atoi(wh[1])
	if err != nil {
		panic(err)
	}
	return w, h, true
}

type PlaceholderFlags struct {
	OutputFlags
	W int
	H int
}

func WritePlaceholder(f PlaceholderFlags, placeholder string) {
	dimStr := fmt.Sprintf("%dx%d", f.W, f.H)
	if placeholder == "" {
		placeholder = dimStr
	}
	if placeholder == noneText {
		placeholder = ""
	}
	slog.Info("Processing",
		slog.String("text", placeholder),
		slog.String("dimension", dimStr),
		slog.String("bg", f.Background),
	)

	outName := fmt.Sprintf("%spc%d.png", dimStr, f.Padding)
	if placeholder != "" && placeholder != dimStr {
		outName = filenameNormalizer.Replace(placeholder) + "." + outName
	}
	outfile, ok := canWriteOutImage(f.OutputFlags, outName)
	if !ok {
		return
	}

	bgColor := calculatePlaceholderBackgroundColor(f.Background)
	placeholder, textColor := calculatePlaceholderTextColor(placeholder, bgColor, dimStr)

	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	if placeholder != "" {
		fontsize, xOffset, yOffset, err := calculateFontSize(f, placeholder, img)
		xcenter := (float64(f.W) / 2.0) - xOffset + (float64(f.W) * float64(f.PadX) / 100)
		ycenter := (float64(f.H) / 2.0) - yOffset + (float64(f.H) * float64(f.PadY) / 100)
		if err != nil {
			slog.Error("Error calculating tff size", slog.String("dimension", dimStr), slog.Any("err", err))
			return
		}

		c := freetype.NewContext()
		c.SetFont(tff)
		c.SetSrc(&image.Uniform{C: color.Transparent})
		c.SetDst(img)
		c.SetClip(img.Bounds())
		c.SetFontSize(fontsize)
		c.SetSrc(image.NewUniform(&image.Uniform{C: textColor}))
		_, err = c.DrawString(placeholder, freetype.Pt(int(xcenter), int(ycenter)))
		if err != nil {
			slog.Error("Error drawing text", slog.String("dimension", dimStr), slog.Any("err", err))
			return
		}
	}
	writeOutImage(f.OutputFlags, outfile, img)
}

func calculateFontSize(f PlaceholderFlags, text string, img draw.Image) (float64, float64, float64, error) {
	maxW := int(math.RoundToEven(float64(f.W) * (1 - float64(f.Padding)*2/100)))
	maxH := int(math.RoundToEven(float64(f.H) * (1 - float64(f.Padding)*2/100)))
	fontsize := float64(maxH)

	// Find the biggest matching tff size for the requested height.
	height := calculateFontHeight(fontsize)
	iter := float64(1)
	for int(height) > maxH {
		fontsize -= 2
		oldHeight := height
		height = calculateFontHeight(fontsize)
		if iter < 1 {
			continue
		}
		reductionRate := math.Ceil(oldHeight-height) / iter
		iter = math.Floor((math.Ceil(height) - math.Floor(float64(maxH))) / reductionRate)
		if iter > 1 {
			fontsize -= iter * 2
		}
	}

	face := truetype.NewFace(tff, &truetype.Options{
		Size: fontsize,
	})
	drawer := font.Drawer{
		Face: face,
		Dst:  img,
		Src:  image.NewUniform(&image.Uniform{C: color.Transparent}),
		Dot:  freetype.Pt(0, 0),
	}
	// Find the biggest matching tff size for the requested width.
	actWidth := float64(drawer.MeasureString(text)) / 64
	iter = float64(1)
	for int(actWidth) > maxW {
		if err := face.Close(); err != nil {
			panic(err)
		}
		fontsize -= 2
		face = truetype.NewFace(tff, &truetype.Options{
			Size: fontsize,
		})
		drawer.Face = face
		oldWidth := actWidth
		actWidth = float64(drawer.MeasureString(text)) / 64
		if iter < 1 {
			continue
		}
		reductionRate := math.Ceil(oldWidth-actWidth) / iter
		iter = math.Floor((math.Ceil(actWidth) - math.Floor(float64(maxW))) / reductionRate)
		if iter > 1 {
			fontsize -= iter * 2
		}
	}

	if err := face.Close(); err != nil {
		panic(err)
	}
	face = truetype.NewFace(tff, &truetype.Options{
		Size: fontsize,
	})
	defer face.Close()
	// Calculate offset based on bounds.
	bound, adv := drawer.BoundString(text)
	yBaselineToCenterOffset := float64(bound.Max.Y+bound.Min.Y) / 2
	return fontsize, float64(adv) / 2 / 64, yBaselineToCenterOffset / 64, nil
}

func calculateFontHeight(fontsize float64) float64 {
	face := truetype.NewFace(tff, &truetype.Options{
		Size: fontsize,
	})
	defer face.Close()
	return float64(face.Metrics().Height) / 64
}

func calculatePlaceholderTextColor(text string, bg color.Color, dimStr string) (string, color.Color) {
	if cname := placeholderTextColorRegex.FindString(text); cname != "" {
		c, ok := calculatePlaceholderColor(cname[1:len(cname)-1], TransparentColor)
		if ok {
			text := strings.TrimSpace(strings.TrimSuffix(text, cname))
			if text == "" {
				text = dimStr
			}
			return text, c
		}
		slog.Warn("Unsupported text color, fallback to auto contrast",
			slog.String("color", cname))
	}
	// Transparent.
	if bg == color.Transparent {
		return text, color.Black
	}
	return text, contrastColor(bg)
}

func calculatePlaceholderBackgroundColor(bg string) color.Color {
	c, ok := calculatePlaceholderColor(bg, TransparentColor)
	if ok {
		return c
	}
	slog.Warn("Unsupported color, fallback to default",
		slog.String("color", bg),
		slog.String("default", BackgroundDefaultColor))
	c, _ = calculatePlaceholderColor(BackgroundDefaultColor, TransparentColor)
	return c
}

func calculatePlaceholderColor(cname string, fallback string) (color.Color, bool) {
	if strings.HasPrefix(cname, AutoColor) {
		i := rand.Intn(len(matcolornames.Names))
		return matcolornames.Map[matcolornames.Names[i]], true
	}

	if cname == TransparentColor {
		return color.Transparent, true
	}
	if !strings.HasPrefix(cname, "#") {
		// SVG color names.
		c, ok := colornames.Map[cname]
		if ok {
			return c, true
		}
		// Material design color names.
		c, ok = matcolornames.Map[cname]
		if ok {
			return c, true
		}
	}
	c, err := utils.ParseHexColor(cname)
	if err == nil {
		return c, true
	}

	if fallback == TransparentColor || fallback == "" {
		return color.Transparent, false
	}
	return calculatePlaceholderColor(fallback, TransparentColor)
}

// Chooses a contrasting color (black or white) based on luminance
func contrastColor(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r)/65535, float64(g)/65535, float64(b)/65535
	adjust := func(val float64) float64 {
		if val <= 0.03928 {
			return val / 12.92
		}
		return math.Pow((val+0.055)/1.055, 2.4)
	}
	rLinear := adjust(rf)
	gLinear := adjust(gf)
	bLinear := adjust(bf)
	relativeLuminance := 0.2126*rLinear + 0.7152*gLinear + 0.0722*bLinear
	if relativeLuminance > 0.5 {
		return color.RGBA{
			R: 18, G: 18, B: 18, A: 255,
		}
	}
	return color.RGBA{
		R: 250, G: 250, B: 250, A: 255,
	}
}
