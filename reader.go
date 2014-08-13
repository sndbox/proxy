package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

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
	var br *bufio.Reader
	if casted, ok := r.(*bufio.Reader); ok {
		br = casted
	} else {
		br = bufio.NewReader(r)
	}
	rr := &RequestReader{
		baseReader{br, make(chan error)},
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
	var br *bufio.Reader
	if casted, ok := r.(*bufio.Reader); ok {
		br = casted
	} else {
		br = bufio.NewReader(r)
	}
	rr := &ResponseReader{
		baseReader{br, make(chan error)},
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

type FixedLengthBodyReader struct {
	r             io.Reader
	contentLength int
	bodyCh        chan []byte
	errCh         chan error
	done          chan struct{}
}

func NewFixedLengthBodyReader(r io.Reader, cl int) *FixedLengthBodyReader {
	return &FixedLengthBodyReader{
		r, cl, make(chan []byte), make(chan error), make(chan struct{})}
}

func (r *FixedLengthBodyReader) Start() {
	go func() {
		defer func() {
			close(r.bodyCh)
			//close(r.errCh)
		}()

		buf := make([]byte, 4096)
		for total := 0; total < r.contentLength; {
			var n int
			var err error
			m := r.contentLength - total
			if len(buf) > m {
				n, err = r.r.Read(buf[:m])
			} else {
				n, err = r.r.Read(buf)
			}
			if n > 0 {
				// TODO: avoid copy
				tmp := make([]byte, n)
				copy(tmp, buf[:n])
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
	}()
}

func (r *FixedLengthBodyReader) Cancel() {
	r.done <- struct{}{}
}

func (r *FixedLengthBodyReader) BodyReceived() <-chan []byte {
	return r.bodyCh
}

func (r *FixedLengthBodyReader) ErrorOccurred() <-chan error {
	return r.errCh
}
