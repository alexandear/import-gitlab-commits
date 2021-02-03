package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
	"github.com/alexandear/fake-private-contributions/internal/fetcher"
)

type Fetcher interface {
	FirstProject(ctx context.Context) (*pkg.Project, error)
	Next(project *pkg.Project) <-chan *pkg.Commit
}

type App struct {
	logger  *log.Logger
	fetcher Fetcher
}

func New(logger *log.Logger, gitlabToken, gitlabBaseURL string) (*App, error) {
	gitlabClient, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabBaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	f := fetcher.New(logger, gitlabClient)
	a := &App{
		logger:  logger,
		fetcher: f,
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Println("init repo in memory")

	fs := memfs.New()

	r, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	a.logger.Println("make initial commit")

	committer := &object.Signature{
		Name:  "Oleksandr Redko",
		Email: "oleksandr.red+github@gmail.com",
	}

	project, err := a.fetcher.FirstProject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get first project: %w", err)
	}

	a.logger.Printf("got project: %v", project)

	i := 0

	for commit := range a.fetcher.Next(project) {
		committer.When = commit.When

		h, cerr := w.Commit("commit "+strconv.Itoa(i), &git.CommitOptions{
			Author:    committer,
			Committer: committer,
		})
		if cerr != nil {
			return fmt.Errorf("failed to commit: %w", cerr)
		}

		log.Println("committed", i, h)

		i++
	}

	a.logger.Println("log")

	ci, err := r.Log(&git.LogOptions{})
	if err != nil {
		return fmt.Errorf("failed to get log: %w", err)
	}

	for {
		c, err := ci.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("failed during iterate: %w", err)
		}

		a.logger.Printf("commit %s\nAuthor: %s\nAuthor date: %s\nCommitter: %s\nCommit date: %s\n   %s\n\n",
			c.Hash, c.Author.Name, c.Author.When, c.Committer.Name, c.Committer.When, c.Message)
	}

	return nil
}
