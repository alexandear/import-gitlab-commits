package app

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	"github.com/alexandear/import-gitlab-commits/test"
)

func TestService_hasContributionsByUser(t *testing.T) {
	git := initGit(t)
	s := NewGitLab(test.NewLog(t), git)
	user := newCurrentUser(t, git)

	assert.False(t, s.hasUserContributions(context.Background(), user, 3))
	assert.True(t, s.hasUserContributions(context.Background(), user, 575))
}

func initGit(t *testing.T) *gitlab.Client {
	t.Helper()

	token := os.Getenv("GITLAB_TOKEN")
	baseURL := os.Getenv("GITLAB_BASE_URL")

	if token == "" || baseURL == "" {
		t.SkipNow()
	}

	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	require.NoError(t, err)

	return git
}

func newCurrentUser(t *testing.T, gitlabClient *gitlab.Client) *User {
	t.Helper()

	user, _, err := gitlabClient.Users.CurrentUser()
	require.NoError(t, err)

	return &User{
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: *user.CreatedAt,
	}
}
