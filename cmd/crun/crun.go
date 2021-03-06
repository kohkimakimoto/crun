package main

import (
	"flag"
	"fmt"
	"github.com/Songmu/wrapcommander"
	"github.com/kohkimakimoto/crun/crun"
	"os"
)

func main() {
	os.Exit(realMain())
}

type stringSlice []string

func (ss *stringSlice) String() string {
	return fmt.Sprintf("%v", *ss)
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
	var optVersion, optQuiet, optLua, optWithoutOverlapping, optNoConfig bool
	var optTag, optWd, optLogFile, optLogPrefix, optConfigFile, optMutexdir, optMutex, optUser, optGroup string
	var optTimeout int64
	var optEnv, optPre, optNotice, optSuccess, optFailure, optPost stringSlice

	flag.StringVar(&optTag, "t", "", "")
	flag.StringVar(&optTag, "tag", "", "")
	flag.StringVar(&optWd, "w", "", "")
	flag.StringVar(&optWd, "working-directory", "", "")
	flag.StringVar(&optConfigFile, "c", "", "")
	flag.StringVar(&optConfigFile, "config-file", "", "")
	flag.StringVar(&optLogFile, "log-file", "", "")
	flag.StringVar(&optLogPrefix, "log-prefix", "", "")
	flag.StringVar(&optMutexdir, "mutexdir", "", "")
	flag.StringVar(&optMutex, "mutex", "", "")
	flag.StringVar(&optUser, "user", "", "")
	flag.StringVar(&optGroup, "group", "", "")
	flag.Var(&optEnv, "e", "")
	flag.Var(&optEnv, "env", "")
	flag.BoolVar(&optVersion, "v", false, "")
	flag.BoolVar(&optVersion, "version", false, "")
	flag.BoolVar(&optQuiet, "q", false, "")
	flag.BoolVar(&optQuiet, "quiet", false, "")
	flag.BoolVar(&optNoConfig, "n", false, "")
	flag.BoolVar(&optNoConfig, "no-config", false, "")
	flag.BoolVar(&optWithoutOverlapping, "without-overlapping", false, "")
	flag.Int64Var(&optTimeout, "timeout", 0, "")
	flag.Var(&optPre, "pre", "")
	flag.Var(&optNotice, "notice", "")
	flag.Var(&optSuccess, "success", "")
	flag.Var(&optFailure, "failure", "")
	flag.Var(&optPost, "post", "")
	// hidden flag
	flag.BoolVar(&optLua, "lua", false, "")

	flag.Usage = func() {
		fmt.Println(`Usage: ` + crun.Name + ` [OPTIONS...] <COMMAND...>

` + crun.Name + ` -- Command execution wrapper.
version ` + crun.Version + ` (` + crun.CommitHash + `)

Copyright (c) Kohki Makimoto <kohki.makimoto@gmail.com>
The MIT License (MIT)

Options:
  (General)
  -c, --config-file <path>         Load config from the file.
  -n, --no-config                  No config file will be used
  -t, --tag <string>               Set a tag of the job.
  -w, --working-directory <dir>    If specified, use the given directory as working directory.
  -e, --env <KEY=VALUE>            Set custom environment variables. ex) -e KEY=VALUE
  --user <user>                    Set an execution user
  --group <group>                  Set an execution group

  (Hook Handlers)
  --pre <handler>                  Set a pre handler. This option can be set multi time.
  --notice <handler>               Set a notice handler. This option can be set multi time.
  --success <handler>              Set a success handler. This option can be set multi time.
  --failure <handler>              Set a failure handler. This option can be set multi time.
  --post <handler>                 Set a post handler. This option can be set multi time.

  (Logging)
  --log-file <path>                The file path to write merged output. The strftime format like '%Y%m%d.log' is available.
  --log-prefix <string>            The prefix for the merged output log. This option is used with '--log-file' option.
  -q, --quiet                      Suppress outputting to stdout.

  (Overlapping)
  --without-overlapping            Prevent overlapping execution the job.
  --mutexdir <dir>                 The directory path to store job mutex files. (default: /tmp/crun)
  --mutex <string>                 Overriding the mutex id.

  (Timeout)
  --timeout <number>               The command is terminated when the timeout elapses. The unit is second.

  (Help)
  -h, --help                       Show help.
  -v, --version                    Print the version.
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

	if optLua {
		// run lua mode for extension script.
		if flag.NArg() == 0 {
			flag.Usage()
			return 0
		}

		lapp := crun.NewLuaApp()
		if err := lapp.Run(flag.Args()); err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return 1
		}

		return 0
	}

	c := crun.New()
	defer c.Close()

	if !optNoConfig {
		if err := loadConfigFile(c, optConfigFile); err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return 1
		}
	}

	c.CommandArgs = flag.Args()
	if optTag != "" {
		c.Config.Tag = optTag
	}
	if optWd != "" {
		c.Config.WorkingDirectory = optWd
	}
	if len(optPre) > 0 {
		c.Config.PreHandlers = append(c.Config.PreHandlers, optPre...)
	}
	if len(optNotice) > 0 {
		c.Config.NoticeHandlers = append(c.Config.NoticeHandlers, optNotice...)
	}
	if len(optSuccess) > 0 {
		c.Config.SuccessHandlers = append(c.Config.SuccessHandlers, optSuccess...)
	}
	if len(optFailure) > 0 {
		c.Config.FailureHandlers = append(c.Config.FailureHandlers, optFailure...)
	}
	if len(optPost) > 0 {
		c.Config.PostHandlers = append(c.Config.PostHandlers, optPost...)
	}
	if optLogFile != "" {
		c.Config.LogFile = optLogFile
	}
	if optLogPrefix != "" {
		c.Config.LogPrefix = optLogPrefix
	}
	if optQuiet {
		c.Config.Quiet = optQuiet
	}
	if optWithoutOverlapping {
		c.Config.WithoutOverlapping = optWithoutOverlapping
	}
	if optMutexdir != "" {
		c.Config.Mutexdir = optMutexdir
	}
	if optMutex != "" {
		c.Config.Mutex = optMutex
	}
	if optUser != "" {
		c.Config.User = optUser
	}
	if optGroup != "" {
		c.Config.Group = optGroup
	}
	if len(optEnv) > 0 {
		c.Config.Environment = append(c.Config.Environment, optEnv...)
	}
	if optTimeout > 0 {
		c.Config.Timeout = optTimeout
	}

	r, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return wrapcommander.ResolveExitCode(err)
	}

	return r.ExitCode
}

func loadConfigFile(c *crun.Crun, optConfigFile string) error {
	if optConfigFile != "" {
		if err := c.Config.LoadConfigFile(optConfigFile); err != nil {
			return fmt.Errorf("failed to open file: %s %v", optConfigFile, err)
		}
	}

	return nil
}
