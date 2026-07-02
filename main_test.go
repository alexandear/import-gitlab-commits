package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexandear/import-gitlab-commits/internal/testutil"
)

func TestExecute(t *testing.T) {
	t.Run("error when wrong GITLAB_TOKEN", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "")

		err := Execute(testutil.NewLog(t))

		require.ErrorContains(t, err, "GITLAB_TOKEN")
	})

	t.Run("error when wrong GITLAB_BASE_URL", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", ":")

		err := Execute(testutil.NewLog(t))

		require.ErrorContains(t, err, "GITLAB_BASE_URL")
	})

	t.Run("error when wrong COMMITTER_NAME", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", "https://gitlab.com")
		t.Setenv("COMMITTER_NAME", "")

		err := Execute(testutil.NewLog(t))

		require.ErrorContains(t, err, "COMMITTER_NAME")
	})

	t.Run("error when wrong COMMITTER_EMAIL", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", "https://gitlab.com")
		t.Setenv("COMMITTER_NAME", "John Doe")
		t.Setenv("COMMITTER_EMAIL", "")

		err := Execute(testutil.NewLog(t))

		require.ErrorContains(t, err, "COMMITTER_EMAIL")
	})
}

func TestParseExtraEmails(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "single", raw: "john@example.com", want: []string{"john@example.com"}},
		{
			name: "multiple with spaces",
			raw:  "john@example.com, jane@example.com ,  extra@example.com",
			want: []string{"john@example.com", "jane@example.com", "extra@example.com"},
		},
		{name: "empty entries are dropped", raw: "john@example.com,,  ,", want: []string{"john@example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseExtraEmails(tt.raw))
		})
	}
}
