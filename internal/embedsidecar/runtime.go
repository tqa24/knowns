// Package embedsidecar provides backward-compatible type aliases for the ONNX
// runtime that has been moved into the search package. The knowns-embed
// standalone binary still imports this package.
package embedsidecar

import "github.com/howznguyen/knowns/internal/search"

// ModelConfig is an alias for the ONNX model configuration.
type ModelConfig = search.ORTModelConfig

// Runtime is an alias for the ONNX runtime.
type Runtime = search.ORTRuntime

// ResolveSharedLibraryPath locates the ONNX Runtime shared library.
func ResolveSharedLibraryPath() string {
	return search.ResolveORTLibraryPath()
}
