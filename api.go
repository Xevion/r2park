package main

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

var (
	client             *http.Client
	requestCounter     uint
	lastReload         int64
	parsePattern       = regexp.MustCompile(`\s*(.+)\n\s+(.+)\n\s+(\d+)\s*`)
	cachedLocations    []Location
	cachedLocationsMap map[uint]Location
	cacheExpiry        time.Time
)

func init() {
	cacheExpiry = time.Now().Add(-time.Second) // Set the cache as expired initially
	cookies, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client = &http.Client{Jar: cookies}
}

func onRequest(req *http.Request) {
	log.Debugf("GET %s", req.URL.String())
	requestCounter++
}

func onResponse(res *http.Response) {
	log.Debugf("%s %d %s", res.Status, res.ContentLength, res.Header["Content-Type"])
}

// Reloads the current session and completes the initial connection process that procedes bot operations.
// This should be done regularly, such as after 10 requests or 15 minutes.
func reload() {
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

	// Verify that a PHPSESSID cookie is present
	site_cookies := client.Jar.Cookies(req.URL)
	has_php_session := slices.ContainsFunc(site_cookies, func(cookie *http.Cookie) bool {
		return cookie.Name == "PHPSESSID"
	})

	if !has_php_session {
		log.Fatal("PHPSESSID cookie not found")
	} else {
		log.Debugf("PHPSESSID cookie found")
	}
}

// Attempts to reload the current session based on a given location's name.
// This uses the current request counter and last reload time to determine if a reload is necessary.
func tryReload() {
	currentTime := time.Now().Unix()
	lastReloadDiff := currentTime - lastReload

	if requestCounter >= 10 {
		log.Info("Reloading session due to request count...")
	} else if lastReloadDiff >= 15*60 {
		log.Info("Reloading session due to time...")
	} else {
		return
	}

	reload()
	lastReload = currentTime
	requestCounter = 0
}

func register(location uint, code string, make string, model string, plate string) {

}

func GetLocations() []Location {
	if time.Now().Before(cacheExpiry) {
		return cachedLocations
	}

	tryReload("")
	log.Printf("Refetching locations (%s since refresh)", time.Now().Sub(cacheExpiry))

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

	cachedLocations = make([]Location, 0, 150)

	doc.Find("input.property").Each(func(i int, s *goquery.Selection) {
		matches := parsePattern.FindStringSubmatch(s.Parent().Text())

		key_attr, _ := s.Attr("data-locate-key")
		id_attr, _ := s.Attr("value")
		id, _ := strconv.ParseUint(id_attr, 10, 32)

		cachedLocations = append(cachedLocations, Location{
			id:      uint(id),
			key:     key_attr,
			name:    matches[1],
			address: matches[2] + " " + matches[3],
		})
	})

	// Build the map
	cachedLocationsMap = make(map[uint]Location, len(cachedLocations))
	for _, location := range cachedLocations {
		cachedLocationsMap[location.id] = location
	}
	cacheExpiry = time.Now().Add(time.Hour * 8) // Cache for 8 hours

	return cachedLocations
}
