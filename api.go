package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	client         *http.Client
	requestCounter uint
	lastReload     int64
	parsePattern   = regexp.MustCompile("\\s*(.+)\\n\\s+(.+)\\n\\s+(\\d+)\\s*")
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

	// Ring the second doorbell (seems to be a way of validating whether a client is a 'browser' or not)
	req = BuildRequest("GET", "/index.php", map[string]string{
		"width":  "1920",
		"height": "1080",
	})
	onRequest(req)
	res, _ = client.Do(req)
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
	cachedLocations    []Location
	cachedLocationsMap map[uint]Location
	cacheExpiry        time.Time
)

func init() {
	cacheExpiry = time.Now().Add(time.Hour * 24)
}

func GetLocations() []Location {
	if len(cachedLocations) > 0 && time.Now().Before(cacheExpiry) {
		return cachedLocations
	}

	tryReload("")

	body := "propertyNameEntered=" // Empty, so we get all locations
	req := BuildRequestWithBody("POST", "/register-get-properties-from-name", nil, bytes.NewBufferString(body))
	SetTypicalHeaders(req, nil, nil, true)

	onRequest(req)
	res, err := client.Do(req)
	onResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Print(err)
		return nil
	}

	cachedLocations := make([]Location, 0, 150)

	doc.Find("input.property").Each(func(i int, s *goquery.Selection) {
		matches := parsePattern.FindStringSubmatch(s.Parent().Text())
		id, _ := strconv.ParseUint(matches[3], 10, 32)

		cachedLocations = append(cachedLocations, Location{
			id:      uint(id),
			name:    matches[1],
			address: matches[2],
		})
	})

	// Build the map
	cachedLocationsMap = make(map[uint]Location, len(cachedLocations))
	for _, location := range cachedLocations {
		cachedLocationsMap[location.id] = location
	}
	cacheExpiry = time.Now().Add(time.Hour * 3)

	return cachedLocations
}
