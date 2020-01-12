# Crun <!-- omit in toc -->

Crun (Command-RUN) is a command execution wrapper. It is a simple command that is used with another command.
Crun provides several useful features for your command execution as the following.

* **Hook Handlers**: You can run arbitrary scripts before and after a command execution. It is useful for notifications.
* **Logging**: Crun supports logging STDOUT and STDERR to a file.
* **Timeout**: Crun terminates the command when the timeout elapses.
* **Preventing Overlaps**: Crun prevents to overlap the command execution.
* **Environment Variables**: You can specify the environment variables.

Crun is based on a fork of [Songmu/horenso](https://github.com/Songmu/horenso), and It has been heavily modified.

## Table of Contents <!-- omit in toc -->

- [Installation](#installation)
- [Options](#options)
- [Usage](#usage)
  - [Hook Handlers](#hook-handlers)
    - [Result JSON](#result-json)
    - [Execution Sequence](#execution-sequence)
  - [Logging](#logging)
  - [Timeout](#timeout)
  - [Preventing Overlaps](#preventing-overlaps)
  - [Environment Variables](#environment-variables)
- [Config](#config)
- [Lua Interpreter](#lua-interpreter)
  - [Example](#example)
- [Author](#author)
- [License](#license)

## Installation

Crun is provided as a single binary. You can download it on Github releases page.

[Download latest version](https://github.com/kohkimakimoto/crun/releases/latest)

If you use CentOS7, you can also use RPM package that is stored in the same releases page.

## Options

You can see the all command line options by executing `crun -h`:

```
Usage: crun [OPTIONS...] <COMMAND...>

crun -- Command execution wrapper.
version 0.8.0 (a21875bc6deb21e0f006b2e999504b173af51397)

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
```


## Usage

In the following example, Crun prints command's exit code by using a post handler. Try it out!

```
$ crun --post='python -c "import sys, json; print(\"post handler detected the command exited with: \" + str(json.load(sys.stdin)[\"exitCode\"]))"' -- echo Helloworld!
Helloworld!
post handler detected the command exited with: 0
```

### Hook Handlers

Crun runs arbitrary scripts with some hook handlers. In the following example, it appends a post handler which is executed when your command finishes.

```
$ crun --post /path/to/posthandler.sh -- /path/to/yourcommand [...]
```

I implemented some handlers. Please see [handlers](https://github.com/kohkimakimoto/crun/tree/master/handlers) directory.

#### Result JSON

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
  "signaled": false,
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

#### Execution Sequence

Crun supports several hook points: `pre`, `notice`, `success`, `failure` and `post`. The following table defines execution sequence:

1. Run `pre` handlers.
2. Start the command
3. Run `notice` handlers (non-blocking)
4. Wait to finish the command
5. Run `success` or `failure` handlers
6. Run `post` handlers

### Logging

Crun supports logging STDOUT and STDERR to a file.

```
$ crun --log-file /var/log/file.log -- /path/to/yourcommand
```

You can also add a prefix to the log lines by using `--log-prefix` option.

```
$ crun --log-file /var/log/file.log --log--prefix "%time %tag %pid: " -- /path/to/yourcommand
```

In the log prefix string, you can use the following replacement:

* `%time`: Timestamp.
* `%tag`: The tag that is specified by `--tag` option.
* `%pid`: The process id.

### Timeout

If you use `--timeout` option, Crun terminates the command when the timeout elapses.

```
$ crun --timeout 10 -- /path/to/yourcommand
```

### Preventing Overlaps

If you use `--without-overlapping`, Crun prevents to overlap the command execution.

```
$ crun --without-overlapping -- /path/to/yourcommand [...]
```

### Environment Variables

You can specify the environment variables with such as `KEY=VALUE` format.

```
$ crun -e "KEY=VALUE" -- /path/to/yourcommand [...]
```

## Config

Instead of specifying command line options, You can use config file with `-c` option.

Example:

```toml
pre = []

notice = []

success = []

failure = []

post = [
  "/path/to/posthandler",
]

environment = [
  "KEY=VALUE"
]

log_file = "/path/to/logfile.log"

log_prefix = "%time %tag %pid: "
```

You can use the config file like the following:

```
$ crun -c /path/to/config.toml -- /path/to/yourcommand [...]
```

## Lua Interpreter

You can implement Crun handlers in any programming languages you like. But crun has a built-in Lua interpreter to implement handlers without dependences.

### Example

```lua
#!/bin/sh
_=[[
exec crun --lua "$0" "$@"
]]
local json = require "json"
local report = json.decode(io.read("*a"))
print(report.command)
```

The first four lines code is a trick to use [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) to run the script using Crun's Lua.

See [crun-handler-slack](https://github.com/kohkimakimoto/crun/tree/master/handlers/crun-handler-slack). It's a good example.

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
