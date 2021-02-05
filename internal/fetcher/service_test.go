package fetcher

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

func TestService_FetchProjects(t *testing.T) {
	git := initGit(t)

	s, err := New(log.New(newTestWriter(t), "", log.Lshortfile|log.Ltime), git, newCurrentUser(t, git))
	require.NoError(t, err)

	for p := range s.FetchProjects(context.Background()) {
		t.Log(p)
	}
}

func TestService_hasContributionsByUser(t *testing.T) {
	git := initGit(t)
	s, err := New(log.New(newTestWriter(t), "", log.Lshortfile|log.Ltime), git, newCurrentUser(t, git))
	require.NoError(t, err)

	assert.False(t, s.hasContributionsByCurrentUser(context.Background(), 3))
	assert.True(t, s.hasContributionsByCurrentUser(context.Background(), 575))
}

func initGit(t *testing.T) *gitlab.Client {
	token := os.Getenv("GITLAB_TOKEN")
	baseURL := os.Getenv("GITLAB_BASE_URL")
	if token == "" || baseURL == "" {
		t.SkipNow()
	}

	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	require.NoError(t, err)

	return git
}

type testWriter struct {
	t *testing.T
}

func newTestWriter(t *testing.T) *testWriter {
	return &testWriter{t: t}
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	str := string(p)

	w.t.Log(str[:len(str)-1])

	return len(p), nil
}

func newCurrentUser(t *testing.T, gitlabClient *gitlab.Client) *pkg.User {
	u, _, err := gitlabClient.Users.CurrentUser()
	require.NoError(t, err)

	return &pkg.User{
		Name:  u.Name,
		Email: u.Email,
	}
}
