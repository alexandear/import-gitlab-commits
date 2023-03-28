package cmd_test

import (
	"log"
	"testing"

	"github.com/alexandear/import-gitlab-commits/cmd"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	t.Run("error when wrong GITLAB_TOKEN", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "")

		err := cmd.Execute(log.Default())

		assert.ErrorContains(t, err, "GITLAB_TOKEN")
	})

	t.Run("error when wrong GITLAB_BASE_URL", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", ":")

		err := cmd.Execute(log.Default())

		assert.ErrorContains(t, err, "GITLAB_BASE_URL")
	})

	t.Run("error when wrong COMMITTER_NAME", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", "https://gitlab.com")
		t.Setenv("COMMITTER_NAME", "")

		err := cmd.Execute(log.Default())

		assert.ErrorContains(t, err, "COMMITTER_NAME")
	})

	t.Run("error when wrong COMMITTER_EMAIL", func(t *testing.T) {
		t.Setenv("GITLAB_TOKEN", "yourgitlabtoken")
		t.Setenv("GITLAB_BASE_URL", "https://gitlab.com")
		t.Setenv("COMMITTER_NAME", "John Doe")
		t.Setenv("COMMITTER_EMAIL", "")

		err := cmd.Execute(log.Default())

		assert.ErrorContains(t, err, "COMMITTER_EMAIL")
	})
}
