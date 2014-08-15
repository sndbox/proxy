package main

import (
	"net/http"
	"io"
)

type PostEchoHandler struct{}

func (h PostEchoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	io.Copy(w, req.Body)
}
