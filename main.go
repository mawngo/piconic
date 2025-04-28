package main

import (
	_ "embed"
	"github.com/mawngo/piconic/cmd"
	"github.com/mawngo/piconic/internal/icon"
)

//go:embed Roboto-SemiBold.ttf
var ttf []byte

func main() {
	icon.InitFont(ttf)
	cli := cmd.NewCLI()
	cli.Execute()
}
