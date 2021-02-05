package bbolt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5/utils/binary"
	"go.etcd.io/bbolt"

	pkg "github.com/alexandear/fake-private-contributions/internal"
)

const (
	bucketProjects = "projects"
)

type Storage struct {
	db *bbolt.DB
}

func New(db *bbolt.DB) *Storage {
	return &Storage{
		db: db,
	}
}

func (s *Storage) AddCommit(projectID int, commit *pkg.Commit) error {
	key := []byte(commit.When.Format(time.RFC3339))

	pb, err := intToBytes(projectID)
	if err != nil {
		return fmt.Errorf("failed to convert project id to bytes: %w", err)
	}

	cb, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}

	if err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(pb)
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

func (s *Storage) NextCommit(projectID int) chan *pkg.Commit {
	commits := make(chan *pkg.Commit, 100)

	go func() {
		defer close(commits)

		bucket, err := intToBytes(projectID)
		if err != nil {
			log.Printf("failed to convert project id to bytes: %v", err)

			return
		}

		if err := s.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucket)

			if b == nil {
				return nil
			}

			return b.ForEach(func(k, v []byte) error {
				commit := &pkg.Commit{}
				if err := json.Unmarshal(v, commit); err != nil {
					return fmt.Errorf("failed to unmarshal commit %v: %w", v, err)
				}

				commits <- commit

				return nil
			})
		}); err != nil {
			log.Printf("failed to view commits: %v", err)
		}
	}()

	return commits
}

func (s *Storage) AddProject(project *pkg.Project) error {
	key, err := intToBytes(project.ID)
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	pb, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	if err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketProjects))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		if b.Get(key) != nil {
			return nil
		}

		if err := b.Put(key, pb); err != nil {
			return fmt.Errorf("failed to put: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

func (s *Storage) NextProject() chan *pkg.Project {
	projects := make(chan *pkg.Project, 100)

	go func() {
		defer close(projects)

		if err := s.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(bucketProjects))

			if b == nil {
				return nil
			}

			return b.ForEach(func(k, v []byte) error {
				project := &pkg.Project{}
				if err := json.Unmarshal(v, project); err != nil {
					return fmt.Errorf("failed to unmarshal project %v: %w", v, err)
				}

				projects <- project

				return nil
			})
		}); err != nil {
			log.Printf("failed to view projects: %v", err)
		}
	}()

	return projects
}

func (s *Storage) LastProject() (*pkg.Project, error) {
	project := &pkg.Project{}

	if err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketProjects))

		if b == nil {
			return nil
		}

		cur := b.Cursor()
		if cur == nil {
			return nil
		}

		k, v := cur.Last()
		if k == nil {
			return nil
		}

		if err := json.Unmarshal(v, project); err != nil {
			return fmt.Errorf("failed to unmarshal project %v: %w", v, err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to view projects: %w", err)
	}

	return project, nil
}

func intToBytes(v int) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := binary.WriteUint32(buf, uint32(v)); err != nil {
		return nil, fmt.Errorf("failed to write uint32 %d: %w", v, err)
	}

	return buf.Bytes(), nil
}
