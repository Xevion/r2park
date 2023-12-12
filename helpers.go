package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
)

const baseURL = "https://www.register2park.com"

// Builds a URL for the given path and parameters
func BuildRequestWithBody(method string, path string, params map[string]string, body io.Reader) *http.Request {
	var url string
	if !strings.HasPrefix(path, "http") {
		url = baseURL + path
	} else {
		url = path
	}

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
	SetTypicalHeaders(request, nil, nil, false)

	return request
}

func BuildRequest(method string, path string, params map[string]string) *http.Request {
	return BuildRequestWithBody(method, path, params, nil)
}

func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Sets User-Agent, Host, Content-Type, and Referer headers on the given request.
// If contentType is empty, "application/x-www-form-urlencoded; charset=UTF-8" is used.
// If referrer is empty, "https://www.register2park.com/register" is used.
// If xmlRequest is true, "X-Requested-With" is set to "XMLHttpRequest".
func SetTypicalHeaders(req *http.Request, contentType *string, referrer *string, xmlRequest bool) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")

	if xmlRequest {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	if contentType == nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	} else {
		req.Header.Set("Content-Type", *contentType)
	}

	if referrer == nil {
		req.Header.Set("Referer", "https://www.register2park.com/register")
	} else {
		req.Header.Set("Referer", *referrer)
	}
}

func GetRandomItems[T any](arr []T, N int, seed_value int64) ([]T, error) {
	randgen := rand.New(rand.NewSource(seed_value))
	arrValue := reflect.ValueOf(arr)

	if arrValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not a slice")
	}

	if arrValue.Len() < N {
		return nil, fmt.Errorf("array length is less than N")
	}

	selectedIndices := make(map[int]bool)
	selectedItems := make([]T, 0, N)

	for len(selectedItems) < N {
		randomIndex := randgen.Intn(arrValue.Len())

		// Check if the index is not already selected
		if !selectedIndices[randomIndex] {
			selectedIndices[randomIndex] = true
			selectedItems = append(selectedItems, arrValue.Index(randomIndex).Interface().(T))
		}
	}

	return selectedItems, nil
}

func FilterLocations(all_locations []Location, query string, limit int, seed_value int64) []Location {
	if len(query) == 0 {
		randomized, err := GetRandomItems(all_locations, limit, seed_value)
		if err != nil {
			log.Fatal(err)
		}
		return randomized
	}

	lower_query := strings.ToLower(query)
	filtered_locations := make([]Location, 0, limit)
	matches := 0

	for _, location := range all_locations {
		if strings.Contains(strings.ToLower(location.name), lower_query) {
			filtered_locations = append(filtered_locations, location)
			matches++

			if matches >= limit {
				break
			}
		}
	}

	return filtered_locations
}
