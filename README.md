# Crun <!-- omit in toc -->

Crun (Command-RUN) is a command execution wrapper. It is a simple command that is used with another command.
Crun provides several useful features for your command execution.

* **Hooks** 
* **Logging**
* **Preventing Overlaps**
* **Environment Variables**

Crun is based on a fork of [Songmu/horenso](https://github.com/Songmu/horenso), and It has been heavily modified.

## Table of Contents <!-- omit in toc -->

- [Installation](#installation)
- [Usage](#usage)
  - [Handlers](#handlers)
  - [Result JSON](#result-json)
  - [Execution Sequence](#execution-sequence)
  - [Logging](#logging)
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

## Usage

In the following example, Crun prints command's exit code by using a post handler. Try it out!

```
$ crun --post='python -c "import sys, json; print(\"post handler detected the command exited with: \" + str(json.load(sys.stdin)[\"exitCode\"]))"' -- echo Helloworld!
Helloworld!
post handler detected the command exited with: 0
```

### Handlers

Crun runs arbitrary commands with some hook handlers. In the following example, it appends a post handler which is executed when your command finish.

```
$ crun --post /path/to/posthandler.sh -- /path/to/yourcommand [...]
```

I implemented some handlers. Please see [handlers](https://github.com/kohkimakimoto/crun/tree/master/handlers) directory.

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

### Execution Sequence

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

### Preventing Overlaps

If you use `--without-overlapping`, Crun prevent to overlap the command execution.

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
post = [
  "/path/to/posthandler",
]
environment = [
  "KEY=VALUE"
]
log_file = "/path/to/logfile.log"
```

You can use the config file like the following:

```
$ crun -c /path/to/config.toml -- /path/to/yourcommand [...]
```

## Lua Interpreter

You can implement Crun handlers in any programming languages you like. But crun has built-in Lua interpreter to implement handlers without dependences.

### Example

```lua
#!/bin/sh
_=[[ 
exec crun --lua "$0" "$@"
]]

local report = json.decode(io.read("*a"))
print(report.command)
```

The first four lines are trick to use [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)) to run the script using Crun built-in lua intepreter.

See [crun-handler-slack](https://github.com/kohkimakimoto/crun/tree/master/handlers/crun-handler-slack). It's a good example.

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
