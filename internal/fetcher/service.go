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

func (s *Service) FetchCommits(ctx context.Context, project *pkg.Project) chan *pkg.Commit {
	const chanSize = 100

	commits := make(chan *pkg.Commit, chanSize)

	go func() {
		defer close(commits)

		if project == nil {
			return
		}

		opt := &gitlab.ListCommitsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: chanSize,
				Page:    1,
			},
		}

		for {
			comms, resp, err := s.gitlabClient.Commits.ListCommits(project.ID, opt, gitlab.WithContext(ctx))
			if err != nil {
				s.logger.Printf("failed to get commits for project %d: %v", project.ID, err)

				return
			}

			for _, c := range comms {
				if c.CommittedDate == nil {
					continue
				}

				s.logger.Printf("fetching commit: %s", c.ID)

				commits <- &pkg.Commit{
					When:    *c.CommittedDate,
					Message: c.CommittedDate.String(),
				}
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opt.Page = resp.NextPage
		}
	}()

	return commits
}

func (s *Service) FetchProjects(ctx context.Context) <-chan *pkg.Project {
	const chanSize = 100

	projects := make(chan *pkg.Project, chanSize)

	go func() {
		defer close(projects)

		page := 1
		for page > 0 {
			select {
			case <-ctx.Done():
				s.logger.Printf("fetching canceled")

				return
			default:
			}

			nextPage, err := s.fetchProjectPage(ctx, page, chanSize, projects)
			if err != nil {
				s.logger.Printf("failed to fetch one project page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return projects
}

func (s *Service) fetchProjectPage(ctx context.Context, page, perPage int, projects chan<- *pkg.Project,
) (nextPage int, err error) {
	ctxOne, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
		Simple:     gitlab.Bool(true),
		Membership: gitlab.Bool(true),
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
				s.logger.Printf("project %d hasn't current user %s contributions", p.ID, s.user.Email)

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
		s.logger.Printf("total pages")

		return 0, nil
	}

	return resp.NextPage, nil
}

func (s *Service) hasContributionsByCurrentUser(ctx context.Context, projectID int) bool {
	opt := &gitlab.ListContributorsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
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
