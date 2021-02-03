package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	baseURL := os.Getenv("GITLAB_BASE_URL")
	if baseURL == "" {
		return NewErrInvalidArgument("empty GITLAB_BASE_URL")
	}

	a, err := app.New(logger, token, baseURL)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	return a.Run(ctx)
}
