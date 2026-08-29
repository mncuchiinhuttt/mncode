//go:build !windows

package tools

import "os"

func replaceExistingFile(source, destination string) error {
	return os.Rename(source, destination)
}
