package main

import (
	"os"

	"github.com/AndreeJait/kyan/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}