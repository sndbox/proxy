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

var ResponseOK = &Response{
	Version: "HTTP/1.1",
	Status:  200,
	Phrase:  "OK",
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

func RemoveHopByHopHeaders(h HTTPHeader) {
	delete(h, "connection")
	delete(h, "keep-alive")
	delete(h, "proxy-authenticate")
	delete(h, "proxy-authorization")
	delete(h, "te")
	delete(h, "trailer")
	delete(h, "transfer-encoding")
	delete(h, "upgrade")
	// non-standard
	delete(h, "proxy-connection")
}
