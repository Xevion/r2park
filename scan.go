package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func Scan() {
	locations := GetLocations()
	total := len(locations)

	for i, location := range locations {
		log.Debugf("[%6.2f] Fetching \"%s\" ", float64(i+1)/float64(total)*100, location.name)

		body := fmt.Sprintf("propertyIdSelected=%d&propertySource=parking-snap", location.id)
		req := BuildRequestWithBody("POST", "/register-get-vehicle-form", nil, bytes.NewBufferString(body))
		SetTypicalHeaders(req, nil, nil, false)
		res, _ := doRequest(req)

		html, _ := io.ReadAll(res.Body)

		html_path := filepath.Join("./forms", fmt.Sprintf("%d.html", location.id))

		// Check that file exists and has more than 16 bytes
		stats, err := os.Stat(html_path)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			// File does not exist, create it
			file, err := os.Create(html_path)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			_, err = file.Write(html)
			if err != nil {
				panic(err)
			}
		} else if stats.Size() < 16 {
			// File exists, but is empty
			file, err := os.OpenFile(html_path, os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			_, err = file.Write(html)
			if err != nil {
				panic(err)
			}
		} else {
			// File exists and is not empty, do nothing
		}
	}
}
