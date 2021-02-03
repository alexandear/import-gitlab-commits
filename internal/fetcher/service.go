package fetcher

import (
	"math/rand"
	"time"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

type Service struct{}

func New() *Service {
	rand.Seed(time.Now().Unix())

	return &Service{}
}

func (s *Service) Next() <-chan *pkg.Commit {
	fakeCommits := []*pkg.Commit{
		{
			When: time.Date(2010, time.December, 20, 15, 15, rand.Intn(61), 0, time.UTC),
		},
		{
			When: time.Date(2015, time.July, 14, 4, 4, rand.Intn(61), 0, time.UTC),
		},
	}
	commits := make(chan *pkg.Commit, 100)

	go func() {
		for _, f := range fakeCommits {
			commits <- f
		}

		close(commits)
	}()

	return commits
}
