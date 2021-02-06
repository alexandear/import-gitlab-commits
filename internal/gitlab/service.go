package gitlab

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/import-gitlab-contribs/internal"
)

const (
	maxCommits = 1000
)

type Service struct {
	logger *log.Logger

	gitlabClient *gitlab.Client
}

func New(logger *log.Logger, gitlabClient *gitlab.Client) *Service {
	return &Service{
		logger:       logger,
		gitlabClient: gitlabClient,
	}
}

func (s *Service) CurrentUser(ctx context.Context) (*pkg.User, error) {
	u, _, err := s.gitlabClient.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}

	return &pkg.User{
		Name:      u.Name,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: *u.CreatedAt,
	}, nil
}

func (s *Service) FetchProjectPage(ctx context.Context, page int, user *pkg.User, idAfter int,
) (projects []*pkg.Project, nextPage int, err error) {
	const perPage = 100

	projects = make([]*pkg.Project, 0, perPage)

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

	for _, p := range projs {
		if !s.hasUserContributions(ctx, user, p.ID) {
			continue
		}

		s.logger.Printf("fetching project: %d", p.ID)

		projects = append(projects, &pkg.Project{ID: p.ID})
	}

	if resp.CurrentPage >= resp.TotalPages {
		return projects, 0, nil
	}

	return projects, resp.NextPage, nil
}

func (s *Service) hasUserContributions(ctx context.Context, user *pkg.User, projectID int) bool {
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

func (s *Service) FetchCommits(ctx context.Context, user *pkg.User, projectID int, since time.Time,
) ([]*pkg.Commit, error) {
	commits := make([]*pkg.Commit, 0, maxCommits)

	if since.IsZero() {
		since = user.CreatedAt
	}

	page := 1
	for page > 0 {
		cms, nextPage, err := s.fetchCommitPage(ctx, user, page, 100, since, projectID)
		if err != nil {
			return nil, fmt.Errorf("fetch one commit page: %w", err)
		}

		commits = append(commits, cms...)
		page = nextPage
	}

	// reverse slice
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}

func (s *Service) fetchCommitPage(ctx context.Context, user *pkg.User, page, perPage int, since time.Time,
	projectID int) (commits []*pkg.Commit, nextPage int, err error) {
	commits = make([]*pkg.Commit, 0, perPage)

	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
		Since: gitlab.Time(since),
		All:   gitlab.Bool(true),
	}

	comms, resp, err := s.gitlabClient.Commits.ListCommits(projectID, opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, 0, fmt.Errorf("get commits for project %d: %w", projectID, err)
	}

	for _, c := range comms {
		if !strings.EqualFold(c.AuthorEmail, user.Email) || !strings.EqualFold(c.CommitterEmail, user.Email) {
			continue
		}

		s.logger.Printf("fetching commit: %s %s", c.ShortID, c.CommittedDate)

		commits = append(commits, pkg.NewCommit(*c.CommittedDate, projectID, c.ID))
	}

	if resp.CurrentPage >= resp.TotalPages {
		return commits, 0, nil
	}

	return commits, resp.NextPage, nil
}
