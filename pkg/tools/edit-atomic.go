package tools

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type editPathLock struct {
	mu   sync.Mutex
	refs int
}

var editPathLocks struct {
	sync.Mutex
	items map[string]*editPathLock
}

func acquireEditPath(path string) func() {
	editPathLocks.Lock()
	if editPathLocks.items == nil {
		editPathLocks.items = make(map[string]*editPathLock)
	}
	lock := editPathLocks.items[path]
	if lock == nil {
		lock = &editPathLock{}
		editPathLocks.items[path] = lock
	}
	lock.refs++
	editPathLocks.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		editPathLocks.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(editPathLocks.items, path)
		}
		editPathLocks.Unlock()
	}
}

func atomicEditWrite(path string, original, updated []byte, mode os.FileMode) error {
	latest, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(latest, original) {
		return fmt.Errorf("stale edit rejected: file changed while edit was being prepared")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mncode-edit-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode.Perm()); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(updated); err != nil {
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
	if err := replaceExistingFile(tmpPath, path); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
