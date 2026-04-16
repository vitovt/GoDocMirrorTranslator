package main

import (
	"context"
	"fmt"
	"os"

	"godocmirrortranslator/internal/app"
	"godocmirrortranslator/internal/cli"
	"godocmirrortranslator/internal/gui"
)

func main() {
	if len(os.Args) == 1 {
		if err := gui.Run(context.Background(), app.New(cli.Version), cli.Version, ""); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
