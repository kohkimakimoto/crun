package crun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/Songmu/wrapcommander"
	"github.com/kballard/go-shellquote"
	"github.com/kohkimakimoto/crun/structs"
	"sync"
)

//
// Many code in this file inspired by https://github.com/Songmu/horenso
//
// -----------------------------------------------------------------------
// https://github.com/Songmu/horenso
//
// Copyright (c) 2015 Songmu
//
// MIT License
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
// -----------------------------------------------------------------------

type Job struct {
	CommandArgs          []string
	NoticeHandlers       []string
	NoticeAsyncHandlers  []string
	PreHandlers          []string
	PreAsyncHandlers     []string
	PostHandlers         []string
	PostAsyncHandlers    []string
	SuccessHandlers      []string
	SuccessAsyncHandlers []string
	FailureHandlers      []string
	FailureAsyncHandlers []string
	StdoutFile           string
	StderrFile           string
	LogFile              string
	LogTimestamp         bool
	LogPrefix            string
	Tag                  string
	Quiet                bool
	WorkingDirectory     string
	StdoutWriter         io.Writer
	StderrWriter         io.Writer
	Report               *structs.Report
	Environments         map[string]string
}

func NewJob() *Job {
	return &Job{
		CommandArgs:     []string{},
		NoticeHandlers:  []string{},
		PreHandlers:     []string{},
		PostHandlers:    []string{},
		SuccessHandlers: []string{},
		FailureHandlers: []string{},
		StdoutWriter:    os.Stdout,
		StderrWriter:    os.Stderr,
		Environments:    map[string]string{},
	}
}

// Run runs a job.
// The error only returns the crun core error.
// If a job finished with error. Run does NOT return a error. Return nil.
func (job *Job) Run() error {
	args := job.CommandArgs

	if args == nil || len(args) == 0 {
		return errors.New("there is no command to execute.")
	}

	hostname, _ := os.Hostname()
	r := structs.Report{
		Command:     shellquote.Join(args...),
		CommandArgs: args,
		Tag:         job.Tag,
		Hostname:    hostname,
	}
	job.Report = &r

	if job.StdoutFile != "" && job.StdoutFile == job.StderrFile {
		return errors.New("Couldn't set the same file to '--stderr-file' and `--stdout-file`. You should use '--log-file'.")
	}

	if job.Quiet {
		job.StdoutWriter = ioutil.Discard
	}

	if job.WorkingDirectory != "" {
		if err := os.Chdir(job.WorkingDirectory); err != nil {
			return fmt.Errorf("couldn't change working directory to '%s': %s.", job.WorkingDirectory, err.Error())
		}
	}

	var logWriter io.Writer
	if job.LogFile != "" {
		f, err := os.OpenFile(job.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			job.handleErrorBeforeRunning(r, err.Error())
			return err
		}

		if job.LogPrefix != "" {
			logWriter = newPrefixWriter(f, job)
		} else {
			logWriter = f
		}
	}

	if job.StdoutFile != "" {
		f, err := os.OpenFile(job.StdoutFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			job.handleErrorBeforeRunning(r, err.Error())
			return err
		}

		if job.Quiet {
			// only outout to the file
			if logWriter != nil {
				job.StdoutWriter = io.MultiWriter(f, logWriter)
			} else {
				job.StdoutWriter = f
			}
		} else {
			if logWriter != nil {
				job.StdoutWriter = io.MultiWriter(os.Stdout, f, logWriter)
			} else {
				job.StdoutWriter = io.MultiWriter(os.Stdout, f)
			}
		}
	} else if logWriter != nil {
		if job.Quiet {
			// only output to the file
			job.StdoutWriter = logWriter
		} else {
			job.StdoutWriter = io.MultiWriter(os.Stdout, logWriter)
		}
	}

	if job.StderrFile != "" {
		f, err := os.OpenFile(job.StderrFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			job.handleErrorBeforeRunning(r, err.Error())
			return err
		}

		if job.Quiet {
			if logWriter != nil {
				job.StderrWriter = io.MultiWriter(f, logWriter)
			} else {
				job.StderrWriter = f
			}
		} else {
			if logWriter != nil {
				job.StdoutWriter = io.MultiWriter(os.Stderr, f, logWriter)
			} else {
				job.StderrWriter = io.MultiWriter(os.Stderr, f)
			}
		}
	} else if logWriter != nil {
		if job.Quiet {
			// only outout to the file
			job.StderrWriter = logWriter
		} else {
			job.StderrWriter = io.MultiWriter(os.Stderr, logWriter)
		}
	}

	for k, v := range job.Environments {
		os.Setenv(k, v)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		job.handleErrorBeforeRunning(r, err.Error())
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stdoutPipe.Close()
		job.handleErrorBeforeRunning(r, err.Error())
		return err
	}

	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer
	var bufMerged bytes.Buffer

	stdoutPipe2 := io.TeeReader(stdoutPipe, io.MultiWriter(&bufStdout, &bufMerged))
	stderrPipe2 := io.TeeReader(stderrPipe, io.MultiWriter(&bufStderr, &bufMerged))

	// run pre handlers
	job.runPreAsyncHandlers(r)
	job.runPreHandlers(r)

	r.StartAt = now()
	err = cmd.Start()
	if err != nil {
		stderrPipe.Close()
		stdoutPipe.Close()
		job.handleErrorBeforeRunning(r, err.Error())

		return err
	}
	if cmd.Process != nil {
		r.Pid = &cmd.Process.Pid
	}

	// run notice handlers
	done := make(chan struct{})
	go func() {
		job.runNoticeAsyncHandlers(r)
		job.runNoticeHandlers(r)
		done <- struct{}{}
	}()

	outDone := make(chan struct{})
	go func() {
		defer func() {
			stdoutPipe.Close()
			outDone <- struct{}{}
		}()
		io.Copy(job.StdoutWriter, stdoutPipe2)
	}()

	errDone := make(chan struct{})
	go func() {
		defer func() {
			stderrPipe.Close()
			errDone <- struct{}{}
		}()
		io.Copy(job.StderrWriter, stderrPipe2)
	}()

	<-outDone
	<-errDone
	err = cmd.Wait()
	r.EndAt = now()
	ex := wrapcommander.ResolveExitCode(err)
	r.ExitCode = &ex
	r.Result = fmt.Sprintf("command exited with code: %d", *r.ExitCode)
	if *r.ExitCode > 128 {
		r.Result = fmt.Sprintf("command died with signal: %d", *r.ExitCode&127)
	}
	r.Stdout = bufStdout.String()
	r.Stderr = bufStderr.String()
	r.Output = bufMerged.String()
	if p := cmd.ProcessState; p != nil {
		durPtr := func(t time.Duration) *float64 {
			f := float64(t) / float64(time.Second)
			return &f
		}
		r.UserTime = durPtr(p.UserTime())
		r.SystemTime = durPtr(p.SystemTime())
	}

	if err != nil {
		job.runFailureAsyncHandlers(r)
		job.runFailureHandlers(r)
	} else {
		job.runSuccessAsyncHandlers(r)
		job.runSuccessHandlers(r)
	}

	// run post handlers
	job.runPostAsyncHandlers(r)
	job.runPostHandlers(r)

	<-done

	return nil
}

func (job *Job) handleErrorBeforeRunning(r structs.Report, errStr string) {
	fail := -1
	r.ExitCode = &fail
	r.Result = fmt.Sprintf("failed to execute command: %s", errStr)
	job.runPreAsyncHandlers(r)
	job.runPreHandlers(r)

	done := make(chan struct{})
	go func() {
		job.runNoticeAsyncHandlers(r)
		job.runNoticeHandlers(r)
		done <- struct{}{}
	}()

	job.runFailureAsyncHandlers(r)
	job.runFailureHandlers(r)
	job.runPostAsyncHandlers(r)
	job.runPostHandlers(r)

	<-done
}

func (job *Job) runNoticeAsyncHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runAsyncHandlers(job.NoticeAsyncHandlers, json, "notice")
}

func (job *Job) runNoticeHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runHandlers(job.NoticeHandlers, json, "notice")
}

func (job *Job) runPreAsyncHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runAsyncHandlers(job.PreAsyncHandlers, json, "pre")
}

func (job *Job) runPreHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runHandlers(job.PreHandlers, json, "pre")
}

func (job *Job) runPostAsyncHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runAsyncHandlers(job.PostAsyncHandlers, json, "post")
}

func (job *Job) runPostHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runHandlers(job.PostHandlers, json, "post")
}

func (job *Job) runSuccessAsyncHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runAsyncHandlers(job.SuccessAsyncHandlers, json, "success")
}

func (job *Job) runSuccessHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runHandlers(job.SuccessHandlers, json, "success")
}

func (job *Job) runFailureAsyncHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runAsyncHandlers(job.FailureAsyncHandlers, json, "failure")
}

func (job *Job) runFailureHandlers(r structs.Report) {
	json, _ := json.Marshal(r)
	job.runHandlers(job.FailureHandlers, json, "failure")
}

func (job *Job) runHandlers(handlers []string, json []byte, handlerType string) {
	for _, handler := range handlers {
		err := job.runHandler(handler, json, handlerType)
		if err != nil {
			job.StderrWriter.Write([]byte(err.Error() + "\n"))
		}
	}
}

func (job *Job) runAsyncHandlers(handlers []string, json []byte, handlerType string) {
	wg := &sync.WaitGroup{}
	for _, handler := range handlers {
		wg.Add(1)
		go func(handler string) {
			err := job.runHandler(handler, json, handlerType)
			if err != nil {
				job.StderrWriter.Write([]byte(err.Error() + "\n"))
			}
			wg.Done()
		}(handler)
	}

	wg.Wait()
}

func (job *Job) runHandler(cmdStr string, json []byte, handlerType string) error {
	args, err := shellquote.Split(cmdStr)
	if err != nil || len(args) < 1 {
		return fmt.Errorf("invalid handler: %q", cmdStr)
	}

	// set handler type to environment
	env := os.Environ()
	env = append(env, "CRUN_HANDLER_TYPE="+handlerType)

	cmd := exec.Command(args[0], args[1:]...)
	stdinPipe, _ := cmd.StdinPipe()
	cmd.Stdout = job.StdoutWriter
	cmd.Stderr = job.StderrWriter
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
