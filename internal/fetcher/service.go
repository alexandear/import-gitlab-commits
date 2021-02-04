package fetcher

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

type Service struct {
	logger       *log.Logger
	gitlabClient *gitlab.Client
}

func New(logger *log.Logger, gitlabClient *gitlab.Client) *Service {
	return &Service{
		logger:       logger,
		gitlabClient: gitlabClient,
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

			nextPage, err := s.fetchOneProjectPage(ctx, chanSize, page, projects)
			if err != nil {
				s.logger.Printf("failed to fetch one project page: %v", err)

				return
			}

			page = nextPage
		}
	}()

	return projects
}

func (s *Service) fetchOneProjectPage(ctx context.Context, chanSize, page int, projects chan<- *pkg.Project,
) (nextPage int, err error) {
	ctxOne, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: chanSize,
			Page:    page,
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
		case projects <- &pkg.Project{
			ID:   p.ID,
			Name: strconv.Itoa(p.ID),
		}:
			s.logger.Printf("fetching project: %d", p.ID)
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
