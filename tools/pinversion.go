//go:build tools

package tools

// Keep a reference to the code generators so they are not removed by go mod tidy
import (
	_ "github.com/golangci/golangci-lint/pkg/exitcodes"
)
