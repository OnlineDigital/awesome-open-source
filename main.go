package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andreitelteu/inertia-go-solid-template/app/server"
	"github.com/andreitelteu/inertia-go-solid-template/internal/artisan"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "artisan" {
		args = args[1:]
		if len(args) == 0 {
			args = []string{"help"}
		}
	}
	var err error
	if len(args) == 0 {
		root, cwdErr := os.Getwd()
		if cwdErr != nil {
			err = cwdErr
		} else {
			err = artisan.LoadEnv(filepath.Join(root, ".env"))
		}
		if err == nil {
			err = server.RunFromEnvironment()
		}
	} else {
		err = artisan.Run(args, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
