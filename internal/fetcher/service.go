package fetcher

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

const (
	commitsBufferLen = 100
)

type Service struct {
	logger       *log.Logger
	gitlabClient *gitlab.Client
}

func New(logger *log.Logger, gitlabClient *gitlab.Client) *Service {
	rand.Seed(time.Now().Unix())

	return &Service{
		logger:       logger,
		gitlabClient: gitlabClient,
	}
}

func (s *Service) FirstProject(ctx context.Context) (*pkg.Project, error) {
	projects, _, err := s.gitlabClient.Projects.ListProjects(&gitlab.ListProjectsOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) < 1 {
		return nil, nil
	}

	return &pkg.Project{ID: projects[0].ID}, nil
}

func (s *Service) Next(project *pkg.Project) <-chan *pkg.Commit {
	commits := make(chan *pkg.Commit, commitsBufferLen)

	go func(ctx context.Context) {
		defer close(commits)

		if project == nil {
			return
		}

		opt := &gitlab.ListCommitsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 50,
				Page:    1,
			},
		}

		for {
			comms, resp, err := s.gitlabClient.Commits.ListCommits(project.ID, opt, gitlab.WithContext(ctx))
			if err != nil {
				s.logger.Printf("failed to get commits for project: %d", project.ID)

				return
			}

			for _, c := range comms {
				if c.CommittedDate != nil {
					commits <- &pkg.Commit{When: *c.CommittedDate}
				}
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opt.Page = resp.NextPage
		}
	}(context.Background())

	return commits
}
