package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Project struct {
	ID int
}

type Commit struct {
	CommittedAt time.Time
	Message     string
}

func NewCommit(committedAt time.Time, projectID int, hash string) *Commit {
	return &Commit{
		CommittedAt: committedAt,
		Message:     fmt.Sprintf("Project: %d commit: %s", projectID, hash),
	}
}

func ParseCommitMessage(message string) (projectID int, hash string, err error) {
	const messagePartsCount = 4

	messageParts := strings.Split(message, " ")
	if len(messageParts) < messagePartsCount {
		return 0, "", NewErrInvalidArgument(fmt.Sprintf("wrong commit message: %s", message))
	}

	id, errAtoi := strconv.Atoi(messageParts[1])
	if errAtoi != nil {
		return 0, "", fmt.Errorf("failed to convert %s to project id: %w", messageParts[1], errAtoi)
	}

	projectID = id
	hash = messageParts[2]

	return projectID, hash, nil
}

type User struct {
	Name      string
	Email     string
	Username  string
	CreatedAt time.Time
}

type Committer struct {
	Name  string
	Email string
}
