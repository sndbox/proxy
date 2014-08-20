package main

import (
	"bufio"
	"fmt"
	"io"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type ChunkedReader struct {
	r        *bufio.Reader
	chunkLen int // -1 means the beginning of the next chunk
}

func NewChunkedReader(r io.Reader) *ChunkedReader {
	// TODO: avoid creating new bufio.Reader if r is already implements it
	return &ChunkedReader{bufio.NewReader(r), -1}
}

func (r *ChunkedReader) readChunkLength() error {
	b, err := r.r.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("Failed to read chunk length: %v", err)
	}
	blen := len(b)
	if blen < 2 || b[blen-2] != '\r' || b[blen-1] != '\n' {
		return fmt.Errorf("Failed to read CRLF")
	}

	length := int(0)
	for _, v := range b[:blen-2] {
		if v >= '0' && v <= '9' {
			length = length*16 + int(v-'0')
		} else if v >= 'a' && v <= 'f' {
			length = length*16 + int(v-'a') + 10
		} else if v >= 'A' && v <= 'F' {
			length = length*16 + int(v-'a') + 10
		} else {
			return fmt.Errorf("Invalid chunk length: %s", string(b))
		}
	}
	r.chunkLen = length
	return nil
}

func (r *ChunkedReader) readCRLF() error {
	b, err := r.r.ReadBytes('\n')
	if err != nil {
		return err
	}
	blen := len(b)
	if blen != 2 || b[blen-2] != '\r' || b[blen-1] != '\n' {
		return fmt.Errorf("Failed to read CRLF")
	}
	return nil
}

func (r *ChunkedReader) Read(b []byte) (int, error) {
	if r.chunkLen < 0 {
		if err := r.readChunkLength(); err != nil {
			return 0, err
		}
	}
	if r.chunkLen == 0 {
		err := r.readCRLF()
		return 0, err
	}

	n := min(r.chunkLen, len(b))
	m, err := r.r.Read(b[:n])
	r.chunkLen -= m
	if r.chunkLen == 0 {
		r.chunkLen = -1
		err = r.readCRLF()
	}
	if err != nil {
		return m, err
	}

	return m, nil
}

var crlf = []byte("\r\n")
var closeBytes = []byte("0\r\n\r\n")

type ChunkedWriter struct {
	w io.Writer
}

func NewChunkedWriter(w io.Writer) *ChunkedWriter {
	return &ChunkedWriter{w}
}

func (w *ChunkedWriter) writeChunkLength(n int) error {
	b := make([]byte, 0, 8)
	for n > 0 {
		hex := n % 16
		if hex >= 10 {
			b = append(b, byte(hex-10)+'a')
		} else {
			b = append(b, byte(hex)+'0')
		}
		n = n / 16
	}
	// TODO: avoid reverse
	blen := len(b)
	for i := 0; i < blen/2; i++ {
		x, y := b[i], b[blen-i-1]
		b[i] = y
		b[blen-i-1] = x
	}
	b = append(b, crlf...)
	m, err := w.w.Write(b)
	if m != blen+2 || err != nil {
		return fmt.Errorf("Failed to write chunk length")
	}
	return nil
}

func (w *ChunkedWriter) Write(b []byte) (int, error) {
	blen := len(b)
	if err := w.writeChunkLength(blen); err != nil {
		return 0, err
	}
	writeSoFar := 0
	for writeSoFar < blen {
		m, err := w.w.Write(b[writeSoFar:])
		writeSoFar += m
		if err != nil {
			return writeSoFar, err
		}
	}
	m, err := w.w.Write(crlf)
	if m != 2 || err != nil {
		return blen, fmt.Errorf("Failed to write CRLF")
	}
	return blen, nil
}

func (w *ChunkedWriter) Close() error {
	n, err := w.w.Write(closeBytes)
	if err != nil || n != len(closeBytes) {
		return fmt.Errorf("Failed to write the last chunk")
	}
	return nil
}
