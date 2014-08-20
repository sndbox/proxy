package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func readChunkedAsString(s string) (string, error) {
	r := NewChunkedReader(strings.NewReader(s))
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, r)
	return buf.String(), err
}

func TestChunkedReader(t *testing.T) {
	actual, err := readChunkedAsString("6\r\nFooBar\r\n0\r\n\r\n")
	if err != nil {
		t.Error(err)
	}
	ExpectEqual(t, "FooBar", actual)
	actual, err = readChunkedAsString(
		"d\r\nThisIsChunked\r\n18\r\nAllYourBaseAreBelongToUs\r\n0\r\n\r\n")
	if err != nil {
		t.Error(err)
	}
	ExpectEqual(t, "ThisIsChunkedAllYourBaseAreBelongToUs", actual)
}

func TestChunkedWriter(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewChunkedWriter(buf)
	w.Write([]byte("FooBar"))
	w.Close()
	ExpectEqual(t, "6\r\nFooBar\r\n0\r\n\r\n", buf.String())
}
