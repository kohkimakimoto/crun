package crun

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

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"
)

type prefixWriter struct {
	writer    io.Writer
	job       *Job
	midOfLine bool
	m         *sync.Mutex
}

func newPrefixWriter(w io.Writer, job *Job) *prefixWriter {
	return &prefixWriter{
		writer: w,
		job:    job,
		m:      &sync.Mutex{},
	}
}

func (w *prefixWriter) Write(buf []byte) (int, error) {
	w.m.Lock()
	defer w.m.Unlock()

	var bb bytes.Buffer
	for i, chr := range buf {
		if !w.midOfLine {
			bb.Write(w.prefixBytes())
			w.midOfLine = true
		}
		if chr == '\n' {
			w.midOfLine = false
		}
		bb.Write(buf[i : i+1])
	}
	w.writer.Write(bb.Bytes())
	return len(buf), nil
}

const ts = "2006-01-02T15:04:05.000Z07:00"

func (w *prefixWriter) prefixBytes() []byte {
	job := w.job
	str := w.job.LogPrefix

	t := time.Now()
	tsstr := t.Format(ts)
	str = strings.Replace(str, "%ts", tsstr, -1)
	str = strings.Replace(str, "%timestamp", tsstr, -1)
	str = strings.Replace(str, "%tag", job.Tag, -1)
	str = strings.Replace(str, "%t", job.Tag, -1)

	return []byte(str)
}
