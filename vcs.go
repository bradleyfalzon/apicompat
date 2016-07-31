package abicheck

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// vcs interface defines a version control system
// the vcs should be able to handle calls to ReadFile concurrently
type VCS interface {
	// ReadDir returns a list of files in a directory at revision recursively
	// and returns only .go files
	ReadDir(revision, path string) ([]string, error)
	// ReadFile returns the contents of a file at a revision
	ReadFile(revision, filename string) ([]byte, error)
}

var _ VCS = (*Git)(nil)

// git implements vcs and uses exec.Command to access repository
type Git struct{}

func (Git) ReadDir(revision, path string) ([]string, error) {
	// Add trailing slash if path is set and doesn't already contain one
	if path != "" && !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	ls, err := exec.Command("git", "ls-tree", "-r", "--name-only", revision, path).Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, file := range bytes.Split(ls, []byte{'\n'}) {
		if bytes.HasSuffix(file, []byte(".go")) {
			files = append(files, string(file))
		}
	}

	return files, nil
}

func (Git) ReadFile(revision, path string) ([]byte, error) {
	args := []string{"show", revision + ":" + path}
	contents, err := exec.Command("git", args...).Output()
	if err != nil {
		err = fmt.Errorf("could not execute git with args %v: %v", args, err)
	}
	return contents, err
}

var _ VCS = (*StrVCS)(nil)

// strvcs provides a in memory vcs used for testing
type StrVCS struct {
	files map[string]map[string][]byte // revision -> path -> contents
}

// SetFile contents for a particular revision and path
func (v *StrVCS) SetFile(revision, path string, contents []byte) {
	if v.files == nil {
		v.files = make(map[string]map[string][]byte)
	}
	if _, ok := v.files[revision]; !ok {
		v.files[revision] = make(map[string][]byte)
	}
	v.files[revision][path] = contents
}

func (v StrVCS) ReadDir(revision, path string) ([]string, error) {
	var files []string
	for file := range v.files[revision] {
		files = append(files, file)
	}
	return files, nil
}

func (v StrVCS) ReadFile(revision, path string) ([]byte, error) {
	return v.files[revision][path], nil
}
