//go:build !unix

package shareserver

import "os"

// lockDir opens path; other systems run the server for trying it out
// only, so it takes no lock there.
func lockDir(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
}
