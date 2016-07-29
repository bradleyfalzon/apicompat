package main

import "github.com/bradleyfalzon/abicheck"

func main() {
	const (
		oldRev = "HEAD~1"
		newRev = "HEAD"
	)

	checker := abicheck.New(oldRev, newRev)
	checker.Check()
}
