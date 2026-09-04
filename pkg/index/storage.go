package index

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"mncode/pkg/commandutil"
	"mncode/pkg/tools"
)

func staleDocument(root string, doc Document) bool {
	path, err := tools.ResolveWorkspacePath(root, doc.Path, false)
	if err != nil {
		return true
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != doc.Size {
		return true
	}
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, doc.Size+1)); err != nil {
		return true
	}
	return hex.EncodeToString(hash.Sum(nil)) != doc.SHA256
}

func writePrivateBytes(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := commandutil.EnsurePrivateDirectory(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".index-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
