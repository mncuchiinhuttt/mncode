//go:build !windows

package accounts

import "os"

func replaceExistingFile(source, destination string) error {
	return os.Rename(source, destination)
}
