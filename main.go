package main

import (
	"log"
	"os"

	"github.com/alexandear/import-gitlab-commits/cmd"
)

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile|log.Ltime)

	if err := cmd.Execute(logger); err != nil {
		logger.Fatal("Error:", err)
	}
}
