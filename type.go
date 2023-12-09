package webgrab

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// TagName is the name of the tag that contains the data to grab.
const tagName = "webgrab"

// Grab is the struct that contains the configuration for the grabber.
type Grab struct {
	// Timeout is the timeout in seconds for the grabber.
	Timeout int

	// MaxRedirects is the maximum number of redirects to follow.
	MaxRedirects int

	// UserAgent is the user agent to use for the grabber.
	UserAgent string
}

type grabTag struct {
	// Selector is the CSS selector for the tag.
	Selector string

	// Attribute is the attribute of the tag to grab.
	Attribute string
}

func parseTag(tag string) grabTag {
	// Create a new grab tag.
	grabTag := grabTag{}

	// Split the tag by the comma.
	parts := strings.Split(tag, ",")

	// Set the selector.
	grabTag.Selector = parts[0]

	// If there is an attribute, set the attribute.
	if len(parts) > 1 {
		grabTag.Attribute = parts[1]
	}

	// Return the grab tag.
	return grabTag
}

// Grab grabs the data from the given URL and stores it in the given data
// struct.
func (g Grab) Grab(url string, data interface{}) error {
	// Create a new HTTP client with the given timeout.
	client := &http.Client{
		Timeout: time.Second * time.Duration(g.Timeout),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// If the number of redirects is greater than the maximum number of
			// redirects, return an error.
			if len(via) >= g.MaxRedirects {
				return http.ErrUseLastResponse
			}

			// Otherwise, return nil.
			return nil
		},
	}

	// Create a new request with the given URL.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Set the user agent header.
	req.Header.Set("User-Agent", g.UserAgent)

	// Create a new response.
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a new document from the response body.
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// For each field in the data struct, find the corresponding tag in the
	// document and set the value of the field to the text of the tag.
	for i := 0; i < reflect.ValueOf(data).NumField(); i++ {
		// Get the field.
		field := reflect.ValueOf(data).Field(i)

		// Get the tag.
		tag := parseTag(reflect.TypeOf(data).Field(i).Tag.Get(tagName))

		// If the tag is empty, skip it.
		if tag.Selector == "" {
			continue
		}

		// Are we dealing with a slice?
		if field.Kind() == reflect.Slice {
			// Create a new slice.
			slice := reflect.MakeSlice(field.Type(), 0, 0)

			// Find all of the tags in the document.
			doc.Find(tag.Selector).Each(func(i int, s *goquery.Selection) {
				// Get the text of the tag.
				text := s.Text()

				// If the attribute is not empty, get the attribute.
				if tag.Attribute != "" {
					text = s.AttrOr(tag.Attribute, "")
				}

				// Append the text to the slice.
				slice = reflect.Append(slice, reflect.ValueOf(text).Convert(field.Type().Elem()))
			})

			// Set the field to the slice.
			field.Set(slice)
		} else {
			// Find the tag in the document.
			s := doc.Find(tag.Selector)

			// Get the text of the tag.
			text := s.Text()

			// If the attribute is not empty, get the attribute.
			if tag.Attribute != "" {
				text = s.AttrOr(tag.Attribute, "")
			}

			// Set the field, casting the text to the correct type.
			field.Set(reflect.ValueOf(text).Convert(field.Type()))
		}
	}

	// Return nil.
	return nil
}

// NewGrab returns a new Grab struct with default values.
func NewGrab() *Grab {
	return &Grab{
		Timeout:      10,
		MaxRedirects: 10,
		UserAgent:    "Mozilla/5.0 (compatible; WebGrab/1.0;) Go",
	}
}
