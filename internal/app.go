// Package app provides the core functionality for the import-gitlab-commits application,
// including initializing the GitLab client, fetching user information,
// and importing commits into a local git repository.
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
	gogitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	getCurrentUserTimeout = 2 * time.Second

	maxProjects = 1000
)

type App struct {
	logger *log.Logger

	gitlabBaseURL *url.URL
	gitlab        *GitLab

	committerName  string
	committerEmail string
}

type User struct {
	Name      string
	Emails    []string
	Username  string
	CreatedAt time.Time
}

func New(logger *log.Logger, gitlabToken string, gitlabBaseURL *url.URL, committerName, committerEmail string,
) (*App, error) {
	gitlabClient, err := gogitlab.NewClient(gitlabToken, gogitlab.WithBaseURL(gitlabBaseURL.String()))
	if err != nil {
		return nil, fmt.Errorf("create GitLab client: %w", err)
	}

	f := NewGitLab(logger, gitlabClient)

	return &App{
		logger:         logger,
		gitlab:         f,
		gitlabBaseURL:  gitlabBaseURL,
		committerName:  committerName,
		committerEmail: committerEmail,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ctxCurrent, cancel := context.WithTimeout(ctx, getCurrentUserTimeout)
	defer cancel()

	currentUser, err := a.gitlab.CurrentUser(ctxCurrent)
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	a.logger.Printf("Found current user %q", currentUser.Name)

	repoPath := "./" + repoName(a.gitlabBaseURL, currentUser)

	repo, err := a.createOrOpenRepo(repoPath)
	if err != nil {
		return fmt.Errorf("create or open repo: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	lastCommitDate := a.lastCommitDate(repo)

	projectCommitCounter := make(map[int]int, maxProjects)

	idAfter := 0
	page := 1

	for page > 0 {
		projects, nextPage, errFetch := a.gitlab.FetchProjectPage(ctx, page, currentUser, idAfter)
		if errFetch != nil {
			return fmt.Errorf("fetch projects: %w", errFetch)
		}

		for _, project := range projects {
			commits, errCommit := a.doCommitsForProject(ctx, worktree, currentUser, project, lastCommitDate)
			if errCommit != nil {
				return fmt.Errorf("do commits: %w", errCommit)
			}

			projectCommitCounter[project] = commits

			// Update idAfter to the highest project ID seen so far for cursor-based pagination.
			if project > idAfter {
				idAfter = project
			}
		}

		page = nextPage
	}

	for project, commit := range projectCommitCounter {
		a.logger.Printf("project %d: commits %d", project, commit)
	}

	return nil
}

func (a *App) createOrOpenRepo(repoPath string) (*git.Repository, error) {
	repo, err := git.PlainInit(repoPath, false)
	if err == nil {
		a.logger.Printf("Init repository %q", repoPath)

		return repo, nil
	}

	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		a.logger.Printf("Repository %q already exists, opening it", repoPath)

		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("open: %w", err)
		}

		return repo, nil
	}

	return nil, fmt.Errorf("init: %w", err)
}

func (a *App) lastCommitDate(repo *git.Repository) time.Time {
	head, err := repo.Head()
	if err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			a.logger.Printf("Failed to get repo head: %v", err)
		}

		return time.Time{}
	}

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		a.logger.Printf("Failed to get head commit: %v", err)

		return time.Time{}
	}

	projectID, _, err := ParseCommitMessage(headCommit.Message)
	if err != nil {
		a.logger.Printf("Failed to parse commit message: %v", err)

		return time.Time{}
	}

	lastCommitDate := headCommit.Committer.When

	a.logger.Printf("Found last project id %d and last commit date %v", projectID, lastCommitDate)

	return lastCommitDate
}

func (a *App) doCommitsForProject(
	ctx context.Context, worktree *git.Worktree, currentUser *User, projectID int, lastCommitDate time.Time,
) (int, error) {
	commits, err := a.gitlab.FetchCommits(ctx, currentUser, projectID, lastCommitDate)
	if err != nil {
		return 0, fmt.Errorf("fetch commits: %w", err)
	}

	a.logger.Printf("Fetched %d commits for project %d", len(commits), projectID)

	var commitCounter int

	committer := &object.Signature{
		Name:  a.committerName,
		Email: a.committerEmail,
	}

	for _, commit := range commits {
		committer.When = commit.CommittedAt

		if _, errCommit := worktree.Commit(commit.Message, &git.CommitOptions{
			Author:            committer,
			Committer:         committer,
			AllowEmptyCommits: true,
		}); errCommit != nil {
			return commitCounter, fmt.Errorf("commit: %w", errCommit)
		}

		commitCounter++
	}

	return commitCounter, nil
}

// repoName generates unique repo name for the user.
func repoName(baseURL *url.URL, user *User) string {
	host := baseURL.Host

	const hostPortLen = 2

	hostPort := strings.Split(host, ":")
	if len(hostPort) > hostPortLen {
		host = hostPort[0]
	}

	return "repo." + host + "." + user.Username
}
