package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradleyfalzon/abicheck"
)

func main() {
	before := flag.String("before", "", "Compare revision before, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	after := flag.String("after", "", "Compare revision after, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	var args []func(*abicheck.Checker)
	if *verbose {
		args = append(args, abicheck.SetVLog(os.Stdout))
	}

	checker := abicheck.New(args...)
	changes, err := checker.Check(*before, *after)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abicheck: %s\n", err)
		os.Exit(1)
	}

	for _, change := range changes {
		fmt.Println(change)
	}
}
