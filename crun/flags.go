package crun

import (
	"github.com/urfave/cli"
)

var Flags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "pre",
		Usage: "Set pre `handler`",
	},
	cli.StringSliceFlag{
		Name:  "pre-async",
		Usage: "Set async pre `handler`",
	},
	cli.StringSliceFlag{
		Name:  "notice",
		Usage: "Set notice `handler`",
	},
	cli.StringSliceFlag{
		Name:  "notice-async",
		Usage: "Set async notice `handler`",
	},
	cli.StringSliceFlag{
		Name:  "success",
		Usage: "Set success `handler`",
	},
	cli.StringSliceFlag{
		Name:  "success-async",
		Usage: "Set async success `handler`",
	},
	cli.StringSliceFlag{
		Name:  "failure",
		Usage: "Set failure `handler`",
	},
	cli.StringSliceFlag{
		Name:  "failure-async",
		Usage: "Set async failure `handler`",
	},
	cli.StringSliceFlag{
		Name:  "post",
		Usage: "Set post `handler`",
	},
	cli.StringSliceFlag{
		Name:  "post-async",
		Usage: "Set async post `handler`",
	},
	cli.StringFlag{
		Name:  "stdout-file",
		Usage: "`logfile` to write stdout output",
	},
	cli.StringFlag{
		Name:  "stderr-file",
		Usage: "`logfile` to write stderr output",
	},
	cli.StringFlag{
		Name:  "log-file",
		Usage: "`logfile` to write merged output",
	},
	cli.StringFlag{
		Name:  "log-prefix",
		Usage: "`prefix` for the merged output log. This option is used with '--log-file' option",
	},
	cli.StringFlag{
		Name:  "tag, t",
		Usage: "Arbitrary `tag` of the job",
	},
	cli.BoolFlag{
		Name:  "quiet, q",
		Usage: "Suppress outputting to stdout",
	},
	cli.StringFlag{
		Name:  "working-directory, w",
		Usage: "`directory`. when the job runs on",
	},
	cli.StringSliceFlag{
		Name:  "env, e",
		Usage: "Set custom environment `variables`. ex) -e KEY=VALUE",
	},
	cli.BoolFlag{
		Name:  "lua",
		Usage: "Run a script by built-in Lua interpreter (for implementing handlers).",
	},
	cli.BoolFlag{
		Name:  "help, h",
		Usage: "Show help",
	},
}
