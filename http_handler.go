package main

import "net/http"

type HTTPHandler struct{}

var _ http.Handler = (*HTTPHandler)(nil)

func NewHTTPHandler() *HTTPHandler {
	httpHandler := new(HTTPHandler)

	return httpHandler
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}
