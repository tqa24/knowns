//go:build !windows

package search

import "os/exec"

func configureSidecarCommand(cmd *exec.Cmd) {}
