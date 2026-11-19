//go:build unix

package shareserver

import (
	"os"
	"syscall"
)

// lockDir takes path for this process alone, for as long as it runs: a
// second server on the same volume would interleave its writes.
func lockDir(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()

		return nil, err
	}

	return f, nil
}
