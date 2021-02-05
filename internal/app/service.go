package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
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

const (
	getCurrentUserTimeout = 2 * time.Second
)

type Fetcher interface {
	FetchProjects(ctx context.Context) <-chan *pkg.Project
	FetchCommits(ctx context.Context, project *pkg.Project) chan *pkg.Commit
}

type Storage interface {
	AddCommit(projectName string, commit *pkg.Commit) error
	NextCommit(projectName string) chan *pkg.Commit
	AddProject(project *pkg.Project) error
	NextProject() chan *pkg.Project
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

	ctx, cancel := context.WithTimeout(context.Background(), getCurrentUserTimeout)
	currentUser, err := newCurrentUser(ctx, gitlabClient)

	cancel()

	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	const createIfNotExist = os.FileMode(0o600)

	dbName := dbName(gitlabBaseURL, currentUser)

	db, err := bbolt.Open(dbName, createIfNotExist, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbName, err)
	}

	f := fetcher.New(logger, gitlabClient, currentUser)
	a := &App{
		logger:  logger,
		fetcher: f,
		storage: bboltS.New(db),
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Println("init repo in memory")

	a.logger.Println("saving projects to storage")

	projects := a.fetcher.FetchProjects(ctx)

	const workers = 5

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			a.storeProjects(ctx, projects)

			wg.Done()
		}()
	}

	wg.Wait()

	fs := memfs.New()

	r, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	project := &pkg.Project{
		ID:   277,
		Name: "277",
	}

	for commit := range a.fetcher.FetchCommits(ctx, project) {
		if errAdd := a.storage.AddCommit(project.Name, commit); errAdd != nil {
			a.logger.Printf("failed to add commit %v: %v", commit, errAdd)
		}
	}

	committer := &object.Signature{
		Name:  "Oleksandr Redko",
		Email: "oleksandr.red+github@gmail.com",
	}

	i := 0

	for commit := range a.storage.NextCommit(project.Name) {
		committer.When = commit.When

		h, cerr := w.Commit(commit.Message, &git.CommitOptions{
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

func (a *App) storeProjects(ctx context.Context, projects <-chan *pkg.Project) {
	for project := range projects {
		select {
		default:
			a.logger.Printf("adding project: %v", project)

			if errAdd := a.storage.AddProject(project); errAdd != nil {
				a.logger.Printf("failed to add project %v: %v", project, errAdd)
			}
		case <-ctx.Done():
			a.logger.Printf("save projects done")

			return
		}
	}
}

func newCurrentUser(ctx context.Context, gitlabClient *gitlab.Client) (*pkg.User, error) {
	u, _, err := gitlabClient.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return &pkg.User{
		Name:     u.Name,
		Email:    u.Email,
		Username: u.Username,
	}, nil
}

func dbName(baseURL *url.URL, user *pkg.User) string {
	host := baseURL.Host

	hostPort := strings.Split(host, ":")
	if len(hostPort) > 2 {
		host = hostPort[0]
	}

	return host + "." + user.Username + ".db"
}
