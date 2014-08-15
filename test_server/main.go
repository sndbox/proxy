package main

import (
	"flag"
	"net/http"
)

var port = flag.String("port", "9100", "port number")

// TODO: timeout

func main() {
	flag.Parse()
	http.Handle("/chunked", &ChunkedHandler{})
	http.Handle("/post_echo", &PostEchoHandler{})
	http.ListenAndServe(":"+*port, nil)
}
