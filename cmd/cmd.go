package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/alexandear/import-gitlab-commits/app"
)

const (
	runTimeout = 10 * time.Minute
)

func Execute(logger *log.Logger) error {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return errors.New(`empty GITLAB_TOKEN, example "yourgitlabtoken"`)
	}

	baseURL, err := url.Parse(os.Getenv("GITLAB_BASE_URL"))
	if err != nil {
		return errors.New(`wrong GITLAB_BASE_URL, example "https://gitlab.com"`)
	}

	committerName := os.Getenv("COMMITTER_NAME")
	if committerName == "" {
		return errors.New(`empty COMMITTER_NAME, example "John Doe"`)
	}

	committerEmail := os.Getenv("COMMITTER_EMAIL")
	if committerEmail == "" {
		return errors.New(`empty COMMITTER_EMAIL, example "john.doe@example.com"`)
	}

	app, err := app.New(logger, token, baseURL, committerName, committerEmail)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("app run: %w", err)
	}

	return nil
}
