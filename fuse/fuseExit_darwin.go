package fuse

import (
	"fmt"
	"os"
	"os/exec"

	"bazil.org/fuse"
)

func mountFailedErr(err error) error {
	if exited, ok := err.(*exec.ExitError); ok {
		// load_osxfuse or mount_osxfuse failed. We can't determine which here.
		// Determine which would have been used so we can return the full path.
		for _, loc := range []fuse.OSXFUSEPaths{fuse.OSXFUSELocationV3, fuse.OSXFUSELocationV2} {
			if _, err := os.Stat(loc.Mount); !os.IsNotExist(err) {
				return fmt.Errorf("Received %v running %v or %v", exited, loc.Load, loc.Mount)
			}
		}
	}
	return err
}
