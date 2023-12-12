package main

import (
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"
)

var (
	client         *http.Client
	cookies        *cookiejar.Jar
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

	// This request provides a PHPSESSID cookie
	req = BuildRequest("GET", "https://api.parkingsnap.com/supportedLanguages?target=en&siteName=", nil)
	client.Do(req)

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
