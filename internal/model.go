package pkg

import (
	"time"
)

type Project struct {
	ID   int
	Name string
}

type Commit struct {
	// When is commit date.
	When time.Time `json:"when"`

	// Commit message.
	Message string `json:"message"`
}

type User struct {
	Name      string
	Email     string
	Username  string
	CreatedAt time.Time
}
