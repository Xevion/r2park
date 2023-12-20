package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

var (
	client             *http.Client
	requestCounter     uint
	lastReload         time.Time
	parsePattern       = regexp.MustCompile(`\s*(.+)\n\s+(.+)\n\s+(\d+)\s*`)
	cachedLocations    []Location
	cachedLocationsMap map[uint]Location
	cacheExpiry        time.Time
)

func init() {
	lastReload = time.Now().Add(-time.Hour * 24 * 365) // Set the last reload time to a year ago
	cacheExpiry = time.Now().Add(-time.Second)         // Set the cache as expired initially
	cookies, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client = &http.Client{Jar: cookies}
}

func onRequest(req *http.Request) {
	log.Debugf("%s %s", req.Method, req.URL.String())
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
	doRequest(req)

	// Ring the second doorbell (seems to be a way of validating whether a client is a 'browser' or not)
	req = BuildRequest("GET", "/index.php", map[string]string{
		"width":  "1920",
		"height": "1080",
	})
	doRequest(req)

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
	currentTime := time.Now()
	lastReloadDiff := currentTime.Sub(lastReload)

	if requestCounter >= 10 {
		log.Info("Reloading session due to request count...")
	} else if lastReloadDiff >= 15*60 {
		log.Infof("Reloading session due to time (%s)...", lastReloadDiff)
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

	tryReload()
	log.Printf("Refetching locations (%s since refresh)", time.Now().Sub(cacheExpiry))

	body := "propertyNameEntered=" // Empty, so we get all locations
	req := BuildRequestWithBody("POST", "/register-get-properties-from-name", nil, bytes.NewBufferString(body))
	SetTypicalHeaders(req, nil, nil, true)

	res, err := doRequest(req)
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

type GetFormResult struct {
	propertyName     string
	address          string
	fields           []Field  // label & inputs in the form
	hiddenInputs     []string // hidden inputs in the form
	requireGuestCode bool     // whether a guest code is required
	residentProfileId string
	err              error    // any error that occurred
}

func GetForm(id uint) GetFormResult {
	body := fmt.Sprintf("propertyIdSelected=%d&propertySource=parking-snap", id)
	req := BuildRequestWithBody("POST", "/register-get-vehicle-form", nil, bytes.NewBufferString(body))
	SetTypicalHeaders(req, nil, nil, false)

	res, _ := doRequest(req)

	// Read and parse the HTML response body
	html, _ := io.ReadAll(res.Body)
	htmlString := string(html)
	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(htmlString))
	if err != nil {
		return GetFormResult{err: err}
	}

	// Check if this form is a VIP property
	if CheckGuestCodeRequired(doc) {
		return GetFormResult{
			requireGuestCode: true,
		}
	}

	// Get the resident profile 
	nextButton := doc.Find("#vehicleInformationVIP").First()
	residentProfileId, _ := nextButton.Attr("data-resident-profile-id")

	// Get the hidden inputs & form fields
	formFields, hiddenInputs := GetFields(doc)

	// Acquire the title/address
	titleElement := doc.Find("div > div > h4").First()
	title := titleElement.Text()
	address := strings.TrimSpace(titleElement.Next().Text())

	return GetFormResult{
		propertyName: title,
		address:      address,
		fields:       formFields,
		hiddenInputs: hiddenInputs,
		residentProfileId: residentProfileId,
	}
}

func GetVipForm(id uint, guestCode string) GetFormResult {
	body := fmt.Sprintf("propertyIdSelected=%d&propertySource=parking-snap&guestCode=%s", id, guestCode)
	req := BuildRequestWithBody("POST", "/register-get-vip-vehicle-form", nil, bytes.NewBufferString(body))
	SetTypicalHeaders(req, nil, nil, false)

	res, _ := doRequest(req)

	html, _ := io.ReadAll(res.Body)
	htmlString := string(html)

	if htmlString == "guest-code" {
		return GetFormResult{
			requireGuestCode: true,
			err:              fmt.Errorf("guest code is invalid"),
		}
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(htmlString))
	if err != nil {
		return GetFormResult{err: err}
	}

	// Get the hidden inputs & form fields
	formFields, hiddenInputs := GetFields(doc)

	// Acquire the title/address
	titleElement := doc.Find("div > div > h4").First()
	title := titleElement.Text()
	address := strings.TrimSpace(titleElement.Next().Text())

	return GetFormResult{
		propertyName: title,
		address:      address,
		fields:       formFields,
		hiddenInputs: hiddenInputs,
	}
}

type RegistrationResult struct {
	timestamp        time.Time
	confirmationCode string
	emailIdentifier  string
}

func RegisterVehicle(formParams map[string]string, propertyId uint, residentProfileId uint, hiddenParams []string) (bool, RegistrationResult) {
	body := url.Values{}
	body.Set("propertySource", "parking-snap")
	body.Set("propertyIdSelected", strconv.FormatUint(uint64(propertyId), 10))
	body.Set("residentProfileId", strconv.FormatUint(uint64(residentProfileId), 10))

	// Some parameters in the form are hidden, so they're just set empty to mimic the browser's behavior
	for _, hiddenParam := range hiddenParams {
		body.Set(hiddenParam, "")
	}

	// These parameters are actively used in the form
	for key, value := range formParams {
		body.Set(key, value)
	}

	req := BuildRequestWithBody("GET", "/register-vehicle-vip-process", nil, strings.NewReader(body.Encode()))
	SetTypicalHeaders(req, nil, nil, false)

	res, _ := doRequest(req)

	html, _ := io.ReadAll(res.Body)
	htmlString := string(html)

	// TODO: Parsing of success/failure
	log.Debugf("RegisterVehicle response: %s", htmlString)

	return (false, nil)
}
