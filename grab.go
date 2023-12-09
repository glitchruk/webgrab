package webgrab

import (
	"fmt"
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
	// Field is the name of the field in the struct.
	Field string

	// FieldType is the type of the field in the struct.
	FieldType reflect.Type

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

// parseStruct parses the given struct and returns a slice of grab tags.
func parseStruct(data interface{}) []grabTag {
	// Create a new slice of grab tags.
	tags := make([]grabTag, 0)

	// For each field in the data struct, find the corresponding tag in the
	// document and set the value of the field to the text of the tag.
	for i := 0; i < reflect.TypeOf(data).Elem().NumField(); i++ {
		// Get the tag.
		tag := parseTag(reflect.TypeOf(data).Elem().Field(i).Tag.Get(tagName))
		tag.Field = reflect.TypeOf(data).Elem().Field(i).Name
		tag.FieldType = reflect.TypeOf(data).Elem().Field(i).Type

		// If the tag is empty, skip it.
		if tag.Selector == "" {
			continue
		}

		// Append the tag to the slice.
		tags = append(tags, tag)
	}

	// Return the tags.
	return tags
}

// Grab grabs the data from the given URL and stores it in the given data
// struct.
func (g Grab) Grab(url string, data interface{}) error {
	// If the data is not a pointer, return an error.
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return fmt.Errorf("data must be a pointer")
	}

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

	// Perform the scrape.
	err = g.scrapeStruct(doc, data)
	if err != nil {
		return err
	}

	// Return nil.
	return nil
}

func (g Grab) scrape(doc *goquery.Document, tag grabTag) (string, error) {
	// Find the tag in the document.
	sel := doc.Find(tag.Selector)

	// If the tag was not found, return an error.
	if sel.Length() == 0 {
		return "", fmt.Errorf("tag not found: %s", tag.Selector)
	}

	// If the attribute is empty, return the text of the tag.
	if tag.Attribute == "" {
		return sel.Text(), nil
	}

	// Return the attribute.
	return sel.AttrOr(tag.Attribute, ""), nil
}

func (g Grab) scrapeSlice(doc *goquery.Document, tag grabTag) ([]string, error) {
	// Create a new slice of strings.
	strings := make([]string, 0)

	// Find the tags in the document.
	sel := doc.Find(tag.Selector)

	// If the tag was not found, return an error.
	if sel.Length() == 0 {
		return nil, fmt.Errorf("tag not found: %s", tag.Selector)
	}

	// For each tag, append the text of the tag to the slice.
	sel.Each(func(i int, s *goquery.Selection) {
		// If the attribute is empty, append the text of the tag.
		if tag.Attribute == "" {
			strings = append(strings, s.Text())
			return
		}

		// Append the attribute.
		strings = append(strings, s.AttrOr(tag.Attribute, ""))
	})

	// Return the slice.
	return strings, nil
}

func (g Grab) scrapeStruct(doc *goquery.Document, nested interface{}) error {
	// Parse the struct.
	tags := parseStruct(nested)

	// For each tag, find the corresponding tag in the document and set the
	// value of the field to the text of the tag.
	for _, tag := range tags {
		// If the field is a struct, scrape the struct.
		if tag.FieldType.Kind() == reflect.Struct {
			err := g.scrapeStruct(doc, reflect.ValueOf(nested).Elem().FieldByName(tag.Field).Addr().Interface())
			if err != nil {
				return err
			}
			continue
		}

		// If the field is a slice, scrape the slice.
		if tag.FieldType.Kind() == reflect.Slice {
			strings, err := g.scrapeSlice(doc, tag)
			if err != nil {
				return err
			}

			// Create a new slice.
			slice := reflect.MakeSlice(tag.FieldType, len(strings), len(strings))

			// For each string, set the value of the slice to the string.
			for i, str := range strings {
				slice.Index(i).SetString(str)
			}

			// Set the value of the field to the slice.
			reflect.ValueOf(nested).Elem().FieldByName(tag.Field).Set(slice)
			continue
		}

		// If the field is a string, scrape the string.
		if tag.FieldType.Kind() == reflect.String {
			str, err := g.scrape(doc, tag)
			if err != nil {
				return err
			}

			// Set the value of the field to the string.
			reflect.ValueOf(nested).Elem().FieldByName(tag.Field).SetString(str)
			continue
		}
	}

	// Return nil.
	return nil
}

// NewGrab returns a new Grab struct with default values.
func NewGrabber() *Grab {
	return &Grab{
		Timeout:      10,
		MaxRedirects: 10,
		UserAgent:    "Mozilla/5.0 (compatible; WebGrab/1.0;) Go",
	}
}
