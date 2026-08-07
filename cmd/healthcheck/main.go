package main

import (
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Get(os.Args[1])
	if err != nil {
		os.Exit(1)
	}
	_, _ = io.CopyN(io.Discard, response.Body, 4096)
	_ = response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		os.Exit(1)
	}
}
