package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradleyfalzon/abicheck"
)

func main() {
	// TODO print CLI arguments, note that it does support GOARCH, GOOS, GOPATH etc, ./... works too
	before := flag.String("before", "", "Compare revision before, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	after := flag.String("after", "", "Compare revision after, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	excludeFile := flag.String("exclude-file", "", "Exclude files based on regexp pattern")
	excludeDir := flag.String("exclude-ipath", "", "Exclude directory based on regexp pattern")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()
	path := flag.Arg(0)

	// TODO make it auto discover
	git, err := abicheck.NewGit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "abicheck: %s\n", err)
		os.Exit(2)
	}

	args := []func(*abicheck.Checker){abicheck.SetVCS(git)}
	if *verbose {
		args = append(args, abicheck.SetVLog(os.Stdout))
	}
	if *excludeFile != "" {
		args = append(args, abicheck.SetExcludeFile(*excludeFile))
	}
	if *excludeDir != "" {
		args = append(args, abicheck.SetExcludeDir(*excludeDir))
	}

	checker := abicheck.New(args...)
	changes, err := checker.Check(path, *before, *after)
	if err != nil {
		fmt.Fprintf(os.Stderr, "abicheck: %s\n", err)
		os.Exit(1)
	}

	for _, change := range changes {
		fmt.Println(change)
	}
}
