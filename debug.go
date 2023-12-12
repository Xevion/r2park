package main

import (
	"fmt"
	"io"
	"net/http"
)

func DebugRequest(req *http.Request) string {
	str := fmt.Sprintf("[%s %s]", req.Method, req.URL.String())

	// Add all headers
	for header_name, header_values := range req.Header {
		for _, header_value := range header_values {
			str += fmt.Sprintf("\n%s: %s", header_name, header_value)
		}
	}

	// Add body
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)

		if err != nil {
			str += fmt.Sprintf("\n\n {error while reading request body buffer: %s}", err)
		} else {
			str += fmt.Sprintf("\n\n%s", body)
		}
	}

	return str
}

func DebugResponse(res *http.Response) string {
	str := fmt.Sprintf("[%s %s]", res.Status, res.Request.URL.String())

	// Add all headers
	for header_name, header_values := range res.Header {
		for _, header_value := range header_values {
			str += fmt.Sprintf("\n%s: %s", header_name, header_value)
		}
	}

	// Add body
	if res.Body != nil {
		body, err := io.ReadAll(res.Body)

		if err != nil {
			str += fmt.Sprintf("\n\n {error while reading response body buffer: %s}", err)
		} else {
			str += fmt.Sprintf("\n\n%s", body)
		}
	}

	return str
}
