# Crun <!-- omit in toc -->

Crun (Command-RUN) is a command execution tool.

The main feature of crun is to append hook handlers to command execution. It is useful for cron jobs to report their results. Crun is based on a fork of [Songmu/horenso](https://github.com/Songmu/horenso), and It has been heavily modified.

## Table of Contents <!-- omit in toc -->

## Installation

Crun is provided as a single binary. You can download it on Github releases page.

[Download latest version](https://github.com/kohkimakimoto/crun/releases/latest)

If you use CentOS7, you can also use RPM package that is stored in the same releases page.

## Usage

### Example

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

### Execution Sequence

Crun supports several hook points: `pre`, `notice`, `success`, `failure` and `post`. The following table defines execution sequence:

1. Run `pre` handlers. If they finish with error, it skips running the command and `notice` handlers.
2. Start the command
3. Run `notice` handlers (non-blocking)
4. Wait to finish the command
5. Run `success` or `failure` handlers
6. Run `post` handlers

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
