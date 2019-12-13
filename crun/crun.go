package crun

import (
	"errors"
	"fmt"
	"github.com/Songmu/wrapcommander"
	"github.com/kballard/go-shellquote"
	"github.com/kohkimakimoto/crun/structs"
	"os"
	"os/exec"
	"time"
)

type Crun struct {
	CommandArgs   []string
	Tag    string
	Report *structs.Report
}

func New() *Crun {
	return &Crun{}
}

func (c *Crun) Run() (*structs.Report, error) {
	hostname, _ := os.Hostname()
	r := &structs.Report{
		Command:     shellquote.Join(c.CommandArgs...),
		CommandArgs: c.CommandArgs,
		Tag:         c.Tag,
		ExitCode:    -1,
		Hostname:    hostname,
	}
	c.Report = r

	if c.CommandArgs == nil || len(c.CommandArgs) == 0 {
		return r, errors.New("requires a command to execute")
	}

	cmd := exec.Command(c.CommandArgs[0], c.CommandArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr= os.Stderr

	if err := cmd.Start(); err != nil {
		return r, err
	}

	err := cmd.Wait()
	r.EndAt = now()
	es := wrapcommander.ResolveExitStatus(err)
	r.ExitCode = es.ExitCode()
	r.Signaled = es.Signaled()
	r.Result = fmt.Sprintf("command exited with code: %d", r.ExitCode)
	if r.Signaled {
		r.Result = fmt.Sprintf("command died with signal: %d", r.ExitCode&127)
	}

	if p := cmd.ProcessState; p != nil {
		r.UserTime = float64(p.UserTime()) / float64(time.Second)
		r.SystemTime = float64(p.SystemTime()) / float64(time.Second)
	}

	return r, nil
}

func now() *time.Time {
	now := time.Now()
	return &now
}
