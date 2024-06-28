package cmd

import (
	"fmt"
	"github.com/mawngo/piconic/internal/scan"
	"github.com/mawngo/piconic/internal/utils"
	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
	matcolornames "golang.org/x/exp/shiny/materialdesign/colornames"
	_ "golang.org/x/image/bmp" // Enable support for bmp.
	"golang.org/x/image/colornames"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // Enable support for webp.
	"image"
	"image/color"
	_ "image/jpeg" // Enable support for jpeg.
	"image/png"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const backgroundDefaultColor = "#f8fafc"
const transparentColor = "transparent"

func Init() *slog.LevelVar {
	level := &slog.LevelVar{}
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{
			Level:      level,
			TimeFormat: time.Kitchen,
		}))
	slog.SetDefault(logger)
	cobra.EnableCommandSorting = false
	return level
}

type CLI struct {
	command *cobra.Command
}

// NewCLI create new CLI instance and setup application config.
func NewCLI() *CLI {
	level := Init()

	f := flags{
		Size:       200,
		Output:     ".",
		Padding:    10,
		Round:      0,
		Background: backgroundDefaultColor,
		Trim:       transparentColor,
	}

	command := cobra.Command{
		Use:   "piconic [files...]",
		Short: "Generate icon from images",
		Args:  cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			debug, err := cmd.PersistentFlags().GetBool("debug")
			if err != nil {
				return err
			}
			if debug {
				level.Set(slog.LevelDebug)
			}
			return nil
		},
		Run: func(_ *cobra.Command, args []string) {
			now := time.Now()
			if _, err := os.Stat(f.Output); err != nil {
				err := os.Mkdir(f.Output, os.ModePerm)
				if err != nil {
					slog.Info("Error creating output directory", slog.Any("dir", f.Output))
					return
				}
			}

			concurrency := runtime.NumCPU()
			con := make(chan struct{}, concurrency)
			for _, arg := range args {
				for img := range scan.Img(arg) {
					process(f, img, con)
				}
			}

			for range concurrency {
				con <- struct{}{}
			}
			slog.Info("Processing completed", slog.Duration("took", time.Since(now)))
		},
	}

	command.Flags().StringVarP(&f.Output, "out", "o", f.Output, "Output directory name")
	command.Flags().BoolVarP(&f.Overwrite, "overwrite", "w", f.Overwrite, "Overwrite output if exists")
	command.Flags().UintVarP(&f.Size, "size", "s", f.Size, "Size of the output image")
	command.Flags().StringVarP(&f.Background, "bg", "b", f.Background, "Background color [transparent, hex, material color name like Yellow500 or svg 1.1 color name like yellow]")
	command.Flags().StringVar(&f.Trim, "trim", f.Trim, "List of color to trim when process image")
	command.Flags().UintVarP(&f.Padding, "padding", "p", f.Padding, "Padding of the icon image (by % of the size)")
	command.Flags().UintVarP(&f.Round, "round", "r", f.Round, "Round the output image (by % of the size)")
	command.Flags().UintVar(&f.SrcRound, "src-round", f.SrcRound, "Round the source image (by % of the size)")
	command.Flags().IntVar(&f.PadX, "padx", f.PadX, "Additional padding to the x axis (by % of the size)")
	command.Flags().IntVar(&f.PadY, "pady", f.PadY, "Additional padding to the y axis (by % of the size)")
	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	return &CLI{&command}
}

type flags struct {
	Size       uint
	Output     string
	Padding    uint
	Round      uint
	SrcRound   uint
	Overwrite  bool
	Background string
	Trim       string
	PadX       int
	PadY       int
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func process(f flags, img scan.DecodedImage, con chan struct{}) {
	con <- struct{}{}
	go func() {
		defer func() {
			<-con
		}()
		generateIcon(f, img)
	}()
}

func generateIcon(f flags, img scan.DecodedImage) {
	outName := fmt.Sprintf("%s.%dpc%d.png", strings.TrimSuffix(filepath.Base(img.Path), filepath.Ext(img.Path)), f.Size, f.Padding)
	outfile := filepath.Join(f.Output, outName)
	if _, err := os.Stat(outfile); err == nil {
		slog.Info("File existed",
			slog.Any("path", outfile),
			slog.Bool("override", f.Overwrite),
		)
		if !f.Overwrite {
			return
		}
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
	if f.Round > 0 {
		err := utils.RoundImage(bgImg, float64(f.Round)/100)
		if err != nil {
			slog.Warn("Output format does not support rounding", slog.String("out", outfile))
		}
	}

	o, err := os.Create(outfile)
	if err == nil {
		err = png.Encode(o, bgImg)
	}
	if err != nil {
		slog.Error("Error writing image", slog.String("out", outfile), slog.Any("err", err))
		return
	}
}

func resize(f flags, img scan.DecodedImage, rect image.Rectangle) scan.DecodedImage {
	imgSize := img.Width
	if img.Width < img.Height {
		imgSize = img.Height
	}
	targetSize := float64(f.Size) - float64(f.Size)*(float64(f.Padding)/100)*2
	ratio := targetSize / float64(imgSize)
	slog.Debug("Resize ratio", slog.String("path", img.Path), slog.Float64("ratio", ratio))

	width := int(math.RoundToEven(float64(img.Width) * ratio))
	height := int(math.RoundToEven(float64(img.Height) * ratio))

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

func calculateTargetRect(f flags, img scan.DecodedImage) (color.Color, image.Rectangle) {
	if f.Trim == "" {
		return calculateColor(f.Background, backgroundDefaultColor), img.Bounds()
	}
	colors := strings.Split(f.Trim, ",")
	trim := make([]color.Color, 0, len(colors))
	for _, s := range colors {
		trim = append(trim, calculateColor(strings.TrimSpace(s), transparentColor))
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
	return calculateColor(f.Background, backgroundDefaultColor), image.Rectangle{Min: minPt, Max: maxPt}
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

func calculateColor(bg string, fallback string) color.Color {
	if bg == transparentColor {
		return utils.EmptyColor
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
		if fallback == transparentColor {
			return utils.EmptyColor
		}
		c, err = utils.ParseHexColor(fallback)
		if err != nil {
			panic(err)
		}
	}
	return c
}
