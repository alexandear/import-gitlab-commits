package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/alexandear/fake-private-contributions/internal/app"
)

const (
	runTimeout = 2 * time.Minute
)

var ErrInvalidArgument = errors.New("invalid argument")

func NewErrInvalidArgument(arg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, arg)
}

func Execute(logger *log.Logger) error {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return NewErrInvalidArgument("empty GITLAB_TOKEN")
	}

	baseURL, err := url.Parse(os.Getenv("GITLAB_BASE_URL"))
	if err != nil {
		return fmt.Errorf("wrong GITLAB_BASE_URL value: %w", err)
	}

	committerName := os.Getenv("COMMITTER_NAME")
	if committerName == "" {
		return NewErrInvalidArgument("empty COMMITTER_NAME")
	}

	committerEmail := os.Getenv("COMMITTER_EMAIL")
	if committerEmail == "" {
		return NewErrInvalidArgument("empty COMMITTER_EMAIL")
	}

	a, err := app.New(logger, token, baseURL, committerName, committerEmail)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	return a.Run(ctx)
}
