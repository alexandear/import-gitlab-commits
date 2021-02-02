package main

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func main() {
	rand.Seed(time.Now().Unix())

	logger := log.New(os.Stdout, "", log.Lshortfile|log.Ltime)

	logger.Println("init repo in memory")

	fs := memfs.New()

	r, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		logger.Fatalf("failed to init: %v", err)
	}

	w, err := r.Worktree()
	if err != nil {
		logger.Fatalf("failed to get worktree: %v", err)
	}

	logger.Println("make initial commit")

	committer := &object.Signature{
		Name:  "Oleksandr Redko",
		Email: "oleksandr.red+github@gmail.com",
	}

	contributions := []time.Time{
		time.Date(2010, time.December, 20, 15, 15, rand.Intn(61), 0, time.UTC),
		time.Date(2015, time.July, 14, 4, 4, rand.Intn(61), 0, time.UTC),
	}

	for i, when := range contributions {
		committer.When = when

		h, cerr := w.Commit("commit "+strconv.Itoa(i), &git.CommitOptions{
			Author:    committer,
			Committer: committer,
		})
		if cerr != nil {
			logger.Fatalf("failed to commit: %v", cerr)
		}

		log.Println("committed", i, h)
	}

	logger.Println("log")

	ci, err := r.Log(&git.LogOptions{})
	if err != nil {
		logger.Fatalf("failed to get log: %v", err)
	}

	for {
		c, err := ci.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			logger.Fatalf("failed during iterate: %v", err)
		}

		logger.Printf("commit %s\nAuthor: %s\nAuthor date: %s\nCommitter: %s\nCommit date: %s\n   %s\n\n",
			c.Hash, c.Author.Name, c.Author.When, c.Committer.Name, c.Committer.When, c.Message)
	}
}
