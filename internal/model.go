package pkg

import (
	"time"
)

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Commit struct {
	CommittedAt time.Time `json:"committed"`
	Message     string    `json:"message"`
}

type User struct {
	Name      string
	Email     string
	Username  string
	CreatedAt time.Time
}
