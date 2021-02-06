package gitlab

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
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
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return &pkg.User{
		Name:      u.Name,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: *u.CreatedAt,
	}, nil
}

func (s *Service) FetchProjects(ctx context.Context, user *pkg.User, idAfter int) <-chan *pkg.Project {
	const chanSize = 100

	projects := make(chan *pkg.Project, chanSize)

	go func() {
		defer close(projects)

		page := 1
		for page > 0 {
			select {
			case <-ctx.Done():
				s.logger.Printf("fetching projects canceled")

				return
			default:
			}

			nextPage, err := s.fetchProjectPage(ctx, user, page, chanSize, idAfter, projects)
			if err != nil {
				s.logger.Printf("failed to fetch one project page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return projects
}

func (s *Service) fetchProjectPage(ctx context.Context, user *pkg.User, page, perPage, idAfter int,
	projects chan<- *pkg.Project,
) (nextPage int, err error) {
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
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	for _, p := range projs {
		select {
		default:
			if !s.hasUserContributions(ctx, user, p.ID) {
				continue
			}

			s.logger.Printf("fetching project: %d", p.ID)

			projects <- &pkg.Project{ID: p.ID}
		case <-ctx.Done():
			return 0, nil
		}
	}

	if resp.CurrentPage >= resp.TotalPages {
		return 0, nil
	}

	return resp.NextPage, nil
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
			s.logger.Printf("failed to get contributors for project %d: %v", projectID, err)

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

func (s *Service) FetchCommits(ctx context.Context, user *pkg.User, projectID int, since time.Time) chan *pkg.Commit {
	const chanSize = 100

	commits := make(chan *pkg.Commit, chanSize)

	if since.IsZero() {
		since = user.CreatedAt
	}

	go func() {
		defer close(commits)

		page := 1
		for page > 0 {
			select {
			case <-ctx.Done():
				s.logger.Printf("fetching commits canceled")

				return
			default:
			}

			nextPage, err := s.fetchCommitPage(ctx, user, page, chanSize, since, projectID, commits)
			if err != nil {
				s.logger.Printf("failed to fetch one commit page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return commits
}

func (s *Service) fetchCommitPage(ctx context.Context, user *pkg.User, page, perPage int, since time.Time,
	projectID int, commits chan<- *pkg.Commit) (nextPage int, err error) {
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
		return 0, fmt.Errorf("failed to get commits for project %d: %w", projectID, err)
	}

	for _, c := range comms {
		select {
		default:
			if !strings.EqualFold(c.AuthorEmail, user.Email) || !strings.EqualFold(c.CommitterEmail, user.Email) {
				continue
			}

			s.logger.Printf("fetching commit: %s %s", c.ShortID, c.CommittedDate)

			commits <- pkg.NewCommit(*c.CommittedDate, projectID, c.ID)
		case <-ctx.Done():
			return 0, nil
		}
	}

	if resp.CurrentPage >= resp.TotalPages {
		return 0, nil
	}

	return resp.NextPage, nil
}
