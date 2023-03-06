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

	pkg "github.com/alexandear/import-gitlab-commits/internal"
	"github.com/alexandear/import-gitlab-commits/internal/gitlab"
)

const (
	getCurrentUserTimeout = 2 * time.Second

	maxProjects = 1000
)

type Gitlab interface {
	CurrentUser(ctx context.Context) (*pkg.User, error)
	FetchProjectPage(ctx context.Context, page int, user *pkg.User, idAfter int,
	) (projects []*pkg.Project, nextPage int, err error)
	FetchCommits(ctx context.Context, user *pkg.User, projectID int, since time.Time) ([]*pkg.Commit, error)
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
		return nil, fmt.Errorf("create GitLab client: %w", err)
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
		return fmt.Errorf("get current user: %w", err)
	}

	a.logger.Printf("Found current user %q\n", currentUser.Name)

	repoPath := "./" + repoName(a.gitlabBaseURL, currentUser)

	r, err := git.PlainInit(repoPath, false)
	if err == nil {
		a.logger.Printf("Init repository %q\n", repoPath)
	} else if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		a.logger.Printf("Repository %q already exists, opening it\n", repoPath)

		r, err = git.PlainOpen(repoPath)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
	} else {
		return fmt.Errorf("init: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	var lastCommitDate time.Time

	switch head, errHead := r.Head(); {
	case errHead == nil:
		headCommit, errCommit := r.CommitObject(head.Hash())
		if errCommit != nil {
			return fmt.Errorf("get head commit: %w", errCommit)
		}

		projectID, _, errParse := pkg.ParseCommitMessage(headCommit.Message)
		if errParse != nil {
			return fmt.Errorf("parse commit message: %w", errParse)
		}

		lastCommitDate = headCommit.Committer.When

		a.logger.Printf("Found last project id %d and last commit date %v\n", projectID, lastCommitDate)
	case errors.Is(errHead, plumbing.ErrReferenceNotFound):
	default:
		return fmt.Errorf("get head: %w", errHead)
	}

	projectCommitCounter := make(map[int]int, maxProjects)

	projectID := 0
	page := 1

	for page > 0 {
		projects, nextPage, errFetch := a.gitlab.FetchProjectPage(ctx, page, currentUser, projectID)
		if errFetch != nil {
			return fmt.Errorf("fetch projects: %w", errFetch)
		}

		for _, project := range projects {
			commits, errCommit := a.doCommitsForProject(ctx, w, currentUser, project, lastCommitDate)
			if errCommit != nil {
				return fmt.Errorf("do commits: %w", errCommit)
			}

			projectCommitCounter[project.ID] = commits
		}

		page = nextPage
	}

	for project, commit := range projectCommitCounter {
		a.logger.Printf("project %d: commits %d", project, commit)
	}

	return nil
}

func (a *App) doCommitsForProject(ctx context.Context, w *git.Worktree, currentUser *pkg.User, project *pkg.Project,
	lastCommitDate time.Time) (int, error) {
	commits, err := a.gitlab.FetchCommits(ctx, currentUser, project.ID, lastCommitDate)
	if err != nil {
		return 0, fmt.Errorf("fetch commits: %w", err)
	}

	a.logger.Printf("fetched %d commits for project %d", len(commits), project.ID)

	var commitCounter int

	committer := &object.Signature{
		Name:  a.committer.Name,
		Email: a.committer.Email,
	}

	for _, commit := range commits {
		committer.When = commit.CommittedAt

		if _, errCommit := w.Commit(commit.Message, &git.CommitOptions{
			Author:    committer,
			Committer: committer,
		}); errCommit != nil {
			return commitCounter, fmt.Errorf("commit: %w", errCommit)
		}

		commitCounter++
	}

	return commitCounter, nil
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
