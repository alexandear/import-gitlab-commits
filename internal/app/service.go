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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	FetchProjects(ctx context.Context, idAfter int) <-chan *pkg.Project
	FetchCommits(ctx context.Context, projectID int, since time.Time) chan *pkg.Commit
}

type Storage interface {
	AddCommit(projectID int, commit *pkg.Commit) error
	LastCommit(projectID int) (*pkg.Commit, error)
	NextCommit(projectID int) chan *pkg.Commit
	AddProject(project *pkg.Project) error
	NextProject() chan *pkg.Project
	LastProject() (*pkg.Project, error)
}

type App struct {
	logger  *log.Logger
	fetcher Fetcher
	storage Storage

	gitlabBaseURL *url.URL
	currentUser   *pkg.User
	committer     *pkg.Committer
}

func New(logger *log.Logger, gitlabToken string, gitlabBaseURL *url.URL, committerName, committerEmail string,
) (*App, error) {
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

	dbName := objectName(gitlabBaseURL, currentUser) + ".db"

	db, err := bbolt.Open(dbName, createIfNotExist, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbName, err)
	}

	f := fetcher.New(logger, gitlabClient, currentUser)
	a := &App{
		logger:        logger,
		fetcher:       f,
		storage:       bboltS.New(db),
		gitlabBaseURL: gitlabBaseURL,
		currentUser:   currentUser,
		committer: &pkg.Committer{
			Name:  committerName,
			Email: committerEmail,
		},
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.fetchAndStoreProjects(ctx); err != nil {
		return fmt.Errorf("failed to get fetch and store projects: %w", err)
	}

	a.fetchAndStoreCommits(ctx)

	repoPath := "./" + objectName(a.gitlabBaseURL, a.currentUser)

	r, err := git.PlainInit(repoPath, false)
	if err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	committer := &object.Signature{
		Name:  a.committer.Name,
		Email: a.committer.Email,
	}

	commitCounter := 0

	for project := range a.storage.NextProject() {
		for commit := range a.storage.NextCommit(project.ID) {
			committer.When = commit.CommittedAt

			h, cerr := w.Commit(commit.Message, &git.CommitOptions{
				Author:    committer,
				Committer: committer,
			})
			if cerr != nil {
				return fmt.Errorf("failed to commit: %w", cerr)
			}

			log.Printf("committed %d %s", commitCounter, h)

			commitCounter++
		}
	}

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

	a.logger.Printf("commits total: %d", commitCounter)

	return nil
}

func (a *App) fetchAndStoreProjects(ctx context.Context) error {
	lastProject, err := a.storage.LastProject()
	if err != nil {
		return fmt.Errorf("failed to get last project: %w", err)
	}

	a.logger.Printf("last saved project is %d", lastProject.ID)

	projects := a.fetcher.FetchProjects(ctx, lastProject.ID)

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

func (a *App) fetchAndStoreCommits(ctx context.Context) {
	projects := a.storage.NextProject()

	const workers = 5

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			a.storeCommits(ctx, projects)

			wg.Done()
		}()
	}

	wg.Wait()
}

func (a *App) storeCommits(ctx context.Context, projects chan *pkg.Project) {
	project := <-projects

	a.logger.Printf("storing commits for project %d", project.ID)

	var since time.Time

	last, errLast := a.storage.LastCommit(project.ID)
	if errLast == nil {
		a.logger.Printf("last saved commit for project %d is %s", project.ID, last.CommittedAt)

		since = last.CommittedAt
	} else {
		a.logger.Printf("failed to get last commit: %v", errLast)
	}

	for commit := range a.fetcher.FetchCommits(ctx, project.ID, since) {
		a.logger.Printf("adding commit %v for project %d", commit, project.ID)

		if errAdd := a.storage.AddCommit(project.ID, commit); errAdd != nil {
			a.logger.Printf("failed to add commit %v: %v", commit, errAdd)
		}
	}
}

func newCurrentUser(ctx context.Context, gitlabClient *gitlab.Client) (*pkg.User, error) {
	u, _, err := gitlabClient.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return &pkg.User{
		Name:      u.Name,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: *u.CreatedAt,
	}, nil
}

// objectName used to generate unique db and repo names for the user.
func objectName(baseURL *url.URL, user *pkg.User) string {
	host := baseURL.Host

	hostPort := strings.Split(host, ":")
	if len(hostPort) > 2 {
		host = hostPort[0]
	}

	return host + "." + user.Username
}
