package main

// Not map[string][]string, unlike http.Header
type HTTPHeader map[string]string

type Request struct {
	Method  string
	URI     string
	Version string
	Headers HTTPHeader
}

type Response struct {
	Version string
	Status  int
	Phrase  string
	Headers HTTPHeader
}

var ResponseInternalError = &Response{
	Version: "HTTP/1.1",
	Status:  500,
	Phrase:  "Internal Server Error",
}

var ResponseBadRequest = &Response{
	Version: "HTTP/1.1",
	Status:  400,
	Phrase:  "Bad Request",
}
