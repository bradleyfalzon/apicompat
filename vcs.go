package apicompat

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// revisionFS is a keyword to use the file system not VCS for read operations
const revisionFS = "."

// VCS defines a version control system
// the vcs should be able to handle calls to ReadFile concurrently
// A special case for the revision of "." (without quotes) is used to check
// local filesystem
type VCS interface {
	// ReadDir returns a list of files in a directory at revision
	ReadDir(revision, path string) ([]os.FileInfo, error)
	// OpenFile returns a reader for a given absolute path at a revision
	OpenFile(revision, path string) (io.ReadCloser, error)
	// DefaultRevision returns the default revisions if none specified
	DefaultRevision() (before string, after string)
}

// guarantee at compile time that *Git implements VCS
var _ VCS = (*Git)(nil)

// Git implements vcs and uses exec.Command to access repository
type Git struct {
	dir  string // directory of .git, used to for --git-dir
	base string // directory containing .git, used to to make paths relative
}

// NewGit returns a VCS based based on git.
func NewGit(path string) (*Git, error) {
	// Find the directory of .git, assumes git can find it via cwd
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	dir, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running %v: %v output: %q", cmd.Args, err, dir)
	}

	base := string(bytes.TrimSpace(dir))
	return &Git{
		base: base,
		dir:  filepath.Join(base, ".git"),
	}, nil
}

// rel returns the relative path to this path.
func (g *Git) rel(path string) (string, error) {
	relPath, err := filepath.Rel(g.base, path)
	if err != nil {
		return "", fmt.Errorf("git cannot make path relative: %v", err)
	}
	return relPath, nil
}

// ReadDir returns a list of files in a directory at revision
func (g *Git) ReadDir(revision, path string) ([]os.FileInfo, error) {
	if revision == revisionFS {
		return ioutil.ReadDir(path)
	}

	relPath, err := g.rel(path)
	if err != nil {
		return nil, err
	}
	relPath += string(os.PathSeparator)

	args := []string{"--git-dir", g.dir, "ls-tree", revision, relPath}
	ls, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("could not execute git %v, error: %s", args, err)
	}

	var files []os.FileInfo
	for _, file := range bytes.Split(ls, []byte{'\n'}) {
		// 100644 blob 78edbf3fb411055b4a3d4d3d137ccbec160ac956    .gitignore
		// 040000 tree e62f2cac29e1d6e31aeac65ded75df98b9c1be43    testdata
		fields := bytes.Fields(file)
		if len(fields) != 4 {
			continue
		}

		files = append(files, fileInfo{
			// name is basename (no directory structure)
			name: strings.TrimPrefix(string(fields[3]), relPath),
			dir:  bytes.Equal(fields[1], []byte("tree")),
		})
	}
	return files, nil
}

// OpenFile returns a reader for a given absolute path at a revision
func (g *Git) OpenFile(revision, path string) (io.ReadCloser, error) {
	if revision == revisionFS {
		return os.Open(path)
	}

	relPath, err := g.rel(path)
	if err != nil {
		return nil, err
	}

	var args = []string{"--git-dir", g.dir, "show", revision + ":" + relPath}
	contents, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("could not execute git with args %v: %v", args, err)
	}
	return ioutil.NopCloser(bytes.NewReader(contents)), nil
}

// DefaultRevision returns the default revisions if none specified
func (g *Git) DefaultRevision() (string, string) {
	// Check if there's unstaged changes, if so, return dot
	contents, _ := exec.Command("git", "--git-dir", g.dir, "ls-files", "-m").Output()
	if len(contents) > 0 {
		return "HEAD", "."
	}
	return "HEAD~1", "HEAD"
}

// fileInfo is a struct to simulate the real filesystem file info
type fileInfo struct {
	name string // base name of file
	dir  bool
}

// Name is one of the method needed to implement os.FileInfo
func (fi fileInfo) Name() string { return fi.name }

// Size is one of the method needed to implement os.FileInfo
func (fi fileInfo) Size() int64 { panic("not implemented") }

// Mode is one of the method needed to implement os.FileInfo
func (fi fileInfo) Mode() os.FileMode { panic("not implemented") }

// ModTime is one of the method needed to implement os.FileInfo
func (fi fileInfo) ModTime() time.Time { panic("not implemented") }

// IsDir is one of the method needed to implement os.FileInfo
func (fi fileInfo) IsDir() bool { return fi.dir }

// Sys is one of the method needed to implement os.FileInfo
func (fi fileInfo) Sys() interface{} { panic("not implemented") }

// guarantee at compile time that StrVCS implements VCS
var _ VCS = (*StrVCS)(nil)

// StrVCS provides a in memory vcs used for testing, but does not support
// subdirectories.
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

// ReadDir implements VCS.ReadDir
func (v StrVCS) ReadDir(revision, path string) (files []os.FileInfo, err error) {
	for file := range v.files[revision] {
		files = append(files, fileInfo{
			name: file,
		})
	}
	return files, nil
}

// OpenFile implements VCS.OpenFile
func (v StrVCS) OpenFile(revision, path string) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(v.files[revision][filepath.Base(path)])), nil
}

// DefaultRevision implements VCS.DefaultRevision
func (StrVCS) DefaultRevision() (string, string) {
	return "rev1", "rev2"
}
