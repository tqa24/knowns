package search

import "fmt"

// ErrSemanticRuntimeUnavailable is returned when the ONNX runtime library is unavailable.
var ErrSemanticRuntimeUnavailable = fmt.Errorf("semantic search is unavailable: ONNX runtime shared library not found")

// IsSidecarAvailable is kept for backward compat. It now checks whether the
// native ONNX runtime library can be located.
//
// Deprecated: use IsONNXAvailable.
func IsSidecarAvailable() (bool, string) {
	return IsONNXAvailable()
}
