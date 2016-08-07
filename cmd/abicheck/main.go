package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradleyfalzon/abicheck"
)

func main() {
	const (
		oldRev = "HEAD~1"
		newRev = "HEAD"
	)

	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	var checker *abicheck.Checker
	if *verbose {
		checker = abicheck.New(abicheck.SetVLog(os.Stdout))
	} else {
		checker = abicheck.New()
	}

	changes, err := checker.Check(oldRev, newRev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abicheck: %s\n", err)
		os.Exit(1)
	}

	for _, change := range changes {
		fmt.Println(change)
	}
}
