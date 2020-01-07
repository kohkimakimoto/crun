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
	"github.com/lestrrat-go/strftime"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

type Crun struct {
	Config       *Config
	Report       *structs.Report
	CommandArgs  []string
	StdoutWriter io.Writer
	StderrWriter io.Writer
	lockfile     *os.File
}

func New() *Crun {
	return &Crun{
		Config:       newConfig(),
		StdoutWriter: os.Stdout,
		StderrWriter: os.Stderr,
	}
}

func (c *Crun) Close() {
	// nothing to do now...
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

	if c.CommandArgs == nil || len(c.CommandArgs) == 0 {
		return r, errors.New("requires a command to execute")
	}

	if err := c.Config.Prepare(); err != nil {
		return r, err
	}

	// create mutex directory
	if _, err := os.Stat(c.Config.Mutexdir); os.IsNotExist(err) {
		defaultUmask := syscall.Umask(0)
		if err := os.MkdirAll(c.Config.Mutexdir, 0777); err != nil {
			return c.handleErrorBeforeRunning(r, err, nil)
		}
		syscall.Umask(defaultUmask)
	}

	if c.Config.Quiet {
		c.StdoutWriter = ioutil.Discard
	}

	for k, v := range c.Config.EnvironmentMap {
		os.Setenv(k, v)
	}

	var logWriter io.Writer
	if c.Config.LogFile != "" {
		logfile, err := strftime.Format(c.Config.LogFile, time.Now())
		if err != nil {
			return c.handleErrorBeforeRunning(r, err, nil)
		}

		f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return c.handleErrorBeforeRunning(r, err, nil)
		}

		logWriter = newLogWriter(f, c)
	}

	if logWriter != nil {
		c.StdoutWriter = io.MultiWriter(c.StdoutWriter, logWriter)
		c.StderrWriter = io.MultiWriter(c.StderrWriter, logWriter)
	}

	if c.Config.WithoutOverlapping {
		if err := c.lockForWithoutOverlapping(); err != nil {
			return c.handleErrorBeforeRunning(r, err, []string{"CRUN_OVERLAPPING=1"})
		}
		defer c.unlockForWithoutOverlapping()
	}

	cmd := exec.Command(c.CommandArgs[0], c.CommandArgs[1:]...)
	cmd.Stdin = os.Stdin

	if os.Getuid() == 0 {
		uid, gid, err := c.getUidAndGid()
		if err != nil {
			return c.handleErrorBeforeRunning(r, err, nil)
		}

		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	}

	if c.Config.WorkingDirectory != "" {
		cmd.Dir = c.Config.WorkingDirectory
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return c.handleErrorBeforeRunning(r, err, nil)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stdoutPipe.Close()
		return c.handleErrorBeforeRunning(r, err, nil)
	}

	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer
	var bufMerged bytes.Buffer

	stdoutPipe2 := io.TeeReader(stdoutPipe, io.MultiWriter(&bufStdout, &bufMerged))
	stderrPipe2 := io.TeeReader(stderrPipe, io.MultiWriter(&bufStderr, &bufMerged))

	// run pre handlers
	if err := c.runPreHandlers(r, nil); err != nil {
		return c.handleErrorBeforeRunning(r, err, nil)
	}

	r.StartAt = now()
	if err := cmd.Start(); err != nil {
		stderrPipe.Close()
		stdoutPipe.Close()
		return c.handleErrorBeforeRunning(r, err, nil)
	}
	if cmd.Process != nil {
		r.Pid = cmd.Process.Pid
	}

	// run notice handlers
	noticeHandlersDone := make(chan error)
	go func() {
		noticeHandlersDone <- c.runNoticeHandlers(r, nil)
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

	envForHandler := []string{}
	if c.Config.Timeout > 0 {
		done := make(chan error)
		go func() {
			if err := eg.Wait(); err != nil {
				c.handleError(err)
			}
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(time.Duration(c.Config.Timeout) * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				c.handleError(fmt.Errorf("failed to kill: " + err.Error()))
			}
			err = fmt.Errorf("crun terminated the command. it took time over %d sec", c.Config.Timeout)
			c.handleError(err)

			envForHandler = append(envForHandler, fmt.Sprintf("CRUN_TIMEOUT=%d", c.Config.Timeout))
		case err = <-done:
			defer close(done)
		}
	} else {
		if err := eg.Wait(); err != nil {
			c.handleError(err)
		}
		err = cmd.Wait()
	}

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
		if err := c.runFailureHandlers(r, nil); err != nil {
			c.handleError(err)
		}
	} else {
		if err := c.runSuccessHandlers(r, nil); err != nil {
			c.handleError(err)
		}
	}

	// run post handlers
	if err := c.runPostHandlers(r, nil); err != nil {
		c.handleError(err)
	}

	<-noticeHandlersDone
	return r, nil
}

func (c *Crun) getUidAndGid() (int, int, error) {
	var uid, gid int
	if c.Config.User != "" {
		u, err := LookupUserStruct(c.Config.User)
		if err != nil {
			return -1, -1, err
		}

		uid, err = strconv.Atoi(u.Uid)
		if err != nil {
			return -1, -1, err
		}

		gid, err = strconv.Atoi(u.Gid)
		if err != nil {
			return -1, -1, err
		}
	} else {
		uid = os.Getuid()
		gid = os.Getgid()
	}

	if c.Config.Group != "" {
		id, err := LookupGroup(c.Config.Group)
		if err != nil {
			return -1, -1, err
		}
		gid = id
	}

	return uid, gid, nil
}

func (c *Crun) handleError(err error) {
	c.StderrWriter.Write([]byte(err.Error() + "\n"))
}

func (c *Crun) handleErrorBeforeRunning(r *structs.Report, err error, customEnv []string) (*structs.Report, error) {
	r.ExitCode = -1
	r.Result = err.Error()
	if err := c.runFailureHandlers(r, customEnv); err != nil {
		c.handleError(err)
	}
	if err := c.runPostHandlers(r, customEnv); err != nil {
		c.handleError(err)
	}

	return r, err
}

func (c *Crun) runPreHandlers(r *structs.Report, customEnv []string) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.PreHandlers, b, "pre", customEnv)
}

func (c *Crun) runNoticeHandlers(r *structs.Report, customEnv []string) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.NoticeHandlers, b, "notice", customEnv)
}

func (c *Crun) runPostHandlers(r *structs.Report, customEnv []string) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.PostHandlers, b, "post", customEnv)
}

func (c *Crun) runSuccessHandlers(r *structs.Report, customEnv []string) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.SuccessHandlers, b, "success", customEnv)
}

func (c *Crun) runFailureHandlers(r *structs.Report, customEnv []string) error {
	b, _ := json.Marshal(r)
	return c.runHandlers(c.Config.FailureHandlers, b, "failure", customEnv)
}

func (c *Crun) runHandlers(handlers []string, json []byte, handlerType string, customEnv []string) error {
	eg := &errgroup.Group{}
	for _, handler := range handlers {
		h := handler
		eg.Go(func() error {
			return c.runHandler(h, json, handlerType, customEnv)
		})
	}
	return eg.Wait()
}

func (c *Crun) runHandler(cmdStr string, json []byte, handlerType string, customEnv []string) error {
	args, err := shellquote.Split(cmdStr)
	if err != nil || len(args) < 1 {
		return fmt.Errorf("invalid handler: %q", cmdStr)
	}

	// set handler type to environment
	env := os.Environ()
	env = append(env, "CRUN_HANDLER_TYPE="+handlerType)

	if customEnv != nil {
		for _, ce := range customEnv {
			env = append(env, ce)
		}
	}

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
			return fmt.Errorf("failed to run the command, because '%s' has already been running", c.Command())
		}
	}

	return nil
}

func (c *Crun) unlockForWithoutOverlapping() {
	if c.lockfile != nil {
		funlock(c.lockfile)
	}
}

func (c *Crun) overlappingMutexFile() string {
	return filepath.Join(c.Config.Mutexdir, c.overlappingMutexName())
}

func (c *Crun) overlappingMutexName() string {
	mutex := c.Config.Mutex
	if mutex == "" {
		mutex = fmt.Sprintf("%x", sha1.Sum([]byte(c.Command())))
	}
	return fmt.Sprintf("crun-mutex-%s", mutex)
}

func (c *Crun) Command() string {
	return shellquote.Join(c.CommandArgs...)
}

func now() *time.Time {
	now := time.Now()
	return &now
}
