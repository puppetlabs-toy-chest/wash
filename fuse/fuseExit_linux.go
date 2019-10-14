package fuse

import (
	"fmt"
	"os/exec"
)

func mountFailedErr(err error) error {
	if exited, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("Received %v running fusermount", exited)
	}
	return err
}
