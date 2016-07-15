*In development or abandoned* (check last commit time to determine which)

[![Build Status](https://travis-ci.org/bradleyfalzon/abicheck.svg?branch=master)](https://travis-ci.org/bradleyfalzon/abicheck)

# Testing

- This uses golden masters for the tests
- Run tests with: `go test`
- Save new results with: `go test -args update`
- Alternatively to do a test run: `go install && ( cd testgit; ./make.sh && abicheck )`
