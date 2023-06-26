package test

import (
	"log"
	"testing"
)

// Writer used for a logger in tests.
type Writer struct {
	t *testing.T
}

func NewLog(t *testing.T) *log.Logger {
	t.Helper()

	return log.New(NewWriter(t), "", log.Lshortfile|log.Ltime)
}

func NewWriter(t *testing.T) *Writer {
	t.Helper()

	return &Writer{t: t}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	str := string(p)

	w.t.Log(str[:len(str)-1])

	return len(p), nil
}
