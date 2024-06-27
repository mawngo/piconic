package cmd

import (
	"fmt"
	"github.com/mawngo/piconic/internal/scan"
	"github.com/phsym/console-slog"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
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
		Run: func(cmd *cobra.Command, args []string) {
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
	command.Flags().IntVarP(&f.Padding, "padding", "p", f.Padding, "Padding of the image (by % of the size)")
	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	return &CLI{&command}
}

type flags struct {
	Size    uint
	Output  string
	Padding int
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func process(flags flags, img scan.DecodedImage) {

}
