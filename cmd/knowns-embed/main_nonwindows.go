//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "knowns-embed native sidecar is only built for Windows in this repo")
	os.Exit(1)
}
