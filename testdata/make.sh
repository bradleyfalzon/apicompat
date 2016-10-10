#!/usr/bin/env bash
# make.sh initialises a project, and commits the test data
set -eu

DIR=$(basename `pwd`)

if [[ "$DIR" != 'testdata' ]]; then
    echo 'Not in testdata directory'
    exit 1
fi

# Before/after is a library with a breaking change
BEFORE_LIB="package testdata\n\nconst A int = 1"
AFTER_LIB="package testdata\n\nconst A uint = 1"

# Before_main/after_main is a main application with a breaking change
BEFORE_MAIN="package main\n\nconst A int = 1"
AFTER_MAIN="package main\n\nconst A uint = 1"

# Remove old dirs
[[ -d gopath ]] && rm -rf gopath

# Initialise
mkdir -p gopath/src/example.com/lib/{b,internal,vendor,main}/c/
cd gopath
git init
git config --local user.name "testdata"
git config --local user.email "testdata@example.com"

# Initial commit
echo -e $BEFORE_LIB > src/example.com/lib/testdata.go
echo -e $BEFORE_LIB > src/example.com/lib/b/c/testdata.go
echo -e $BEFORE_LIB > src/example.com/lib/internal/c/testdata.go
echo -e $BEFORE_LIB > src/example.com/lib/vendor/c/testdata.go
echo -e $BEFORE_MAIN > src/example.com/lib/main/main.go
git add .
git commit -m '1st commit'

# Second commit
echo -e $AFTER_LIB > src/example.com/lib/testdata.go
echo -e $AFTER_LIB > src/example.com/lib/b/c/testdata.go
echo -e $AFTER_LIB > src/example.com/lib/internal/c/testdata.go
echo -e $AFTER_LIB > src/example.com/lib/vendor/c/testdata.go
echo -e $AFTER_MAIN > src/example.com/lib/main/main.go
git add .
git commit -m '2nd commit'
