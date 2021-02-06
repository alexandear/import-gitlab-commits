package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	goGitlab "github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
	"github.com/alexandear/fake-private-contributions/internal/gitlab"
)

const (
	getCurrentUserTimeout = 2 * time.Second
)

type Gitlab interface {
	CurrentUser(ctx context.Context) (*pkg.User, error)
	FetchProjects(ctx context.Context, user *pkg.User, idAfter int) <-chan *pkg.Project
	FetchCommits(ctx context.Context, user *pkg.User, projectID int, since time.Time) chan *pkg.Commit
}

type App struct {
	logger *log.Logger

	gitlabBaseURL *url.URL
	gitlab        Gitlab

	committer *pkg.Committer
}

func New(logger *log.Logger, gitlabToken string, gitlabBaseURL *url.URL, committerName, committerEmail string,
) (*App, error) {
	gitlabClient, err := goGitlab.NewClient(gitlabToken, goGitlab.WithBaseURL(gitlabBaseURL.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	f := gitlab.New(logger, gitlabClient)
	a := &App{
		logger:        logger,
		gitlab:        f,
		gitlabBaseURL: gitlabBaseURL,
		committer: &pkg.Committer{
			Name:  committerName,
			Email: committerEmail,
		},
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	ctxCurrent, cancel := context.WithTimeout(ctx, getCurrentUserTimeout)
	defer cancel()

	currentUser, err := a.gitlab.CurrentUser(ctxCurrent)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	repoPath := "./" + repoName(a.gitlabBaseURL, currentUser)

	r, err := git.PlainInit(repoPath, false)
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		r, err = git.PlainOpen(repoPath)
		if err != nil {
			return fmt.Errorf("failed to open: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	var (
		lastProjectID  int
		lastCommitDate time.Time
	)

	switch head, err := r.Head(); {
	case err == nil:
		headCommit, errHead := r.CommitObject(head.Hash())
		if errHead != nil {
			return fmt.Errorf("failed to get head commit: %w", errHead)
		}

		id, _, errParse := pkg.ParseCommitMessage(headCommit.Message)
		if errParse != nil {
			return fmt.Errorf("failed to parse commit message: %w", errParse)
		}

		lastProjectID = id
		lastCommitDate = headCommit.Committer.When
	case errors.Is(err, plumbing.ErrReferenceNotFound):
	default:
		return fmt.Errorf("failed to get head: %w", err)
	}

	committer := &object.Signature{
		Name:  a.committer.Name,
		Email: a.committer.Email,
	}

	projectCounter := 0
	commitCounter := 0

	for project := range a.gitlab.FetchProjects(ctx, currentUser, lastProjectID) {
		commits := make([]*pkg.Commit, 0, 1000)
		for commit := range a.gitlab.FetchCommits(ctx, currentUser, project.ID, lastCommitDate) {
			commits = append(commits, commit)
		}

		a.logger.Printf("fetched %d commits for project %d", len(commits), project.ID)

		for i := len(commits) - 1; i >= 0; i-- {
			commit := commits[i]
			committer.When = commit.CommittedAt

			_, cerr := w.Commit(commit.Message, &git.CommitOptions{
				Author:    committer,
				Committer: committer,
			})
			if cerr != nil {
				return fmt.Errorf("failed to commit: %w", cerr)
			}

			commitCounter++
		}

		projectCounter++
	}

	a.logger.Printf("projects %d, commits %d", projectCounter, commitCounter)

	return nil
}

// repoName generates unique repo name for the user.
func repoName(baseURL *url.URL, user *pkg.User) string {
	host := baseURL.Host

	const hostPortLen = 2

	hostPort := strings.Split(host, ":")
	if len(hostPort) > hostPortLen {
		host = hostPort[0]
	}

	return "repo." + host + "." + user.Username
}
