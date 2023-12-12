package main

import (
	"io"
	"net/http"
	"strings"
)

const baseURL = "https://www.register2park.com"

// Builds a URL for the given path and parameters
func BuildRequestWithBody(method string, path string, params map[string]string, body io.Reader) *http.Request {
	var url string
	if !strings.HasPrefix(path, "http") {
		url = baseURL + path
	}
	url = path

	if params != nil {
		takenFirst := false
		for key, value := range params {
			paramChar := "&"
			if !takenFirst {
				paramChar = "?"
				takenFirst = true
			}

			url += paramChar + key + "=" + value
		}
	}

	request, _ := http.NewRequest(method, url, body)
	AddUserAgent(request)
	return request
}

func BuildRequest(method string, path string, params map[string]string) *http.Request {
	return BuildRequestWithBody(method, path, params, nil)
}

func AddUserAgent(req *http.Request) {
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")
}

func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
