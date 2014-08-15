package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var units = map[byte]int{
	'k': 1000,
	'm': 1000 * 1000,
	'g': 1000 * 1000 * 1000,
}

func sizeToInt(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("Invalid size")
	}
	var err error
	var m, sz int
	m, ok := units[s[len(s)-1:][0]]
	if ok {
		sz, err = strconv.Atoi(s[:len(s)-1])
	} else {
		m = 1
		sz, err = strconv.Atoi(s)
	}
	if err != nil {
		return 0, err
	}
	return sz * m, nil
}

type asciiChunk struct {
	w           io.Writer
	totalLength int
	wroteSoFar  int
	nextAscii   byte
	posInBuf    int
	buf         [4096]byte
}

func newAsciiChunk(w io.Writer, totalLength int) *asciiChunk {
	c := &asciiChunk{w, totalLength, 0, 0, 0, [4096]byte{}}
	c.prepareBuf()
	return c
}

func (c *asciiChunk) prepareBuf() {
	for i := 0; i < len(c.buf); i++ {
		for {
			c.nextAscii = (c.nextAscii + 1) % 128
			if strconv.IsPrint(rune(c.nextAscii)) && c.nextAscii != '\n' {
				break
			}
		}
		c.buf[i] = c.nextAscii
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// Writes a chunk of printable []byte, returns the number of byte written.
func (c *asciiChunk) writeNext() int {
	if c.wroteSoFar >= c.totalLength {
		return 0
	}
	last := min(c.totalLength-c.wroteSoFar, len(c.buf)-c.posInBuf)
	n, err := c.w.Write(c.buf[c.posInBuf:last])
	if err != nil {
		c.wroteSoFar = c.totalLength
		return 0
	}
	c.wroteSoFar += n
	c.posInBuf = (c.posInBuf + n) % len(c.buf)
	return n
}

func getSize(vs url.Values) (int, error) {
	if size := vs.Get("size"); size != "" {
		sz, err := sizeToInt(size)
		if err != nil {
			return 0, err
		}
		return sz, nil
	}
	return 0, fmt.Errorf("no size parameter")
}

func getDelay(vs url.Values) int {
	if ds := vs.Get("delay"); ds != "" {
		delay, err := strconv.Atoi(ds)
		if err != nil {
			return 0
		}
		return delay
	}
	return 0
}

type ChunkedHandler struct{}

func (h ChunkedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()
	sz, err := getSize(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	delay := getDelay(query)

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	chunk := newAsciiChunk(w, sz-1)
	for n := chunk.writeNext(); n > 0; n = chunk.writeNext() {
		flusher.Flush()
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
	w.Write([]byte("\n"))
	flusher.Flush()
}
