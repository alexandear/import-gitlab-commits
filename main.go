package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	app "github.com/alexandear/import-gitlab-commits/internal"
)

const (
	runTimeout = 10 * time.Minute

	helpText = `Import GitLab Commits

Imports commits from a private GitLab repository to a separate repository.

Usage:
  import-gitlab-commits [flags]

Flags:
  -h, --help    Show help message

Environment Variables:
  GITLAB_BASE_URL     GitLab instance URL (e.g., https://gitlab.com)
  GITLAB_TOKEN        GitLab personal access token (scopes: read_api, read_user, read_repository)
  COMMITTER_NAME      Your full name (e.g., John Doe)
  COMMITTER_EMAIL     Your email (e.g., john.doe@example.com)
`
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
	help := flag.Bool("help", false, "Show help message")
	flag.BoolVar(help, "h", false, "Show help message (shorthand)")
	flag.Parse()

	if *help {
		_, _ = os.Stdout.WriteString(helpText)
		os.Exit(0)
	}

	logger := log.New(os.Stdout, "", log.Lshortfile|log.Ltime)

	if err := Execute(logger); err != nil {
		logger.Fatalln("Error:", err)
	}
}
