package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// vcs interface defines a version control system
// the vcs should be able to handle calls to ReadFile concurrently
type vcs interface {
	// ReadDir returns a list of files in a directory at revision recursively
	// and returns only .go files
	ReadDir(revision, path string) ([]string, error)
	// ReadFile returns the contents of a file at a revision
	ReadFile(revision, filename string) ([]byte, error)
}

var _ vcs = (*git)(nil)

// git implements vcs and uses exec.Command to access repository
type git struct{}

func (git) ReadDir(revision, path string) ([]string, error) {
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

func (git) ReadFile(revision, path string) ([]byte, error) {
	args := []string{"show", revision + ":" + path}
	contents, err := exec.Command("git", args...).Output()
	if err != nil {
		err = fmt.Errorf("could not execute git with args %v: %v", args, err)
	}
	return contents, err
}
