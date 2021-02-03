package bbolt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.etcd.io/bbolt"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

type Storage struct {
	db *bbolt.DB
}

func New(db *bbolt.DB) *Storage {
	return &Storage{
		db: db,
	}
}

func (s *Storage) AddCommit(projectName string, commit *pkg.Commit) error {
	key := []byte(commit.When.Format(time.RFC3339))

	cb, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}

	if err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(projectName))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		if b.Get(key) != nil {
			return nil
		}

		if err := b.Put(key, cb); err != nil {
			return fmt.Errorf("failed to put: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

func (s *Storage) NextCommit(projectName string) chan *pkg.Commit {
	commits := make(chan *pkg.Commit, 1000)

	go func() {
		defer close(commits)

		if err := s.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(projectName))

			return b.ForEach(func(k, v []byte) error {
				commit := &pkg.Commit{}
				if err := json.Unmarshal(v, commit); err != nil {
					return fmt.Errorf("failed to unmarshal commit %v: %w", v, err)
				}

				commits <- commit

				return nil
			})
		}); err != nil {
			log.Println(err)
		}
	}()

	return commits
}
