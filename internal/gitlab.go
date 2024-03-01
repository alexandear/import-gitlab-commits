package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"
)

const (
	maxCommits = 1000
)

type GitLab struct {
	logger *log.Logger

	gitlabClient *gitlab.Client
}

func NewGitLab(logger *log.Logger, gitlabClient *gitlab.Client) *GitLab {
	return &GitLab{
		logger:       logger,
		gitlabClient: gitlabClient,
	}
}

func (s *GitLab) CurrentUser(ctx context.Context) (*User, error) {
	user, _, err := s.gitlabClient.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}

	return &User{
		Name:      user.Name,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: *user.CreatedAt,
	}, nil
}

func (s *GitLab) FetchProjectPage(ctx context.Context, page int, user *User, idAfter int,
) (_ []int, nextPage int, _ error) {
	const perPage = 100

	projects := make([]int, 0, perPage)

	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
		OrderBy:    gitlab.String("id"),
		Sort:       gitlab.String("asc"),
		Simple:     gitlab.Bool(true),
		Membership: gitlab.Bool(true),
		IDAfter:    gitlab.Int(idAfter),
	}

	projs, resp, err := s.gitlabClient.Projects.ListProjects(opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}

	for _, proj := range projs {
		if !s.HasUserContributions(ctx, user, proj.ID) {
			continue
		}

		s.logger.Printf("Fetching project: %d", proj.ID)

		projects = append(projects, proj.ID)
	}

	if resp.CurrentPage >= resp.TotalPages {
		return projects, 0, nil
	}

	return projects, resp.NextPage, nil
}

func (s *GitLab) HasUserContributions(ctx context.Context, user *User, projectID int) bool {
	const perPage = 50

	opt := &gitlab.ListContributorsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    1,
		},
	}

	for {
		contrs, resp, err := s.gitlabClient.Repositories.Contributors(projectID, opt, gitlab.WithContext(ctx))
		if err != nil {
			s.logger.Printf("get contributors for project %d: %v", projectID, err)

			return false
		}

		for _, c := range contrs {
			if strings.EqualFold(c.Email, user.Email) {
				return true
			}
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
	}

	return false
}

func (s *GitLab) FetchCommits(ctx context.Context, user *User, projectID int, since time.Time,
) ([]*Commit, error) {
	commits := make([]*Commit, 0, maxCommits)

	page := 1
	for page > 0 {
		cms, nextPage, err := s.fetchCommitPage(ctx, user, page, 100, since, projectID)
		if err != nil {
			return nil, fmt.Errorf("fetch one commit page: %w", err)
		}

		commits = append(commits, cms...)
		page = nextPage
	}

	// Reverse slice.
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}

func (s *GitLab) fetchCommitPage(
	ctx context.Context, user *User, page, perPage int, since time.Time, projectID int,
) (commits []*Commit, nextPage int, err error) {
	commits = make([]*Commit, 0, perPage)

	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
		All: gitlab.Bool(true),
	}

	if !since.IsZero() {
		opt.Since = gitlab.Time(since)
	}

	comms, resp, err := s.gitlabClient.Commits.ListCommits(projectID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, 0, fmt.Errorf("get commits for project %d: %w", projectID, err)
	}

	for _, comm := range comms {
		if !strings.EqualFold(comm.AuthorEmail, user.Email) || !strings.EqualFold(comm.CommitterEmail, user.Email) {
			continue
		}

		s.logger.Printf("fetching commit: %s %s", comm.ShortID, comm.CommittedDate)

		commits = append(commits, NewCommit(*comm.CommittedDate, projectID, comm.ID))
	}

	// For performance reasons, if a query returns more than 10,000 records, GitLab
	// doesn't return TotalPages.
	if resp.TotalPages == 0 {
		return commits, resp.NextPage, nil
	}

	if resp.CurrentPage >= resp.TotalPages {
		return commits, 0, nil
	}

	return commits, resp.NextPage, nil
}
