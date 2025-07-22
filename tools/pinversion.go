// Package tools is a placeholder for tooling imports, and should not be imported in production code.
// Despite `go tool` this package is needed to workaround the fact that @dependabot can't update indirect dependencies.
package tools

import (
	_ "github.com/golangci/golangci-lint/v2/pkg/exitcodes"
)
