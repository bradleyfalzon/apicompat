#!/usr/bin/bash -eu

DIR=$(basename `pwd`)

if [[ "$DIR" != 'testgit' ]]; then
    echo 'Not in testgit directory'
    exit 1
fi

# Remove old git dir
[[ -d .git ]] && rm -vrf .git testdata.go

# Initial commit
git init
cp ../testdata/a.go testdata.go
git add testdata.go
git commit -m '1st commit'

# Second commit
cat ../testdata/b.go > testdata.go
git add testdata.go
git commit -m '2nd commit'
