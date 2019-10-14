package fuse

import (
	"fmt"
	"os/exec"
)

func mountFailedErr(err error) error {
	if exited, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("Received %v running /sbin/mount_fusefs", exited)
	}
	return err
}
