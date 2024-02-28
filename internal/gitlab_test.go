package app_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	app "github.com/alexandear/import-gitlab-commits/internal"
	"github.com/alexandear/import-gitlab-commits/internal/testutil"
)

func TestGitLabHasUserContributions(t *testing.T) {
	git := initGit(t)
	s := app.NewGitLab(testutil.NewLog(t), git)
	user := newCurrentUser(t, git)

	assert.False(t, s.HasUserContributions(context.Background(), user, 3))
	assert.True(t, s.HasUserContributions(context.Background(), user, 575))
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

func newCurrentUser(t *testing.T, gitlabClient *gitlab.Client) *app.User {
	t.Helper()

	user, _, err := gitlabClient.Users.CurrentUser()
	require.NoError(t, err)

	emails, _, err := gitlabClient.Users.ListEmails()
	require.NoError(t, err)

	emailAddresses := make([]string, 0, 10)

	for _, email := range emails {
		emailAddresses = append(emailAddresses, email.Email)
	}

	return &app.User{
		Name:      user.Name,
		Emails:    emailAddresses,
		CreatedAt: *user.CreatedAt,
	}
}
