package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	CommittedAt time.Time
	Message     string
}

func NewCommit(committedAt time.Time, projectID int64, hash string) *Commit {
	return &Commit{
		CommittedAt: committedAt,
		Message:     fmt.Sprintf("Project: %d commit: %s", projectID, hash),
	}
}

func ParseCommitMessage(message string) (projectID int64, hash string, _ error) {
	const messagePartsCount = 4

	messageParts := strings.Split(message, " ")
	if len(messageParts) < messagePartsCount {
		return 0, "", fmt.Errorf("wrong commit message: %s", message)
	}

	id, err := strconv.ParseInt(messageParts[1], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("failed to convert %s to project id: %w", messageParts[1], err)
	}

	projectID = id
	hash = messageParts[3]

	return projectID, hash, nil
}
