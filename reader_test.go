package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

var _ = fmt.Println

func ExpectEqual(t *testing.T, expect, actual string) {
	if expect != actual {
		t.Errorf("Got %s, want %s", actual, expect)
	}
}

func readRequestSync(r io.Reader) (*Request, error) {
	reqReader := NewRequestReader(r)
	reqReader.Start()
	for {
		select {
		case req := <-reqReader.RequestReceived():
			return req, nil
		case err := <-reqReader.ErrorOccurred():
			return nil, err
		}
	}
	return nil, nil
}

func readBodySync(r BodyReader) ([]byte, error) {
	r.Start()

	var body []byte
	for {
		select {
		case b := <-r.BodyReceived():
			if len(b) == 0 {
				return body, nil
			}
			body = append(body, b...)
		case err := <-r.ErrorOccurred():
			return nil, err
		}
	}
}

func readResponseSync(r io.Reader) (*Response, error) {
	resReader := NewResponseReader(r)
	resReader.Start()
	for {
		select {
		case res := <-resReader.ResponseReceived():
			return res, nil
		case err := <-resReader.ErrorOccurred():
			return nil, err
		}
	}
	return nil, nil
}

func TestRequestReader(t *testing.T) {
	r := strings.NewReader("GET / HTTP/1.1\r\nHost: www.google.com\r\n\r\n")
	req, err := readRequestSync(r)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	ExpectEqual(t, "GET", req.Method)
	ExpectEqual(t, "/", req.URI)
	ExpectEqual(t, "HTTP/1.1", req.Version)
	ExpectEqual(t, "www.google.com", req.Headers["host"])
}

func TestResponseReader(t *testing.T) {
	r := strings.NewReader("HTTP/1.1 200 OK\r\nHost: www.google.com\r\n\r\nFooBar")
	res, err := readResponseSync(r)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	ExpectEqual(t, "HTTP/1.1", res.Version)
	ExpectEqual(t, "200", strconv.Itoa(res.Status))
	ExpectEqual(t, "OK", res.Phrase)
	ExpectEqual(t, "www.google.com", res.Headers["host"])
}

func TestFixedLengthBodyReader(t *testing.T) {
	sr := strings.NewReader("FooBarBaz")
	r := NewFixedLengthBodyReader(sr, 6)
	body, err := readBodySync(r)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	ExpectEqual(t, "FooBar", string(body))
}

func TestChunkedBodyReader(t *testing.T) {
	sr := strings.NewReader("6\r\nFooBar\r\nd\r\nThisIsChunked\r\n0\r\n\r\n")
	r := NewChunkedBodyReader(sr)
	body, err := readBodySync(r)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	ExpectEqual(t, "6\r\nFooBar\r\nd\r\nThisIsChunked\r\n0\r\n\r\n", string(body))
}
