package cmd

import (
	"fmt"
	"github.com/mawngo/piconic/internal/scan"
	"github.com/mawngo/piconic/internal/utils"
	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
		Size:    200,
		Output:  ".",
		Padding: 10,
		Round:   0,
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

			for _, arg := range args {
				for img := range scan.Img(arg) {
					process(f, img)
				}
			}

			slog.Info("Processing completed", slog.Duration("took", time.Since(now)))
		},
	}

	command.Flags().UintVarP(&f.Size, "size", "s", f.Size, "Size of the output image")
	command.Flags().StringVarP(&f.Output, "out", "o", f.Output, "Output directory name")
	command.Flags().UintVarP(&f.Padding, "padding", "p", f.Padding, "Padding of the icon image (by % of the size)")
	command.Flags().BoolVarP(&f.Overwrite, "overwrite", "w", f.Overwrite, "Overwrite output if exists")
	command.Flags().UintVarP(&f.Round, "round", "r", f.Round, "Round the output image (by % of the size)")
	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	return &CLI{&command}
}

type flags struct {
	Size      uint
	Output    string
	Padding   uint
	Round     uint
	Overwrite bool
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func process(f flags, img scan.DecodedImage) {
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

	img = resize(f, img)

	bgColor := calculateBGColor(f, img)
	bgImg := image.NewRGBA(image.Rect(0, 0, int(f.Size), int(f.Size)))
	draw.Draw(bgImg, bgImg.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	offset := image.Pt((int(f.Size)-img.Width)/2, (int(f.Size)-img.Height)/2)
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

func resize(f flags, img scan.DecodedImage) scan.DecodedImage {
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
		calculateTrimmedRect(img),
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

func calculateBGColor(_ flags, _ scan.DecodedImage) color.RGBA {
	// TODO: calculate based on image content.
	c, err := utils.ParseHexColor("#f8fafc")
	if err != nil {
		panic(err)
	}
	return c
}

func calculateTrimmedRect(img image.Image) image.Rectangle {
	// TODO: only include image content region.
	return img.Bounds()
}
