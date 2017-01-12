package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradleyfalzon/apicompat"
)

const (
	exitCodeNoError       = 0
	exitCodeInternalError = 1
	exitCodeBreaking      = 2
)

func main() {
	// TODO print CLI arguments, note that it does support GOARCH, GOOS, GOPATH etc, ./... works too
	before := flag.String("before", "", "Compare revision before, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	after := flag.String("after", "", "Compare revision after, leave unset for the VCS default or . to bypass VCS and use filesystem version")
	excludeFile := flag.String("exclude-file", "", "Exclude files based on regexp pattern")
	excludeDir := flag.String("exclude-dir", "", "Exclude directory based on regexp pattern")
	allChanges := flag.Bool("all", false, "Show all changes, not just breaking")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()
	path := flag.Arg(0)
	rel, rec, err := apicompat.RelativePathToTarget(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeInternalError)
	}

	// TODO make it auto discover
	git, err := apicompat.NewGit(rel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeInternalError)
	}

	args := []func(*apicompat.Checker){apicompat.SetVCS(git)}
	if *verbose {
		args = append(args, apicompat.SetVLog(os.Stdout))
	}
	if *excludeFile != "" {
		args = append(args, apicompat.SetExcludeFile(*excludeFile))
	}
	if *excludeDir != "" {
		args = append(args, apicompat.SetExcludeDir(*excludeDir))
	}

	checker := apicompat.New(args...)
	changes, err := checker.Check(rel, rec, *before, *after)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeInternalError)
	}

	exitCode := exitCodeNoError
	for _, change := range changes {
		switch {
		case change.Change == apicompat.Breaking:
			exitCode = exitCodeBreaking
			fmt.Print(change)
		case *allChanges:
			fmt.Print(change)
		}
	}
	os.Exit(exitCode)
}
