# Introduction

[![Build Status](https://travis-ci.org/bradleyfalzon/abicheck.svg?branch=master)](https://travis-ci.org/bradleyfalzon/abicheck) [![Coverage Status](https://coveralls.io/repos/github/bradleyfalzon/abicheck/badge.svg?branch=master)](https://coveralls.io/github/bradleyfalzon/abicheck?branch=master) [![GoDoc](https://godoc.org/github.com/bradleyfalzon/abicheck?status.svg)](https://godoc.org/github.com/bradleyfalzon/abicheck)

`abicheck` is a tool to check for the introduction of backwards incompatible changes. Secondary tasks could be detecting
current semver and suggesting an appropriate increase, and generally listing all changes for help in release
notes/commit messages.

Try it:

```
go get -u github.com/bradleyfalzon/abicheck/cmd/abicheck
cd /your/project/dir/with/comitted/changes
abicheck
```

# Status

`abicheck` is currently under heavy development and heavy refactoring. This initial version was a proof of concept and shortcuts were taken. The current work is focused on (but no limited to):

- Code clean up, such as removing custom types and making it library friendly
- Add type checking to analyse inferred types
- Investigate additional interface checks (e.g., currently renaming an interface with the same methods would be detected as
    a breaking change, this isn't always true).
- Adding Mercurial, SVN and potentially other VCS systems.
- Improve VCS options such as:
    - Choosing the versions to compare
    - Checking of unstaged changes (currently only checks committed changes)
    - Filtering `vendor/` directories
    - Check subdirectories if ran from a subdirectory of the VCS (currently checks all committed code)
- Add docs, flow diagram and fixing of existing docs
- Improve output formats, such as vim quickfix
- Move these tasks to GitHub issues
- Once all other steps have been completed, performance will be investigated

# Testing

- This uses golden masters for the tests
- Run tests with: `go test`
- Save new results with: `go test -args update`
- Alternatively to do a test run: `go install && ( cd testgit; ./make.sh && abicheck )`
