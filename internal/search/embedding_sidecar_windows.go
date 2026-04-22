//go:build windows

package search

import (
	"os/exec"
	"syscall"
)

func configureSidecarCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
