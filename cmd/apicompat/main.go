package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradleyfalzon/apicompat"
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

	// TODO make it auto discover
	git, err := apicompat.NewGit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apicompat: %s\n", err)
		os.Exit(2)
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
	changes, err := checker.Check(path, *before, *after)
	if err != nil {
		fmt.Fprintf(os.Stderr, "apicompat: %s\n", err)
		os.Exit(1)
	}

	for _, change := range changes {
		if *allChanges || change.Change == apicompat.Breaking {
			fmt.Println(change)
		}
	}
}
