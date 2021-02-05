package fetcher

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

type Service struct {
	logger       *log.Logger
	gitlabClient *gitlab.Client
	user         *pkg.User
}

func New(logger *log.Logger, gitlabClient *gitlab.Client, user *pkg.User) *Service {
	return &Service{
		logger:       logger,
		gitlabClient: gitlabClient,
		user:         user,
	}
}

func (s *Service) FetchCommits(ctx context.Context, projectID int, since time.Time) chan *pkg.Commit {
	const chanSize = 100

	commits := make(chan *pkg.Commit, chanSize)

	if since.IsZero() {
		since = s.user.CreatedAt
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

			nextPage, err := s.fetchCommitPage(ctx, page, chanSize, since, projectID, commits)
			if err != nil {
				s.logger.Printf("failed to fetch one commit page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return commits
}

func (s *Service) fetchCommitPage(ctx context.Context, page, perPage int, since time.Time, projectID int,
	commits chan<- *pkg.Commit) (nextPage int, err error) {
	ctxOne, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
		Since: gitlab.Time(since),
		All:   gitlab.Bool(true),
	}

	comms, resp, err := s.gitlabClient.Commits.ListCommits(projectID, opt, gitlab.WithContext(ctxOne))
	if err != nil {
		return 0, fmt.Errorf("failed to get commits for project %d: %w", projectID, err)
	}

	for _, c := range comms {
		select {
		default:
			s.logger.Printf("fetching commit: %s", c.ID)

			if !strings.EqualFold(c.AuthorEmail, s.user.Email) || !strings.EqualFold(c.CommitterEmail, s.user.Email) {
				s.logger.Printf("commit %s isn't current user's %s contribution", c.ID, s.user.Email)

				continue
			}

			commits <- &pkg.Commit{
				CommittedAt: *c.CommittedDate,
				Message:     fmt.Sprintf("Project: %d commit: %s", projectID, c.ID),
			}
		case <-ctx.Done():
			return 0, nil
		}
	}

	if resp.CurrentPage >= resp.TotalPages {
		return 0, nil
	}

	return resp.NextPage, nil
}

func (s *Service) FetchProjects(ctx context.Context, idAfter int) <-chan *pkg.Project {
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

			nextPage, err := s.fetchProjectPage(ctx, page, chanSize, idAfter, projects)
			if err != nil {
				s.logger.Printf("failed to fetch one project page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return projects
}

func (s *Service) fetchProjectPage(ctx context.Context, page, perPage, idAfter int, projects chan<- *pkg.Project,
) (nextPage int, err error) {
	ctxOne, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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

	projs, resp, err := s.gitlabClient.Projects.ListProjects(opt, gitlab.WithContext(ctxOne))
	if err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	for _, p := range projs {
		select {
		default:
			s.logger.Printf("fetching project: %d", p.ID)

			if !s.hasContributionsByCurrentUser(ctx, p.ID) {
				s.logger.Printf("project %d doesn't has user's %s contributions", p.ID, s.user.Email)

				continue
			}

			projects <- &pkg.Project{
				ID:   p.ID,
				Name: strconv.Itoa(p.ID),
			}
		case <-ctx.Done():
			return 0, nil
		}
	}

	if resp.CurrentPage >= resp.TotalPages {
		return 0, nil
	}

	return resp.NextPage, nil
}

func (s *Service) hasContributionsByCurrentUser(ctx context.Context, projectID int) bool {
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
			if strings.EqualFold(c.Email, s.user.Email) {
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
