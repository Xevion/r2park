package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
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

// Plural returns an empty string if n is 1, otherwise it returns an "s".
// This is useful for pluralizing words in a sentence.
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

// doRequest performs the given request and calls onRequest and onResponse.
func doRequest(request *http.Request) (*http.Response, error) {
	onRequest(request)
	response, error := client.Do(request)
	onResponse(response)
	return response, error
}

// GetRandomItems returns N random items from the given array.
// The seedValue is used to control the output.
// If the array is not a slice, an error is returned.
func GetRandomItems[T any](arr []T, N int, seedValue int64) ([]T, error) {
	randgen := rand.New(rand.NewSource(seedValue))
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

// FilterLocations filters the given locations by the given query.
// If the query is empty, the locations are randomized. A seed value is used to control this output.
// If the query is not empty, the locations are filtered by the query.
// The limit parameter controls the maximum number of locations to return in all cases.
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

// The commit ID from the Git repository. Only valid at build time, but is compiled into the binary.
var CommitId = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return strings.Repeat("gubed", 8) // 40 characters
}()

// GetFooterText returns a string that includes the current time and the commit ID.
func GetFooterText() string {
	return fmt.Sprintf("%s (#%s)",
		time.Now().Format("Jan 2, 2006 3:04:05 PM"),
		strings.ToLower(CommitId[:7]))
}

// HandleError(session, interaction, parse_err)
func HandleError(session *discordgo.Session, interaction *discordgo.InteractionCreate, err error, message string) {
	log.Errorf("%s (%v)", message, err)

	// Extract wrapped errors and build embed fields
	innerErrorFields := make([]*discordgo.MessageEmbedField, 0)
	innerCount := 0
	innerError := err
	for {
		if innerError == nil {
			break
		}

		innerErrorFields = append(innerErrorFields, &discordgo.MessageEmbedField{
			Name: "Error",
			// limit to 256 characters
			Value: innerError.Error()[:256],
		})
		innerCount += 1

		if innerCount == 25 {
			log.WithField("error", err).Warn("While forming discord error embed in HandleError, reached maximum inner error count while unwrapping.")
			break
		}

		innerError = errors.Unwrap(innerError)
	}

	_, err = session.FollowupMessageCreate(interaction.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       0xff0000,
				Title:       "An error has occurred.",
				Description: message,
				Footer: &discordgo.MessageEmbedFooter{
					Text: GetFooterText(),
				},
				Fields: innerErrorFields,
			},
		},
	})

	if err != nil {
		log.WithField("error", err).Warn("Unable to provide error response to user.")
	}
}
