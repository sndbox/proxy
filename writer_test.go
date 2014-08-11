package main

import (
	"bytes"
	"testing"
)

func TestWriteRequest(t *testing.T) {
	req := &Request{
		Method:  "GET",
		URI:     "/",
		Version: "HTTP/1.1",
		Headers: map[string]string{
			"Host": "localhost",
		},
	}
	expect := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	w := new(bytes.Buffer)
	WriteRequest(w, req)
	ExpectEqual(t, expect, w.String())
}

func TestWriteResponse(t *testing.T) {
	res := &Response{
		Version: "HTTP/1.1",
		Status:  200,
		Phrase:  "OK",
		Headers: map[string]string{
			"Host": "localhost",
		},
	}
	expect := "HTTP/1.1 200 OK\r\nHost: localhost\r\n\r\n"
	w := new(bytes.Buffer)
	WriteResponse(w, res)
	ExpectEqual(t, expect, w.String())
}
