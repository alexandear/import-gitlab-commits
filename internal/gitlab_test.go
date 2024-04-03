//go:build integration

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

func TestGitLabCurrentUser(t *testing.T) {
	gl := app.NewGitLab(testutil.NewLog(t), gitlabClient(t))

	user, err := gl.CurrentUser(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, user.Name)
	assert.NotEmpty(t, user.Emails)
	assert.NotEmpty(t, user.Username)
	assert.False(t, user.CreatedAt.IsZero())
}

func gitlabClient(t *testing.T) *gitlab.Client {
	t.Helper()

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		t.Fatal("GITLAB_TOKEN is required")
	}

	baseURL := os.Getenv("GITLAB_BASE_URL")
	if baseURL == "" {
		t.Fatal("GITLAB_BASE_URL is required")
	}

	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	require.NoError(t, err)

	return client
}
