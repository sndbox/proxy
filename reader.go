package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

func toBufioReader(r io.Reader) *bufio.Reader {
	if casted, ok := r.(*bufio.Reader); ok {
		return casted
	} else {
		return bufio.NewReader(r)
	}
}

type baseReader struct {
	r     *bufio.Reader
	errCh chan error
}

func (r *baseReader) ErrorOccurred() <-chan error {
	return r.errCh
}

// similar to readLineSlice() in net/textproto/reader.go
func (r *baseReader) readLine() (string, error) {
	var line []byte
	for {
		l, more, err := r.r.ReadLine()
		if err != nil {
			return "", err
		}
		if line == nil && !more {
			return string(l), nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return string(line), nil
}

func (r *baseReader) readHeaders() (HTTPHeader, error) {
	headers := make(map[string]string)
	for {
		line, err := r.readLine()
		if err != nil {
			return nil, fmt.Errorf("Failed to read headers")
		}
		if len(line) == 0 {
			break
		}
		fs := strings.SplitN(line, ":", 2)
		if len(fs) != 2 {
			return nil, fmt.Errorf("Invalid header format")
		}
		hdr := strings.ToLower(strings.TrimSpace(fs[0]))
		headers[hdr] = strings.TrimSpace(fs[1])
	}
	return headers, nil
}

// RequestReader reads HTTP/1.1 request header
type RequestReader struct {
	baseReader
	req   *Request
	reqCh chan *Request
}

func NewRequestReader(r io.Reader) *RequestReader {
	rr := &RequestReader{
		baseReader{toBufioReader(r), make(chan error)},
		&Request{},
		make(chan *Request),
	}
	return rr
}

func (r *RequestReader) Start() {
	go func() {
		if err := r.readRequestLine(); err != nil {
			r.errCh <- err
			return
		}
		if err := r.readRequestHeaders(); err != nil {
			r.errCh <- err
			return
		}
		r.reqCh <- r.req
	}()
}

func (r *RequestReader) readRequestLine() error {
	rl, err := r.readLine()
	if err != nil {
		return fmt.Errorf("Failed to read request line: %v", err)
	}
	fields := strings.Split(rl, " ")
	if len(fields) != 3 {
		return fmt.Errorf("Invalid request line")
	}
	r.req.Method = fields[0]
	r.req.URI = fields[1]
	r.req.Version = fields[2]
	return nil
}

func (r *RequestReader) readRequestHeaders() error {
	headers, err := r.readHeaders()
	if err == nil {
		r.req.Headers = headers
	}
	return err
}

func (r *RequestReader) RequestReceived() <-chan *Request {
	return r.reqCh
}

// ResponseReader reads HTTP response headers
type ResponseReader struct {
	baseReader
	res   *Response
	resCh chan *Response
}

func NewResponseReader(r io.Reader) *ResponseReader {
	rr := &ResponseReader{
		baseReader{toBufioReader(r), make(chan error)},
		&Response{},
		make(chan *Response),
	}
	return rr
}

func (r *ResponseReader) Start() {
	go func() {
		if err := r.readStatusLine(); err != nil {
			r.errCh <- err
			return
		}
		if err := r.readResponseHeaders(); err != nil {
			r.errCh <- err
			return
		}
		r.resCh <- r.res
	}()
}

func parseStatusCode(ss string) (int, error) {
	status, err := strconv.Atoi(ss)
	first := status / 100
	if err != nil || (first < 1 || first > 5) {
		return 0, fmt.Errorf("Invalid status code: %s", ss)
	}
	return status, nil
}

func (r *ResponseReader) readStatusLine() error {
	sl, err := r.readLine()
	if err != nil {
		return fmt.Errorf("Failed to read status line: %v", err)
	}
	// TODO: Not an ideal
	fields := strings.Split(sl, " ")
	if len(fields) < 3 {
		return fmt.Errorf("Invalid status line: %s", sl)
	}
	r.res.Version = fields[0]
	r.res.Status, err = parseStatusCode(fields[1])
	if err != nil {
		return err
	}
	r.res.Phrase = strings.Join(fields[2:], " ")
	return nil
}

func (r *ResponseReader) readResponseHeaders() error {
	headers, err := r.readHeaders()
	if err == nil {
		r.res.Headers = headers
	}
	return err
}

func (r *ResponseReader) ResponseReceived() <-chan *Response {
	return r.resCh
}

// BodyReader reads body of request or response
type BodyReader interface {
	Start()
	Cancel()
	BodyReceived() <-chan []byte
	ErrorOccurred() <-chan error
}

type baseBodyReader struct {
	r      *bufio.Reader
	buf    []byte
	bodyCh chan []byte
	errCh  chan error
	done   chan struct{}
}

func (r *baseBodyReader) Cancel() {
	r.done <- struct{}{}
}

func (r *baseBodyReader) BodyReceived() <-chan []byte {
	return r.bodyCh
}

func (r *baseBodyReader) ErrorOccurred() <-chan error {
	return r.errCh
}

func (r *baseBodyReader) readAndSend(sz int) {
	for total := 0; total < sz; {
		var n int
		var err error
		m := sz - total
		if len(r.buf) > m {
			n, err = r.r.Read(r.buf[:m])
		} else {
			n, err = r.r.Read(r.buf)
		}
		if n > 0 {
			// TODO: avoid copy
			tmp := make([]byte, n)
			copy(tmp, r.buf[:n])
			r.bodyCh <- tmp
			total += n
		}
		if err != nil {
			if err == io.EOF {
				return
			}
			r.errCh <- err
			return
		}
	}
}

// FixedLengthBodyReader reads a fixed size body
type FixedLengthBodyReader struct {
	baseBodyReader
	contentLength int
}

func NewFixedLengthBodyReader(r io.Reader, cl int) *FixedLengthBodyReader {
	return &FixedLengthBodyReader{
		baseBodyReader{
			toBufioReader(r),
			make([]byte, 4096),
			make(chan []byte),
			make(chan error),
			make(chan struct{})},
		cl,
	}
}

func (r *FixedLengthBodyReader) Start() {
	go func() {
		defer func() {
			close(r.bodyCh)
			log.Printf("I FixedLengthBodyReader done")
		}()
		r.readAndSend(r.contentLength)
	}()
}

// ChunkedBodyReader reads chunked body
type ChunkedBodyReader struct {
	baseBodyReader
}

func NewChunkedBodyReader(r io.Reader) *ChunkedBodyReader {
	return &ChunkedBodyReader{
		baseBodyReader{
			toBufioReader(r),
			make([]byte, 4096),
			make(chan []byte),
			make(chan error),
			make(chan struct{}),
		},
	}
}

func (r *ChunkedBodyReader) Start() {
	go func() {
		defer func() {
			close(r.bodyCh)
			log.Printf("I ChunkedBodyReader done")
		}()

		for {
			n, err := r.readAndSendChunkLength()
			if err != nil {
				r.errCh <- err
				return
			}
			if n > 0 {
				r.readAndSend(n)
			}
			// Read trailer \r\n
			b, err := r.r.ReadBytes('\n')
			if err != nil || len(b) != 2 {
				r.errCh <- fmt.Errorf("Missing trailer in chunked encoding")
				return
			}
			r.bodyCh <- b
			if n == 0 {
				return
			}
		}
	}()
}

func (r *ChunkedBodyReader) readAndSendChunkLength() (int, error) {
	b, err := r.r.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	n := len(b)
	if n < 3 || b[n-2] != '\r' || b[n-1] != '\n' {
		return 0, fmt.Errorf("Missing chunk length: %s", string(b))
	}

	l := 0
	for _, c := range b[:n-2] {
		if c >= '0' && c <= '9' {
			l = l*16 + int(c-'0')
		} else if c >= 'a' && c <= 'f' {
			l = l*16 + int(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			l = l*16 + int(c-'A') + 10
		} else {
			return 0, fmt.Errorf("Invalid chunk length: %s", string(b))
		}
	}

	r.bodyCh <- b
	return l, nil
}
