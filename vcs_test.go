package abicheck

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

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

// guarantee at compile time that strVCS implements VCS
var _ VCS = (*strVCS)(nil)

// strVCS provides a in memory vcs used for testing, but does not support
// subdirectories.
type strVCS struct {
	files map[string]map[string][]byte // revision -> path -> contents
}

// SetFile contents for a particular revision and path
func (v *strVCS) SetFile(revision, path string, contents []byte) {
	if v.files == nil {
		v.files = make(map[string]map[string][]byte)
	}
	if _, ok := v.files[revision]; !ok {
		v.files[revision] = make(map[string][]byte)
	}
	v.files[revision][path] = contents
}

func (v strVCS) ReadDir(revision, path string) (files []os.FileInfo, err error) {
	for file := range v.files[revision] {
		files = append(files, fileInfo{
			name: file,
		})
	}
	return files, nil
}

func (v strVCS) OpenFile(revision, path string) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(v.files[revision][filepath.Base(path)])), nil
}

func (strVCS) DefaultRevision() (string, string) {
	return "rev1", "rev2"
}
