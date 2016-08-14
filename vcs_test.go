package abicheck

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

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
