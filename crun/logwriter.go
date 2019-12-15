package crun

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"
)

type logWriter struct {
	writer    io.Writer
	c         *Crun
	midOfLine bool
	m         *sync.Mutex
}

func newLogWriter(w io.Writer, c *Crun) *logWriter {
	return &logWriter{
		writer: w,
		c:      c,
		m:      &sync.Mutex{},
	}
}

func (w *logWriter) Write(buf []byte) (int, error) {
	w.m.Lock()
	defer w.m.Unlock()

	if w.c.Config.LogPrefix == "" {
		return w.writer.Write(buf)
	}

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

func (w *logWriter) prefixBytes() []byte {
	c := w.c
	str := w.c.Config.LogPrefix

	t := time.Now()
	tsstr := t.Format(ts)
	str = strings.Replace(str, "%ts", tsstr, -1)
	str = strings.Replace(str, "%timestamp", tsstr, -1)
	str = strings.Replace(str, "%tag", c.Config.Tag, -1)
	str = strings.Replace(str, "%t", c.Config.Tag, -1)

	return []byte(str)
}
