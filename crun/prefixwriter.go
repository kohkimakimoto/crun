package crun

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"
)

type prefixWriter struct {
	writer    io.Writer
	c         *Crun
	midOfLine bool
	m         *sync.Mutex
}

func newPrefixWriter(w io.Writer, c *Crun) *prefixWriter {
	return &prefixWriter{
		writer: w,
		c:      c,
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
	c := w.c
	str := w.c.LogPrefix

	t := time.Now()
	tsstr := t.Format(ts)
	str = strings.Replace(str, "%ts", tsstr, -1)
	str = strings.Replace(str, "%timestamp", tsstr, -1)
	str = strings.Replace(str, "%tag", c.Tag, -1)
	str = strings.Replace(str, "%t", c.Tag, -1)

	return []byte(str)
}
