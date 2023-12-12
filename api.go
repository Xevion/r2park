package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	client         *http.Client
	requestCounter uint
	lastReload     int64
)

func init() {
	cookies, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client = &http.Client{Jar: cookies}
}

func onRequest(req *http.Request) {
	log.Printf("GET %s", req.URL.String())
	requestCounter++
}

func onResponse(res *http.Response) {
	log.Printf("%s %d %s", res.Status, res.ContentLength, res.Header["Content-Type"])
}

func buildURL(path string, params map[string]string) string {
	return baseURL + path
}

// Reloads the current session and completes the initial connection process that procedes bot operations.
// This should be done regularly, such as after 10 requests or 15 minutes.
func reload(name string) {
	// Ring the doorbell
	req := BuildRequest("GET", "/", nil)
	onRequest(req)
	res, _ := client.Do(req)
	onResponse(res)

	// TODO: Verify that a PHPSESSID cookie is present

	if len(name) > 0 {
		// TODO: GET https://www.register2park.com/register-get-properties-from-name
		// TODO: GET https://www.register2park.com/register?key=678zv9zzylvw
		// TODO: GET https://www.register2park.com/register-get-properties-from-name
	}
}

// Attempts to reload the current session based on a given location's name.
// This uses the current request counter and last reload time to determine if a reload is necessary.
func tryReload(name string) {
	currentTime := time.Now().Unix()
	lastReloadDiff := currentTime - lastReload

	if requestCounter >= 10 {
		log.Println("Reloading session due to request count...")
	} else if lastReloadDiff >= 15*60 {
		log.Println("Reloading session due to time...")
	} else {
		return
	}

	reload(name)
	lastReload = currentTime
	requestCounter = 0
}

func register(location uint, code string, make string, model string, plate string) {

}

type Location struct {
	id      uint   // Used for registration internally
	name    string // Used for autocomplete & location selection
	address string // Not used in this application so far
}

var (
	cachedLocations []Location
	cacheExpiry     time.Time
)

func init() {
	cacheExpiry = time.Now().Add(time.Hour * 24)
}

func GetLocations() []Location {
	if len(cachedLocations) > 0 && time.Now().Before(cacheExpiry) {
		return cachedLocations
	}

	tryReload("")

	body := "propertyNameEntered="
	req := BuildRequestWithBody("GET", "/register-get-properties-from-name", nil, bytes.NewBufferString(body))
	SetTypicalHeaders(req, nil, nil, true)

	onRequest(req)
	res, err := client.Do(req)
	fmt.Println(DebugRequest(res.Request))
	onResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	// print response body
	response_body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(DebugResponse(res))
	fmt.Printf("%s\n", response_body)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Print(err)
		return nil
	}

	locations := make([]Location, 0, 150)

	// Find all input.property
	doc.Find("input.property").Each(func(i int, s *goquery.Selection) {
		log.Printf("%s", s.Text())
	})

	return locations
}
