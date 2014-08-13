package main

import (
	"fmt"
	"io"
	"unicode"
)

func capitalizeHeader(h string) string {
	ret := make([]rune, len(h))
	cap := true
	for i, c := range h {
		r := rune(c)
		if cap && unicode.IsLetter(r) {
			ret[i] = unicode.ToUpper(r)
			cap = false
		} else {
			ret[i] = r
		}
		if c == '-' {
			cap = true
		}
	}
	return string(ret)
}

func WriteRequest(w io.Writer, req *Request) {
	fmt.Fprintf(w, "%s %s %s\r\n", req.Method, req.URI, req.Version)
	for k, v := range req.Headers {
		fmt.Fprintf(w, "%s: %s\r\n", capitalizeHeader(k), v)
	}
	fmt.Fprintf(w, "\r\n")
}

func WriteResponse(w io.Writer, res *Response) {
	fmt.Fprintf(w, "%s %d %s\r\n", res.Version, res.Status, res.Phrase)
	for k, v := range res.Headers {
		fmt.Fprintf(w, "%s: %s\r\n", capitalizeHeader(k), v)
	}
	fmt.Fprintf(w, "\r\n")
}
