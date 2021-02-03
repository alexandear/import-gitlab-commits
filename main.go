package main

import (
	"log"
	"os"

	"github.com/alexandear/fake-private-contributions/cmd"
)

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile|log.Ltime)

	if err := cmd.Execute(logger); err != nil {
		logger.Fatal(err)
	}
}
