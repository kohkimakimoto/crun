package structs

import "time"

// Report is represents the result of the command
//
// this file inspired by https://github.com/Songmu/horenso
//
// -----------------------------------------------------------------------
// https://github.com/Songmu/horenso
// Copyright (c) 2017 Songmu
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

type Report struct {
	Command     string     `json:"command"`
	CommandArgs []string   `json:"commandArgs"`
	Tag         string     `json:"tag,omitempty"`
	Output      string     `json:"output"`
	Stdout      string     `json:"stdout"`
	Stderr      string     `json:"stderr"`
	ExitCode    int        `json:"exitCode"`
	Signaled    bool       `json:"signaled"`
	Result      string     `json:"result"`
	Hostname    string     `json:"hostname"`
	Pid         int        `json:"pid,omitempty"`
	StartAt     *time.Time `json:"startAt,omitempty"`
	EndAt       *time.Time `json:"endAt,omitempty"`
	SystemTime  float64    `json:"systemTime,omitempty"`
	UserTime    float64    `json:"userTime,omitempty"`
}
