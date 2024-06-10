package main

import (
	"bytes"
	"errors"
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
	vehicleIdPattern   = regexp.MustCompile(`data-vehicle-id="(\d+)"`)
	timestampPattern   = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2} [AP]M`)
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
	log.WithFields(log.Fields{
		"method": req.Method,
		"url":    req.URL.String(),
	}).Debugf("%s %s", req.Method, req.URL.String())
	requestCounter++
}

func onResponse(res *http.Response) {
	log.WithFields(log.Fields{
		"status": res.Status,
		"length": res.ContentLength,
		"type":   res.Header["Content-Type"],
	}).Debugf("%s %d %s", res.Status, res.ContentLength, res.Header["Content-Type"])
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
		log.WithFields(log.Fields{"cookies": site_cookies}).Panic("PHPSESSID cookie not found")
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
		log.WithFields(log.Fields{"requestCounter": requestCounter}).Info("Reloading session due to request count")
		log.Info("Reloading session due to request count")
	} else if lastReloadDiff >= 15*60 {
		log.WithFields(log.Fields{"lastReload": lastReload, "difference": lastReloadDiff}).Info("Reloading session due to time")
	} else {
		return
	}

	reload()
	lastReload = currentTime
	requestCounter = 0
}

func GetLocations() []Location {
	if time.Now().Before(cacheExpiry) {
		return cachedLocations
	}

	tryReload()
	log.WithFields(log.Fields{"sinceRefresh": time.Now().Sub(cacheExpiry)}).Debug("Refetching locations")

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

	// Prepare an array to store the locations
	cachedLocations = make([]Location, 0, 150)

	// Parse the locations from the HTML
	doc.Find("input.property").Each(func(i int, s *goquery.Selection) {
		// Parse the name and address
		matches := parsePattern.FindStringSubmatch(s.Parent().Text())

		key_attr, _ := s.Attr("data-locate-key")
		id_attr, _ := s.Attr("value")
		id, _ := strconv.ParseUint(id_attr, 10, 32)

		cachedLocations = append(cachedLocations, Location{
			// the ID used for the form, actively used in selecting the registration form
			id: uint(id),
			// key is not used for anything, but we parse it anyway
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
	propertyName      string
	address           string
	fields            []Field  // label & inputs in the form
	hiddenInputs      []string // hidden inputs in the form
	requireGuestCode  bool     // whether a guest code is required
	residentProfileId string
	err               error // any error that occurred
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

	// TODO: Validate that vehicleInformationVIP is actually real for non-VIP properties

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
		propertyName:      title,
		address:           address,
		fields:            formFields,
		hiddenInputs:      hiddenInputs,
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

	// Acquire the resident profile ID
	nextButton := doc.Find("#vehicleInformationVIP").First()
	residentProfileId, exists := nextButton.Attr("data-resident-profile-id")
	if !exists {
		return GetFormResult{err: errors.New("resident profile ID not found")}
	}

	return GetFormResult{
		propertyName:      title,
		address:           address,
		fields:            formFields,
		hiddenInputs:      hiddenInputs,
		residentProfileId: residentProfileId,
	}
}

type RegistrationResult struct {
	success          bool
	timestamp        time.Time
	confirmationCode string
	vehicleId        string
}

func RegisterVehicle(formParams map[string]string, propertyId uint, residentProfileId uint, hiddenParams []string) (*RegistrationResult, error) {
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

	// Build the request
	req := BuildRequestWithBody("POST", "/register-vehicle-vip-process", nil, strings.NewReader(body.Encode()))
	SetTypicalHeaders(req, nil, nil, false)

	// Send the request
	res, err := doRequest(req)
	if err != nil {
		return nil, err
	}

	// Read the response body
	html, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	htmlString := string(html)

	// Success can be measured with the presence of either "Approved" or "Denied" (if neither, it's an error)
	result := &RegistrationResult{success: strings.Contains(htmlString, "Approved")}

	// Sanity check that a proper result was returned
	if !result.success && !strings.Contains(htmlString, "Denied") {
		return nil, fmt.Errorf("unexpected response: %v - %s", res, htmlString)
	}

	// Parse the HTML response
	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(htmlString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML of approved registration result partial: %w", err)
	}

	// Search for attributes thar are only present on success
	if result.success {
		// Search for 'data-vehicle-id' with regex
		vehicleIdMatches := vehicleIdPattern.FindStringSubmatch(htmlString)
		if len(vehicleIdMatches) > 1 {
			result.vehicleId = vehicleIdMatches[1]
		}

		var sibling *goquery.Selection

		// Look for a 'p' tag that contains the text 'Confirmation Code:'
		doc.Find("div.circle-inner > p").Each(func(i int, s *goquery.Selection) {
			// If we've already found the element, stop searching
			if sibling != nil {
				return
			} else if strings.Contains(s.Text(), "Confirmation Code:") {
				sibling = s
			}
		})

		// The confirmation code is the next sibling of the 'p' tag
		if sibling != nil {
			result.confirmationCode = sibling.Next().Text()
		}
	}

	// Find timestamp: look for a 'strong' tag that contains 'Registration Date/Time:' inside a p, inside a div.circle-inner
	var parent *goquery.Selection
	doc.Find("div.circle-inner > p").Each(func(i int, s *goquery.Selection) {
		// If we've already found the element, stop searching
		if parent != nil {
			return
		} else if strings.Contains(s.Text(), "Date/Time:") {
			// Will start with 'Denied' or 'Registration'
			parent = s.Parent()
		}
	})

	// The timestamp is a untagged text node inside the p tag
	if parent != nil {
		// Get the raw text of the parent, then find the timestamp within it
		rawText := parent.Text()
		match := timestampPattern.FindString(rawText)

		// If we found a match, parse it into a time.Time
		if match != "" {
			timestamp, err := time.Parse("2006-01-02 03:04 PM", match)

			// Silently log the error if timestamp parsing fails
			if err != nil {
				log.Errorf("failed to parse timestamp: %v", err)
				result.timestamp = time.Time{}
			} else {
				result.timestamp = timestamp
			}
		}
	}

	return result, nil
}

// RegisterEmailConfirmation sends a request to the server to send a confirmation email regarding a vehicle's registration.
// Example Parameters: email=xevion@xevion.dev, vehicleId=63283, propertyId=63184
func RegisterEmailConfirmation(email string, vehicleId string, propertyId string) (bool, error) {
	params := map[string]string{
		"email":          email,
		"vehicleId":      vehicleId,
		"propertyId":     propertyId,
		"propertySource": "parking-snap",
	}

	req := BuildRequestWithBody("GET", "/register-vehicle-confirmation-process", params, nil)
	SetTypicalHeaders(req, nil, nil, false)

	res, err := doRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to send email confirmation request: %w", err)
	} else if res.StatusCode != 200 {
		return false, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	html, _ := io.ReadAll(res.Body)
	htmlString := string(html)

	if htmlString == "ok" {
		return true, nil
	}

	return false, fmt.Errorf("unexpected response: %v - %s", res, htmlString)
}
