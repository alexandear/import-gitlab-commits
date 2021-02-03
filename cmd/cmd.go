package cmd

import (
	"log"

	"github.com/alexandear/fake-private-contributions/internal/app"
	"github.com/alexandear/fake-private-contributions/internal/fetcher"
)

func Execute(logger *log.Logger) error {
	f := fetcher.New()

	a := app.New(logger, f)

	return a.Run()
}
