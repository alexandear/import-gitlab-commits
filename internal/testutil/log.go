// Package testutil provides utilities for testing, including test-friendly logging.
package testutil

import (
	"log"
	"testing"
)

// writer used for a logger in tests.
type writer struct {
	t *testing.T
}

// NewLog creates a new logger that writes to the test's output using t.Log.
// The logger includes short file names and timestamps in the log format.
// This is useful for capturing log output during tests without interfering
// with the test runner's output formatting.
func NewLog(t *testing.T) *log.Logger {
	t.Helper()

	return log.New(newWriter(t), "", log.Lshortfile|log.Ltime)
}

func newWriter(t *testing.T) *writer {
	t.Helper()

	return &writer{t: t}
}

func (w *writer) Write(p []byte) (n int, err error) {
	str := string(p)

	w.t.Log(str[:len(str)-1])

	return len(p), nil
}
