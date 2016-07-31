package main

import (
	"fmt"

	"github.com/bradleyfalzon/abicheck"
)

func main() {
	const (
		oldRev = "HEAD~1"
		newRev = "HEAD"
	)

	checker := abicheck.New()
	changes, err := checker.Check(oldRev, newRev)
	if err != nil {
		panic(err)
	}

	for _, change := range changes {
		fmt.Println(change)
	}

	parseTime, diffTime, sortTime := checker.Timing()
	fmt.Printf("Parse time: %v, Diff time: %v, Sort time: %v", parseTime, diffTime, sortTime)
}
