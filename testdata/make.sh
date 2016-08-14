#!/usr/bin/env bash
# make.sh initialises a project, and commits the test data
set -eu

DIR=$(basename `pwd`)

if [[ "$DIR" != 'testdata' ]]; then
    echo 'Not in testdata directory'
    exit 1
fi

# Remove old dirs
[[ -d gopath ]] && rm -rf gopath

# Initialise
mkdir -p gopath/src/example.com/lib/b/c/
cd gopath
git init
git config --local user.name "testdata"
git config --local user.email "testdata@example.com"

# Initial commit
cp ../before.go src/example.com/lib/testdata.go
cp ../before.go src/example.com/lib/b/c/testdata.go
git add .
git commit -m '1st commit'

# Second commit
cat ../after.go > src/example.com/lib/testdata.go
cat ../after.go > src/example.com/lib/b/c/testdata.go
git add .
git commit -m '2nd commit'
