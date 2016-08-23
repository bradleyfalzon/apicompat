#!/usr/bin/env bash
# make.sh initialises a project, and commits the test data
set -eu

DIR=$(basename `pwd`)

if [[ "$DIR" != 'testdata' ]]; then
    echo 'Not in testdata directory'
    exit 1
fi

# Before/after is a simple application with a breaking change
BEFORE="package testdata\n\nconst A int = 1"
AFTER="package testdata\n\nconst A uint = 1"

# Remove old dirs
[[ -d gopath ]] && rm -rf gopath

# Initialise
mkdir -p gopath/src/example.com/lib/{b,internal,vendor}/c/
cd gopath
git init
git config --local user.name "testdata"
git config --local user.email "testdata@example.com"

# Initial commit
echo -e $BEFORE > src/example.com/lib/testdata.go
echo -e $BEFORE > src/example.com/lib/b/c/testdata.go
echo -e $BEFORE > src/example.com/lib/internal/c/testdata.go
echo -e $BEFORE > src/example.com/lib/vendor/c/testdata.go
git add .
git commit -m '1st commit'

# Second commit
echo -e $AFTER > src/example.com/lib/testdata.go
echo -e $AFTER > src/example.com/lib/b/c/testdata.go
echo -e $AFTER > src/example.com/lib/internal/c/testdata.go
echo -e $AFTER > src/example.com/lib/vendor/c/testdata.go
git add .
git commit -m '2nd commit'
