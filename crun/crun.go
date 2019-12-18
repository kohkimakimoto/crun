package crun

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Songmu/wrapcommander"
	"github.com/kballard/go-shellquote"
	"github.com/kohkimakimoto/crun/structs"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Crun struct {
	Config       *Config
	L            *lua.LState
	Report       *structs.Report
	CommandArgs  []string
	StdoutWriter io.Writer
	StderrWriter io.Writer
	lockfile     *os.File
}

func New() *Crun {
	L := lua.NewState()
	openLibs(L)

	return &Crun{
		Config:       newConfig(),
		L:            L,
		StdoutWriter: os.Stdout,
		StderrWriter: os.Stderr,
	}
}

func (c *Crun) Close() {
	c.L.Close()
}

func (c *Crun) Run() (*structs.Report, error) {
	hostname, _ := os.Hostname()
	r := &structs.Report{
		Command:     c.Command(),
		CommandArgs: c.CommandArgs,
		Tag:         c.Config.Tag,
		ExitCode:    -1,
		Hostname:    hostname,
	}
	c.Report = r

	if err := c.Config.Prepare(); err != nil {
		return r, err
	}

	//if c.Config.InitByLua != "" {
	//	if err := c.L.DoString(c.Config.InitByLua); err != nil {
	//		return r, err
	//	}
	//}

	if c.CommandArgs == nil || len(c.CommandArgs) == 0 {
		return r, errors.New("requires a command to execute")
	}

	// create tmp directory
	if _, err := os.Stat(c.Config.Tmpdir); os.IsNotExist(err) {
		defaultUmask := syscall.Umask(0)
		os.MkdirAll(c.Config.Tmpdir, 0777)
		syscall.Umask(defaultUmask)
	}

	if c.Config.Quiet {
		c.StdoutWriter = ioutil.Discard
	}

	if c.Config.WorkingDirectory != "" {
		if err := os.Chdir(c.Config.WorkingDirectory); err != nil {
			return r, fmt.Errorf("couldn't change working directory to '%s': %s.", c.Config.WorkingDirectory, err.Error())
		}
	}

	for k, v := range c.Config.EnvironmentMap {
		os.Setenv(k, v)
	}

	var logWriter io.Writer
	if c.Config.LogFile != "" {
		f, err := os.OpenFile(c.Config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			c.handleErrorBeforeRunning(r, err)
			return r, err
		}

		logWriter = newLogWriter(f, c)
	}

	if logWriter != nil {
		c.StdoutWriter = io.MultiWriter(c.StdoutWriter, logWriter)
		c.StderrWriter = io.MultiWriter(c.StderrWriter, logWriter)
	}

	if c.Config.WithoutOverlapping {
		if err := c.lockForWithoutOverlapping(); err != nil {
			c.handleErrorBeforeRunning(r, err)
			return r, err
		}
		defer c.unlockForWithoutOverlapping()
	}

	cmd := exec.Command(c.CommandArgs[0], c.CommandArgs[1:]...)
	cmd.Stdin = os.Stdin
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		c.handleErrorBeforeRunning(r, err)
		return r, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stdoutPipe.Close()
		c.handleErrorBeforeRunning(r, err)
		return r, err
	}

	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer
	var bufMerged bytes.Buffer

	stdoutPipe2 := io.TeeReader(stdoutPipe, io.MultiWriter(&bufStdout, &bufMerged))
	stderrPipe2 := io.TeeReader(stderrPipe, io.MultiWriter(&bufStderr, &bufMerged))

	// run pre handlers
	if err := c.runPreHandlers(r); err != nil {
		c.handleErrorBeforeRunning(r, err)
		return r, err
	}

	r.StartAt = now()
	if err := cmd.Start(); err != nil {
		stderrPipe.Close()
		stdoutPipe.Close()
		c.handleErrorBeforeRunning(r, err)
		return r, err
	}
	if cmd.Process != nil {
		r.Pid = cmd.Process.Pid
	}

	// run notice handlers
	done := make(chan error)
	go func() {
		done <- c.runNoticeHandlers(r)
	}()

	eg := &errgroup.Group{}
	eg.Go(func() error {
		defer stdoutPipe.Close()
		_, err := io.Copy(c.StdoutWriter, stdoutPipe2)
		return err
	})

	eg.Go(func() error {
		defer stderrPipe.Close()
		_, err := io.Copy(c.StderrWriter, stderrPipe2)
		return err
	})

	if err := eg.Wait(); err != nil {
		c.handleError(err)
	}

	err = cmd.Wait()
	r.EndAt = now()
	es := wrapcommander.ResolveExitStatus(err)
	r.ExitCode = es.ExitCode()
	r.Signaled = es.Signaled()
	r.Result = fmt.Sprintf("command exited with code: %d", r.ExitCode)
	if r.Signaled {
		r.Result = fmt.Sprintf("command died with signal: %d", r.ExitCode&127)
	}
	r.Stdout = bufStdout.String()
	r.Stderr = bufStderr.String()
	r.Output = bufMerged.String()
	if p := cmd.ProcessState; p != nil {
		r.UserTime = float64(p.UserTime()) / float64(time.Second)
		r.SystemTime = float64(p.SystemTime()) / float64(time.Second)
	}

	if err != nil {
		if err := c.runFailureHandlers(r); err != nil {
			c.handleError(err)
		}
	} else {
		if err := c.runSuccessHandlers(r); err != nil {
			c.handleError(err)
		}
	}

	// run post handlers
	if err := c.runPostHandlers(r); err != nil {
		c.handleError(err)
	}

	<-done
	return r, nil
}

func (c *Crun) handleError(err error) {
	c.StderrWriter.Write([]byte(err.Error() + "\n"))
}

func (c *Crun) handleErrorBeforeRunning(r *structs.Report, err error) {
	r.ExitCode = -1
	r.Result = fmt.Sprintf("failed to execute command: %v", err)

	if err := c.runFailureHandlers(r); err != nil {
		c.handleError(err)
	}

	if err := c.runPostHandlers(r); err != nil {
		c.handleError(err)
	}
}

func (c *Crun) runPreHandlers(r *structs.Report) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.PreHandlers, b, "pre")
}

func (c *Crun) runNoticeHandlers(r *structs.Report) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.NoticeHandlers, b, "notice")
}

func (c *Crun) runPostHandlers(r *structs.Report) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.PostHandlers, b, "post")
}

func (c *Crun) runSuccessHandlers(r *structs.Report) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.SuccessHandlers, b, "success")
}

func (c *Crun) runFailureHandlers(r *structs.Report) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.FailureHandlers, b, "failure")
}

func (c *Crun) runHandlers(handlers []string, json []byte, handlerType string) error {
	eg := &errgroup.Group{}
	for _, handler := range handlers {
		h := handler
		eg.Go(func() error {
			return c.runHandler(h, json, handlerType)
		})
	}
	return eg.Wait()
}

func (c *Crun) runHandler(cmdStr string, json []byte, handlerType string) error {
	args, err := shellquote.Split(cmdStr)
	if err != nil || len(args) < 1 {
		return fmt.Errorf("invalid handler: %q", cmdStr)
	}

	// set handler type to environment
	env := os.Environ()
	env = append(env, "CRUN_HANDLER_TYPE="+handlerType)

	cmd := exec.Command(args[0], args[1:]...)
	stdinPipe, _ := cmd.StdinPipe()
	cmd.Stdout = c.StdoutWriter
	cmd.Stderr = c.StderrWriter
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		stdinPipe.Close()
		return err
	}
	stdinPipe.Write(json)
	stdinPipe.Close()

	return cmd.Wait()
}

func (c *Crun) lockForWithoutOverlapping() error {
	mutexFile := c.overlappingMutexFile()

	// create lock file and try to get the lock
	file, err := os.OpenFile(mutexFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	c.lockfile = file

	if err := flock(c.lockfile, 0644, true, 1*time.Millisecond); err != nil {
		if err == ErrTimeout {
			return fmt.Errorf("the command '%s' has already been running", c.Command())
		}
	}

	return nil
}

func (c *Crun) unlockForWithoutOverlapping() {
	if c.lockfile != nil {
		funlock(c.lockfile)
	}

	mutexFile := c.overlappingMutexFile()
	if _, err := os.Stat(mutexFile); err == nil {
		os.RemoveAll(mutexFile)
	}
}

func (c *Crun) overlappingMutexFile() string {
	return filepath.Join(c.Config.Tmpdir, c.overlappingMutexName())
}

func (c *Crun) overlappingMutexName() string {
	return fmt.Sprintf("job-%x", sha1.Sum([]byte(c.Command())))
}

func (c *Crun) Command() string {
	return shellquote.Join(c.CommandArgs...)
}

func now() *time.Time {
	now := time.Now()
	return &now
}
