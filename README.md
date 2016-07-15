_In development or abandoned_ (check last commit time to determine which)

# Testing

- This uses golden masters for the tests
- Run tests with: `go test`
- Save new results with: `go test -args update`
- Alternatively to do a test run: `go install && ( cd testgit; ./make.sh && abicheck )`
