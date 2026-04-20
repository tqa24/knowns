package search

import "fmt"

// ErrSemanticRuntimeUnavailable is returned when the embedding sidecar is unavailable.
var ErrSemanticRuntimeUnavailable = fmt.Errorf("semantic search is unavailable: knowns-embed sidecar binary not found")

// IsONNXAvailable is kept as an alias for backward compat with callers that
// still ask "is the embedding runtime available?". It now reports sidecar status.
//
// Deprecated: use IsSidecarAvailable.
func IsONNXAvailable() (bool, string) {
	return IsSidecarAvailable()
}
