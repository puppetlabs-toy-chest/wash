package munge

import (
	"fmt"
	"os"
	"strconv"
)

// ToUintMode converts a given mode to a uint64.
// The mode can be either an integer or a string
// representing an octal/hex/decimal number.
func ToUintMode(mode interface{}) (uint64, error) {
	switch t := mode.(type) {
	case uint64:
		return t, nil
	case int64:
		return uint64(t), nil
	case float64:
		if t != float64(uint64(t)) {
			return 0, fmt.Errorf("the provided mode %v is a decimal number", t)
		}
		return uint64(t), nil
	case string:
		if intMode, err := strconv.ParseUint(t, 0, 32); err == nil {
			return intMode, nil
		}
		return 0, fmt.Errorf("the provided mode %v is not a octal/hex/decimal number", t)
	default:
		return 0, fmt.Errorf("the provided mode %v is not a uint64, int64, float64, or string", mode)
	}
}

// ToFileMode converts a given mode into an os.FileMode object.
// The mode can be either an integer or a string representing
// an octal/hex/decimal number.
func ToFileMode(mode interface{}) (os.FileMode, error) {
	if fileMode, ok := mode.(os.FileMode); ok {
		return fileMode, nil
	}
	intMode, err := ToUintMode(mode)
	if err != nil {
		return 0, err
	}
	fileMode := os.FileMode(intMode & 0777)
	// Mapping from http://man7.org/linux/man-pages/man7/inode.7.html for stat output
	for bits, mod := range map[uint64]os.FileMode{
		0140000: os.ModeSocket,
		0120000: os.ModeSymlink,
		// Skip file, absence of these implies a regular file.
		0060000: os.ModeDevice,
		0040000: os.ModeDir,
		0020000: os.ModeCharDevice,
		0010000: os.ModeNamedPipe,
		0004000: os.ModeSetuid,
		0002000: os.ModeSetgid,
		0001000: os.ModeSticky,
	} {
		// Ensure exact match of all bits in the mask.
		if intMode&bits == bits {
			fileMode |= mod
		}
	}
	return fileMode, nil
}
