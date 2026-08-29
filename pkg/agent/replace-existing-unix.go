//go:build !windows

package agent

import "os"

func replaceExistingFile(source, destination string) error {
	return os.Rename(source, destination)
}
