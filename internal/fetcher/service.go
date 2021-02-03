package fetcher

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

type Storage interface {
	AddCommit(projectName string, commit *pkg.Commit) error
}

type Service struct {
	logger       *log.Logger
	gitlabClient *gitlab.Client
	storage      Storage
}

func New(logger *log.Logger, gitlabClient *gitlab.Client, storage Storage) *Service {
	return &Service{
		logger:       logger,
		gitlabClient: gitlabClient,
		storage:      storage,
	}
}

func (s *Service) FetchCommits(ctx context.Context, project *pkg.Project) {
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
			if c.CommittedDate == nil {
				continue
			}

			s.logger.Printf("fetching commit: %v", c.ID)

			if err := s.storage.AddCommit(project.Name, &pkg.Commit{
				When:    *c.CommittedDate,
				Message: c.CommittedDate.String(),
			}); err != nil {
				s.logger.Printf("failed to add commit %v to storage: %v", c, err)
			}
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
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

	return &pkg.Project{ID: projects[0].ID, Name: strconv.Itoa(projects[0].ID)}, nil
}
