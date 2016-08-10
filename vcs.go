package abicheck

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// revisionFS is a keyword to use the file system not VCS for read operations
const revisionFS = "."

// vcs interface defines a version control system
// the vcs should be able to handle calls to ReadFile concurrently
// A special case for the revision of "." (without quotes) is used to check
// local filesystem
type VCS interface {
	// ReadDir returns a list of files in a directory at revision
	ReadDir(revision, path string) ([]string, error)
	// ReadFile returns the contents of a file at a revision
	ReadFile(revision, filename string) ([]byte, error)
	// DefaultRevision returns the default revisions if none specified
	DefaultRevision() (before string, after string)
}

var _ VCS = (*Git)(nil)

// git implements vcs and uses exec.Command to access repository
type Git struct{}

func (Git) ReadDir(revision, path string) ([]string, error) {
	if revision == revisionFS {
		return readFSDir(path)
	}

	// Add trailing slash if path is set and doesn't already contain one
	if path != "" && !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	ls, err := exec.Command("git", "ls-tree", "--name-only", revision, path).Output()
	if err != nil {
		return nil, fmt.Errorf("could not git ls-tree revision: %q, path: %q, error: %s", revision, path, err)
	}

	var files []string
	for _, file := range bytes.Split(ls, []byte{'\n'}) {
		files = append(files, string(file))
	}

	return files, nil
}

func (Git) ReadFile(revision, path string) ([]byte, error) {
	if revision == revisionFS {
		return readFSFile(path)
	}

	args := []string{"show", revision + ":" + path}
	contents, err := exec.Command("git", args...).Output()
	if err != nil {
		err = fmt.Errorf("could not execute git with args %v: %v", args, err)
	}
	return contents, err
}

// readFSDir reads contents from the file system directly
func readFSDir(path string) ([]string, error) {
	if path == "" {
		path = "."
	}
	dirFiles, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("could not read filesystem dir %q: %v", path, err)
	}

	files := make([]string, 0, len(dirFiles))
	for _, file := range dirFiles {
		if !file.IsDir() {
			files = append(files, filepath.Join(path, file.Name()))
		}
	}
	return files, nil
}

// readFSFile reads a file from the file system directly
func readFSFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (Git) DefaultRevision() (string, string) {
	// Check if there's unstaged changes, if so, return dot
	contents, _ := exec.Command("git", "ls-files", "-m").Output()
	if len(contents) > 0 {
		return "HEAD", "."
	}
	return "HEAD~1", "HEAD"
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

func (StrVCS) DefaultRevision() (string, string) {
	return "rev1", "rev2"
}
