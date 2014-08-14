package main

import (
	"flag"
	"log"
	"net"
)

var port = flag.String("port", "8080", "port number")

func handle(conn net.Conn) {
	worker := NewWorker()
	worker.Start(conn) // worker takes the ownership of |conn|
}

func serve() {
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handle(conn)
	}
}

func main() {
	serve()
}
