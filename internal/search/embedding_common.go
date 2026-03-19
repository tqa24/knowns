package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ErrSemanticRuntimeUnavailable is returned when the current binary was built
// without ONNX Runtime support.
var ErrSemanticRuntimeUnavailable = fmt.Errorf("semantic search is unavailable in this build; rebuild with cgo enabled")

// findONNXLib searches for the ONNX Runtime shared library in standard locations.
func findONNXLib() string {
	if p := os.Getenv("KNOWNS_ONNX_LIB"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	home, _ := os.UserHomeDir()
	libName := onnxLibName()

	if home != "" {
		p := filepath.Join(home, ".knowns", "lib", libName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	for _, dir := range onnxSystemPaths() {
		p := filepath.Join(dir, libName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// onnxLibName returns the platform-specific library filename.
func onnxLibName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	default:
		return "libonnxruntime.so"
	}
}

// onnxSystemPaths returns common system library directories.
func onnxSystemPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/usr/local/lib", "/opt/homebrew/lib"}
	case "linux":
		return []string{"/usr/local/lib", "/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/lib/aarch64-linux-gnu"}
	case "windows":
		return []string{`C:\Program Files\onnxruntime\lib`}
	}
	return nil
}

// IsONNXAvailable checks if ONNX Runtime is available on the system.
func IsONNXAvailable() (bool, string) {
	path := findONNXLib()
	return path != "", path
}

// ONNXRuntimeDownloadURL returns the download URL for the current platform.
func ONNXRuntimeDownloadURL() (string, string, error) {
	const version = "1.24.3"
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform, ext string
	switch {
	case goos == "darwin" && goarch == "arm64":
		platform = "osx-arm64"
		ext = "tgz"
	case goos == "darwin" && goarch == "amd64":
		platform = "osx-arm64"
		ext = "tgz"
	case goos == "linux" && goarch == "amd64":
		platform = "linux-x64"
		ext = "tgz"
	case goos == "linux" && goarch == "arm64":
		platform = "linux-aarch64"
		ext = "tgz"
	case goos == "windows" && goarch == "amd64":
		platform = "win-x64"
		ext = "zip"
	case goos == "windows" && goarch == "arm64":
		platform = "win-arm64"
		ext = "zip"
	default:
		return "", "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}

	url := fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.%s",
		version, platform, version, ext)
	libName := onnxLibName()

	return url, libName, nil
}

func meanPool(outputData []float32, attentionMask []int64, seqLen, dims int) []float32 {
	result := make([]float32, dims)
	count := float32(0)

	for t := 0; t < seqLen; t++ {
		if attentionMask[t] == 0 {
			continue
		}
		count++
		offset := t * dims
		for d := 0; d < dims; d++ {
			result[d] += outputData[offset+d]
		}
	}

	if count > 0 {
		for d := range result {
			result[d] /= count
		}
	}

	return result
}
