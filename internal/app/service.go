package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/xanzy/go-gitlab"
	"go.etcd.io/bbolt"

	pkg "github.com/alexandear/fake-private-contributions/internal"
	bboltS "github.com/alexandear/fake-private-contributions/internal/bbolt"
	"github.com/alexandear/fake-private-contributions/internal/fetcher"
)

type Fetcher interface {
	FirstProject(ctx context.Context) (*pkg.Project, error)
	FetchCommits(ctx context.Context, project *pkg.Project)
}

type Storage interface {
	NextCommit(projectName string) chan *pkg.Commit
}

type App struct {
	logger  *log.Logger
	fetcher Fetcher
	storage Storage
}

func New(logger *log.Logger, gitlabToken string, gitlabBaseURL *url.URL) (*App, error) {
	gitlabClient, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabBaseURL.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	host := gitlabBaseURL.Host

	hostPort := strings.Split(host, ":")
	if len(hostPort) > 2 {
		host = hostPort[0]
	}

	dbName := host + ".db"

	const createIfNotExist = os.FileMode(0o600)

	db, err := bbolt.Open(dbName, createIfNotExist, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbName, err)
	}

	storage := bboltS.New(db)
	f := fetcher.New(logger, gitlabClient, storage)
	a := &App{
		logger:  logger,
		fetcher: f,
		storage: storage,
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

	project, err := a.fetcher.FirstProject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get first project: %w", err)
	}

	a.logger.Printf("got project: %v", project)

	a.fetcher.FetchCommits(ctx, project)

	committer := &object.Signature{
		Name:  "Oleksandr Redko",
		Email: "oleksandr.red+github@gmail.com",
	}

	i := 0

	for commit := range a.storage.NextCommit(project.Name) {
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
