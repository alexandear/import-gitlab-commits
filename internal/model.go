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
