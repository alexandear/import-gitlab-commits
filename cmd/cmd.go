package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	pkg "github.com/alexandear/fake-private-contributions/internal"
	"github.com/alexandear/fake-private-contributions/internal/app"
)

const (
	runTimeout = 2 * time.Minute
)

func Execute(logger *log.Logger) error {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return pkg.NewErrInvalidArgument(`empty GITLAB_TOKEN, example "yourgitlabtoken"`)
	}

	baseURL, err := url.Parse(os.Getenv("GITLAB_BASE_URL"))
	if err != nil {
		return pkg.NewErrInvalidArgument(`wrong GITLAB_BASE_URL, example "https://gitlab.example.com/"`)
	}

	committerName := os.Getenv("COMMITTER_NAME")
	if committerName == "" {
		return pkg.NewErrInvalidArgument(`empty COMMITTER_NAME, example "John Doe"`)
	}

	committerEmail := os.Getenv("COMMITTER_EMAIL")
	if committerEmail == "" {
		return pkg.NewErrInvalidArgument(`empty COMMITTER_EMAIL, example "john.doe@example.com"`)
	}

	a, err := app.New(logger, token, baseURL, committerName, committerEmail)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	return a.Run(ctx)
}
