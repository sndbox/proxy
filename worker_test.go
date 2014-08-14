package main

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

var _ = fmt.Println

type MockAddr struct {
	str string
}

func (m MockAddr) Network() string { return "" }
func (m MockAddr) String() string  { return m.str }

type MockConn struct {
	*bytes.Buffer
	addr MockAddr
}

func (m *MockConn) Close() error {
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return nil
}

func (m *MockConn) RemoteAddr() net.Addr {
	return m.addr
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func prepareMocks() (*MockConn, *MockConn) {
	sConn := &MockConn{
		new(bytes.Buffer),
		MockAddr{"(server)"},
	}
	serverDialer = func(addr string) (net.Conn, error) {
		return sConn, nil
	}

	cConn := &MockConn{
		new(bytes.Buffer),
		MockAddr{"(client)"},
	}
	return cConn, sConn
}

func TestWorkerContentLength(t *testing.T) {
	cConn, sConn := prepareMocks()

	cConn.WriteString("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	sConn.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 6\r\n\r\nFooBar")

	w := NewWorker()
	w.Start(cConn)

	ss := []string{
		"HTTP/1.1 200 OK\r\n",
		"Content-Length: 6\r\n",
		"Content-Type: text/plain\r\n",
		"\r\n",
		"FooBar",
	}
	expect := strings.Join(ss, "")
	ExpectEqual(t, expect, cConn.String())
}

func TestWorkerChunked(t *testing.T) {
	cConn, sConn := prepareMocks()

	cConn.WriteString("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	sConn.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nTransfer-Encoding: chunked\r\n\r\n6\r\nFooBar\r\n0\r\n")

	w := NewWorker()
	w.Start(cConn)

	ss := []string{
		"HTTP/1.1 200 OK\r\n",
		"Content-Type: text/plain\r\n",
		"Transfer-Encoding: chunked\r\n",
		"\r\n",
		"6\r\n",
		"FooBar\r\n",
		"0\r\n",
	}
	expect := strings.Join(ss, "")
	ExpectEqual(t, expect, cConn.String())
}
