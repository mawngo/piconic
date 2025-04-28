package cmd

import (
	"fmt"
	"github.com/mawngo/piconic/internal/icon"
	"github.com/mawngo/piconic/internal/scan"
	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"runtime"
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

	f := icon.Flags{
		Size: 200,
		OutputFlags: icon.OutputFlags{
			Output:     ".",
			Padding:    10,
			Round:      0,
			Background: icon.AutoColor + "," + icon.BackgroundDefaultColor,
			Trim:       icon.TransparentColor,
		},
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

			// If the first argument is a placeholder size, then switch to generating placeholder.
			if _, _, ok := icon.ParsePlaceholderSize(args[0]); ok {
				placeholders := make(map[string][]icon.PlaceholderFlags)
				sizes := make([]icon.PlaceholderFlags, 0, len(args))
				for _, arg := range args {
					if w, h, ok := icon.ParsePlaceholderSize(arg); ok {
						sizes = append(sizes, icon.PlaceholderFlags{
							OutputFlags: f.OutputFlags,
							W:           w,
							H:           h,
						})
						continue
					}
					arg = strings.TrimSpace(arg)
					placeholders[arg] = append(placeholders[arg], sizes...)
					sizes = make([]icon.PlaceholderFlags, 0, len(args))
				}
				placeholders[""] = append(placeholders[""], sizes...)

				for placeholder, sizes := range placeholders {
					for _, size := range sizes {
						processPlaceholder(size, placeholder, con)
					}
				}
			} else {
				// Generate icon mode.
				for _, arg := range args {
					for img := range scan.Img(arg) {
						processIcon(f, img, con)
					}
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
	command.Flags().StringVarP(&f.Background, "bg", "b", f.Background, "Background color ['transparent', 'auto', 'auto,fallback', hex, material, svg 1.1]")
	command.Flags().StringVar(&f.Trim, "trim", f.Trim, "List of color to trim when process image")
	command.Flags().UintVarP(&f.Padding, "padding", "p", f.Padding, "Padding of the icon image (by % of the size)")
	command.Flags().UintVarP(&f.Round, "round", "r", f.Round, "Round the output image (by % of the size)")
	command.Flags().UintVar(&f.SrcRound, "src-round", f.SrcRound, "Round the source image (by % of the size)")
	command.Flags().IntVar(&f.PadX, "padx", f.PadX, "Additional padding to the x axis (by % of the size)")
	command.Flags().IntVar(&f.PadY, "pady", f.PadY, "Additional padding to the y axis (by % of the size)")
	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	command.Flags().SortFlags = false
	return &CLI{&command}
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func processIcon(f icon.Flags, img scan.DecodedImage, con chan struct{}) {
	con <- struct{}{}
	go func() {
		defer func() {
			<-con
		}()
		icon.WriteIcon(f, img)
	}()
}

func processPlaceholder(f icon.PlaceholderFlags, text string, con chan struct{}) {
	con <- struct{}{}
	go func() {
		defer func() {
			<-con
		}()
		icon.WritePlaceholder(f, text)
	}()
}
