package main

import (
	"flag"
	"log"
	"net"
)

var port = flag.String("port", "8082", "port number")

func handle(conn net.Conn) {
	worker := NewWorker()
	worker.Start(conn)
}

func serve() {
	flag.Parse()
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("E accept error: %v", err)
			continue
		}
		log.Printf("I accept conn: %v", conn)
		go handle(conn)
	}
}

func main() {
	serve()
}
