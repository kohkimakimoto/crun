package main

import (
	"flag"
	"fmt"
	"github.com/Songmu/wrapcommander"
	"github.com/kohkimakimoto/crun/crun"
	"os"
	"strings"
)

func main() {
	os.Exit(realMain())
}

type stringSlice []string

func (ss *stringSlice) String() string {
	return fmt.Sprint("%v", *ss)
}
func (ss *stringSlice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

func realMain() (status int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}()

	// parse flags...
	var optVersion bool
	var optTag, optWd string
	var optEnv stringSlice

	flag.StringVar(&optTag, "t", "", "")
	flag.StringVar(&optTag, "tag", "", "")
	flag.StringVar(&optWd, "w", "", "")
	flag.StringVar(&optWd, "working-directory", "", "")
	flag.Var(&optEnv, "e", "")
	flag.Var(&optEnv, "env", "")
	flag.BoolVar(&optVersion, "v", false, "")
	flag.BoolVar(&optVersion, "version", false, "")

	flag.Usage = func() {
		fmt.Println(`Usage: ` + crun.Name + ` [OPTIONS...] <COMMAND...>

` + crun.Name + ` -- Command execution tool
version ` + crun.Version + ` (` + crun.CommitHash + `)

Copyright (c) Kohki Makimoto <kohki.makimoto@gmail.com>
The MIT License (MIT)

Options:
  -h, --help                 Show help
  -v, --version              Print the version
  -t, --tag                  Arbitrary tag of the job
  -w, --working-directory    If specified, use the given directory as working directory. 
  -e, --env                  Set custom environment variables. ex) -e KEY=VALUE
`)
	}
	flag.Parse()

	if optVersion {
		// show version
		fmt.Println(crun.Name + " version " + crun.Version + " (" + crun.CommitHash + ")")
		return 0
	}

	if len(os.Args) <= 1 {
		flag.Usage()
		return 0
	}

	c := crun.New()
	c.CommandArgs = flag.Args()
	c.Tag = optTag
	c.WorkingDirectory = optWd

	for _, e := range optEnv {
		splitString := strings.SplitN(e, "=", 2)
		if len(splitString) != 2 {
			fmt.Fprintf(os.Stderr, "invalid environment variable format '%s'. must be 'KEY=VALUE'.\n", e)
			return 1
		}
		c.Environments[splitString[0]] = splitString[1]
	}

	r, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return wrapcommander.ResolveExitCode(err)
	}

	return r.ExitCode
}
