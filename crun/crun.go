package crun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Songmu/wrapcommander"
	"github.com/kballard/go-shellquote"
	"github.com/kohkimakimoto/crun/structs"
	lua "github.com/yuin/gopher-lua"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Crun struct {
	Config                 *Config
	L                      *lua.LState
	Report                 *structs.Report
	CommandArgs            []string
	StdoutWriter           io.Writer
	StderrWriter           io.Writer
	preHandlersAreExecuted bool
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
		Command:     shellquote.Join(c.CommandArgs...),
		CommandArgs: c.CommandArgs,
		Tag:         c.Config.Tag,
		ExitCode:    -1,
		Hostname:    hostname,
	}
	c.Report = r

	if c.Config.InitByLua != ""{
		if err := c.L.DoString(c.Config.InitByLua); err != nil {
			return r, err
		}
	}

	if c.CommandArgs == nil || len(c.CommandArgs) == 0 {
		return r, errors.New("requires a command to execute")
	}

	if err := c.Config.ParseEnvironment(); err != nil {
		return r, err
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
			c.handleErrorBeforeRunning(r, err.Error())
			return r, err
		}

		logWriter = newLogWriter(f, c)
	}

	if logWriter != nil {
		c.StdoutWriter = io.MultiWriter(c.StdoutWriter, logWriter)
		c.StderrWriter = io.MultiWriter(c.StderrWriter, logWriter)
	}

	cmd := exec.Command(c.CommandArgs[0], c.CommandArgs[1:]...)
	cmd.Stdin = os.Stdin
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		c.handleErrorBeforeRunning(r, err.Error())
		return r, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stdoutPipe.Close()
		c.handleErrorBeforeRunning(r, err.Error())
		return r, err
	}

	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer
	var bufMerged bytes.Buffer

	stdoutPipe2 := io.TeeReader(stdoutPipe, io.MultiWriter(&bufStdout, &bufMerged))
	stderrPipe2 := io.TeeReader(stderrPipe, io.MultiWriter(&bufStderr, &bufMerged))

	// run pre handlers
	c.runPreHandlers(r)

	r.StartAt = now()
	if err := cmd.Start(); err != nil {
		stderrPipe.Close()
		stdoutPipe.Close()
		c.handleErrorBeforeRunning(r, err.Error())
		return r, err
	}
	if cmd.Process != nil {
		r.Pid = cmd.Process.Pid
	}

	// run notice handlers
	done := make(chan struct{})
	go func() {
		c.runNoticeHandlers(r)
		done <- struct{}{}
	}()

	outDone := make(chan struct{})
	go func() {
		defer func() {
			stdoutPipe.Close()
			outDone <- struct{}{}
		}()
		io.Copy(c.StdoutWriter, stdoutPipe2)
	}()

	errDone := make(chan struct{})
	go func() {
		defer func() {
			stderrPipe.Close()
			errDone <- struct{}{}
		}()
		io.Copy(c.StderrWriter, stderrPipe2)
	}()

	<-outDone
	<-errDone

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
		c.runFailureHandlers(r)
	} else {
		c.runSuccessHandlers(r)
	}

	// run post handlers
	c.runPostHandlers(r)

	<-done

	return r, nil
}

func (c *Crun) handleErrorBeforeRunning(r *structs.Report, errStr string) {
	r.ExitCode = -1
	r.Result = fmt.Sprintf("failed to execute command: %s", errStr)
	c.runPreHandlers(r)

	done := make(chan struct{})
	go func() {
		c.runNoticeHandlers(r)
		done <- struct{}{}
	}()

	c.runFailureHandlers(r)
	c.runPostHandlers(r)

	<-done
}

func (c *Crun) runPreHandlers(r *structs.Report) {
	if c.preHandlersAreExecuted {
		return
	}

	b, _ := json.Marshal(r)
	c.runHandlers(c.Config.PreHandlers, b, "pre")
	c.preHandlersAreExecuted = true
}

func (c *Crun) runNoticeHandlers(r *structs.Report) {
	b, _ := json.Marshal(r)
	c.runHandlers(c.Config.NoticeHandlers, b, "notice")
}

func (c *Crun) runPostHandlers(r *structs.Report) {
	b, _ := json.Marshal(r)
	c.runHandlers(c.Config.PostHandlers, b, "post")
}

func (c *Crun) runSuccessHandlers(r *structs.Report) {
	b, _ := json.Marshal(r)
	c.runHandlers(c.Config.SuccessHandlers, b, "success")
}

func (c *Crun) runFailureHandlers(r *structs.Report) {
	b, _ := json.Marshal(r)
	c.runHandlers(c.Config.FailureHandlers, b, "failure")
}

func (c *Crun) runHandlers(handlers []string, json []byte, handlerType string) {
	wg := &sync.WaitGroup{}
	for _, handler := range handlers {
		wg.Add(1)
		go func(handler string) {
			err := c.runHandler(handler, json, handlerType)
			if err != nil {
				c.StderrWriter.Write([]byte(err.Error() + "\n"))
			}
			wg.Done()
		}(handler)
	}
	wg.Wait()
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

func now() *time.Time {
	now := time.Now()
	return &now
}
