package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kohkimakimoto/crun/crun"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli"
)

func main() {
	os.Exit(realMain())
}

func realMain() (status int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}()

	app := cli.NewApp()
	app.Name = crun.Name
	app.Version = crun.Version + " (" + crun.CommitHash + ")"
	app.Usage = "Command execution tool."
	app.HideHelp = true
	app.Flags = crun.Flags

	app.Action = func(ctx *cli.Context) error {
		if ctx.Bool("help") {
			// show help
			cli.ShowAppHelp(ctx)
			return nil
		}

		if ctx.Bool("lua") {
			// run Lua script mode for extension script.
			if ctx.NArg() == 0 {
				cli.ShowAppHelp(ctx)
				return nil
			}

			L := crun.NewLuaProcess()
			L.ScriptFile = ctx.Args().First()
			L.X = ctx.Bool("x")

			err := L.Run(ctx.Args())
			if err != nil {
				return err
			}

			return nil
		}

		args := ctx.Args()
		if !args.Present() {
			cli.ShowAppHelp(ctx)
			return nil
		}

		commandArgs := []string{}
		for _, arg := range args {
			ss, err := shellwords.Parse(arg)
			if err != nil {
				return err
			}

			commandArgs = append(commandArgs, ss...)
		}

		job := crun.NewJob()
		job.NoticeHandlers = ctx.StringSlice("notice")
		job.NoticeAsyncHandlers = ctx.StringSlice("notice-async")
		job.PreHandlers = ctx.StringSlice("pre")
		job.PreAsyncHandlers = ctx.StringSlice("pre-async")
		job.PostHandlers = ctx.StringSlice("post")
		job.PostAsyncHandlers = ctx.StringSlice("post-async")
		job.SuccessHandlers = ctx.StringSlice("success")
		job.SuccessAsyncHandlers = ctx.StringSlice("success-async")
		job.FailureHandlers = ctx.StringSlice("failure")
		job.FailureAsyncHandlers = ctx.StringSlice("failure-async")
		job.StdoutFile = ctx.String("stdout-file")
		job.StderrFile = ctx.String("stderr-file")
		job.LogFile = ctx.String("log-file")
		job.LogPrefix = ctx.String("log-prefix")
		job.Tag = ctx.String("tag")
		job.Quiet = ctx.Bool("quiet")
		job.WorkingDirectory = ctx.String("working-directory")
		job.CommandArgs = commandArgs

		for _, e := range ctx.StringSlice("env") {
			splitString := strings.SplitN(e, "=", 2)
			if len(splitString) != 2 {
				return fmt.Errorf("Invalid environment variable format '%s'. You have to use 'KEY=VALUE' format.", e)
			}

			job.Environments[splitString[0]] = splitString[1]
		}

		err := job.Run()
		if err != nil {
			return err
		}

		if job.Report != nil && job.Report.ExitCode != nil {
			status = *job.Report.ExitCode
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		status = 1
	}

	return status
}

func init() {
	cli.AppHelpTemplate = `Usage: {{.Name}}{{if .VisibleFlags}} [<options...>]{{end}} <command>

{{.Name}}{{if .Usage}} -- {{.Usage}}{{end}}{{if .Version}}
version {{.Version}}{{end}}{{if .Flags}}

Copyright (c) Kohki Makimoto <kohki.makimoto@gmail.com>
The MIT License (MIT)

Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}{{if .VisibleCommands}}
Commands:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{end}}
`
	cli.CommandHelpTemplate = `Usage: {{.Name}}{{if .VisibleFlags}} [<options...>]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[<arguments...>]{{end}}
{{if .Usage}}
{{.Usage}}{{end}}
{{if .VisibleFlags}}
Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}{{if .Description}}
Description:
  {{.Description}}
{{end}}
`

	cli.SubcommandHelpTemplate = `USAGE: {{.Name}} <command>{{if .VisibleFlags}} [<options...>]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[<arguments...>]{{end}}
{{if .Usage}}
{{.Usage}}{{end}}

Commands:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
  {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
`
}
