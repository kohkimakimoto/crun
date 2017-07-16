# Crun

Crun (Command-RUN) is a command execution tool.

The main feature of crun is to append hook handlers to command execution. It is useful for cron jobs to report their results. Crun is based on a fork of [Songmu/horenso](https://github.com/Songmu/horenso), and It has been heavily modified.

Table of Contents

* [Installation](#installation)
* [Usage](#usage)
  * [Example](#example)
  * [Handlers](#handlers)
  * [Execution Sequence](#execution-sequence)
  * [Result JSON](#result-json)
  * [Logging](#logging)
  * [Lua Interpreter](#lua-interpreter)
  * [Options](#options)
* [Author](#author)
* [License](#license)

## Installation

Crun is provided as a single binary. You can download it on Github releases page.

[Download latest version](https://github.com/kohkimakimoto/crun/releases/latest)

## Usage

### Example

In the following example, Crun prints command's exit code by using a post handler. Try it out!

```
$ crun --post='python -c "import sys, json; print(\"[post handler] exited with: \" + str(json.load(sys.stdin)[\"exitCode\"]))"' -- echo Helloworld!
Helloworld!
[post handler] exited with: 0
```

### Handlers

Crun runs arbitrary commands with some hook handlers. In the following example, it appends a post handler which is executed when your command finish.

```
$ crun --post /path/to/posthandler.sh -- /path/to/yourcommand [...]
```

You can use crun in a wrapper script like the following.

```bash
#!/usr/bin/env bash
crun \
  --post /path/to/post_handler.sh \
  --success /path/to/success_handler.sh \
  --failure /path/to/failure_handler.sh \
  -- "$@"
```

This `wrapper.sh` can be used in the crontab like the following.

```
3 4 * * * /path/to/wrapper.sh /path/to/job
```

I implemented some handlers. Please see [handlers](https://github.com/kohkimakimoto/crun/tree/master/handlers) directory.

### Execution Sequence

Crun supports several hook points: `pre`, `notice`, `success`, `failure` and `post`. The following table defines execution sequence:

1. Run `pre` handlers
1. Start the command
1. Run `notice` handlers (non-blocking)
1. Wait to finish the command
1. Run `success` or `failure` handlers
1. Run `post` handlers

### Result JSON

The all handlers accept a result JSON via STDIN, that reports command result like the following.

```json
{
  "command": "perl -E 'say 1;warn \"$$\\n\";'",
  "commandArgs": [
    "perl",
    "-E",
    "say 1;warn \"$$\\n\";"
  ],
  "output": "1\n95030\n",
  "stdout": "1\n",
  "stderr": "95030\n",
  "exitCode": 0,
  "result": "command exited with code: 0",
  "pid": 95030,
  "startAt": "2015-12-28T00:37:10.494282399+09:00",
  "endAt": "2015-12-28T00:37:10.546466379+09:00",
  "hostname": "webserver.example.com",
  "systemTime": 0.034632,
  "userTime": 0.026523
}
```

It is compatible with [horenso result JSON](https://github.com/Songmu/horenso#result-json).

### Logging

Crun supports logging STDOUT and STDERR to a file.

```
$ crun --log-file /var/log/file.log -- /path/to/yourcommand
```

In the default behavior, This logging functionality does not close STDOUT of the command. If you want to suppress outputting to the console, you can use `--quiet`.

```
$ crun --log-file /var/log/file.log --quiet -- /path/to/yourcommand
```

If you want to add a prefix to any lines of the outputing log. You can use `--log-prefix` option.

```
$ crun --log-file /var/log/file.log --log-prefix '[%ts] ' -- ls
```

The `%ts` is a placeholder for timestamp. The above example outputs like the following log.

```
[2017-07-04T10:17:47.373+09:00] LICENSE
[2017-07-04T10:17:47.373+09:00] Makefile
[2017-07-04T10:17:47.373+09:00] README.md
[2017-07-04T10:17:47.373+09:00] build
[2017-07-04T10:17:47.373+09:00] cmd
[2017-07-04T10:17:47.373+09:00] crun
[2017-07-04T10:17:47.373+09:00] crun.iml
[2017-07-04T10:17:47.373+09:00] glide.lock
[2017-07-04T10:17:47.373+09:00] glide.yaml
[2017-07-04T10:17:47.373+09:00] structs
[2017-07-04T10:17:47.373+09:00] vendor
```

### Lua Interpreter

You can implement crun handlers in any programming languages you like. But crun has built-in Lua interpreter to implement handlers without dependences.

#### Example

```lua
#!/bin/sh
_=[[ 
exec crun --lua "$0" "$@"
]]

local report = json.decode(io.read("*a"))
print(report.command)
```

The first four lines are trick to use [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) to run the script using crun built-in lua intepreter.

See [crun-handler-slack](https://github.com/kohkimakimoto/crun/tree/master/handlers/crun-handler-slack). It's a good example.


### Options

* `--pre`:

* `--pre-async`:

* `--notice`:

* `--notice-async`:

* `--success`:

* `--success-async`:

* `--failure`:

* `--failure-async`:

* `--post`:

* `--post-async`:

* `--stdout-file`:

* `--stderr-file`:

* `--log-file`:

* `--log-prefix`:

* `--tag`:

* `--quiet, -q`:

* `--working-directory, -w`:

* `--env, -e`:

* `--lua`:

See also: `crun -h`.

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
