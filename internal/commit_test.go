package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	app "github.com/alexandear/import-gitlab-commits/internal"
)

func TestNewCommit(t *testing.T) {
	committedAt, err := time.Parse(time.RFC3339, "2021-08-01T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	projectID := int64(2323)
	hash := "9bc457f81c86307f28662b40a164105f14df64e3"

	commit := app.NewCommit(committedAt, projectID, hash)

	assert.Equal(t, &app.Commit{
		CommittedAt: committedAt,
		Message:     "Project: 2323 commit: 9bc457f81c86307f28662b40a164105f14df64e3",
	}, commit)
}

func TestParseCommitMessage(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		msg := "Project: 2323 commit: 9bc457f81c86307f28662b40a164105f14df64e3"
		projectID, hash, err := app.ParseCommitMessage(msg)

		require.NoError(t, err)
		assert.Equal(t, int64(2323), projectID)
		assert.Equal(t, "9bc457f81c86307f28662b40a164105f14df64e3", hash)
	})

	t.Run("wrong project id", func(t *testing.T) {
		msg := "Project: PROJ commit: 9bc457f81c86307f28662b40a164105f14df64e3"
		_, _, err := app.ParseCommitMessage(msg)

		assert.Error(t, err)
	})
}
