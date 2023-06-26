package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	app "github.com/alexandear/import-gitlab-commits/internal"
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

	application, err := app.New(logger, token, baseURL, committerName, committerEmail)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	if err := application.Run(ctx); err != nil {
		return fmt.Errorf("app run: %w", err)
	}

	return nil
}

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile|log.Ltime)

	if err := Execute(logger); err != nil {
		logger.Fatalln("Error:", err)
	}
}
