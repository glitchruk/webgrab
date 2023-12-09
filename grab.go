package webgrab

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// TagName is the name of the tag that contains the data to grab.
const tagNameGrab = "grab"

// TagNameExtract is the name of the tag that contains the regex to use for
// extracting a part of the value.
const tagNameExtract = "extract"

// TagNameFilter is the name of the tag that contains the regex to use for
// filtering the value.
const tagNameFilter = "filter"

// Grabber is the struct that contains the configuration for the grabber.
type Grabber struct {
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

	// Extract is the regex to use for the grabber.
	Extract string

	// Filter is the regex to use for filtering the value.
	Filter string
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
		tag := parseTag(reflect.TypeOf(data).Elem().Field(i).Tag.Get(tagNameGrab))
		tag.Field = reflect.TypeOf(data).Elem().Field(i).Name
		tag.FieldType = reflect.TypeOf(data).Elem().Field(i).Type
		tag.Extract = reflect.TypeOf(data).Elem().Field(i).Tag.Get(tagNameExtract)
		tag.Filter = reflect.TypeOf(data).Elem().Field(i).Tag.Get(tagNameFilter)

		// Append the tag to the slice.
		tags = append(tags, tag)
	}

	// Return the tags.
	return tags
}

// Grab grabs the data from the given URL and stores it in the given data
// struct.
func (g Grabber) Grab(url string, data interface{}) error {
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

func (g Grabber) filter(str string, regex string) bool {
	// Compile the regex.
	re := regexp.MustCompile(regex)

	// Return whether the string matches the regex.
	return re.MatchString(str)
}

// extract extracts the first matched group from the given string using the
// given regex.
func (g Grabber) extract(str string, regex string) string {
	// Compile the regex.
	re := regexp.MustCompile(regex)

	// Find the first match.
	match := re.FindStringSubmatch(str)

	// If there is no match, return an empty string.
	if len(match) == 0 {
		return "(no match)"
	}

	// Return the first matched group.
	return match[1]
}

func (g Grabber) clean(str string, tag grabTag) string {
	// If the regex is empty, return the string.
	if tag.Extract != "" {
		// Extract the part of the value specified by the regex.
		str = g.extract(str, tag.Extract)
	}

	// Return the trimmed string.
	return strings.TrimSpace(str)
}

func (g Grabber) scrape(doc *goquery.Document, tag grabTag) (string, error) {
	// Find the tag in the document.
	sel := doc.Find(tag.Selector)

	// If the tag was not found, return an error.
	if sel.Length() == 0 {
		return "", fmt.Errorf("tag not found: %s", tag.Selector)
	}

	// If tag was found more than once, use the first tag.
	if sel.Length() > 1 {
		sel = sel.First()
	}

	// If the filter is not empty, filter the value.
	if tag.Filter != "" {
		// If the value does not match the filter, return an error.
		if !g.filter(sel.Text(), tag.Filter) {
			return "", fmt.Errorf("tag does not match filter: %s", tag.Filter)
		}
	}

	// If the attribute is empty, return the trimmed text of the tag.
	if tag.Attribute == "" {
		return g.clean(sel.Text(), tag), nil
	}

	// Return the attribute.
	return g.clean(sel.AttrOr(tag.Attribute, ""), tag), nil
}

func (g Grabber) scrapeSlice(doc *goquery.Document, tag grabTag) ([]string, error) {
	// Create a new slice of strings.
	strs := make([]string, 0)

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
			// If the filter is not empty, filter the value.
			if tag.Filter != "" {
				// If the value does not match the filter, return.
				if !g.filter(s.Text(), tag.Filter) {
					return
				}
			}

			// Append the text of the tag.
			strs = append(strs, g.clean(s.Text(), tag))
			return
		}

		// If the filter is not empty, filter the value.
		if tag.Filter != "" {
			// If the value does not match the filter, return.
			if !g.filter(s.AttrOr(tag.Attribute, ""), tag.Filter) {
				return
			}
		}

		// Append the attribute.
		strs = append(strs, g.clean(s.AttrOr(tag.Attribute, ""), tag))
	})

	// Return the slice.
	return strs, nil
}

func (g Grabber) scrapeStruct(doc *goquery.Document, nested interface{}) error {
	// Parse the struct.
	tags := parseStruct(nested)

	// For each tag, find the corresponding tag in the document and set the
	// value of the field to the text of the tag.
	for _, tag := range tags {
		// If the field is a struct, scrape the struct.
		if tag.FieldType.Kind() == reflect.Struct {
			g.scrapeStruct(doc, reflect.ValueOf(nested).Elem().FieldByName(tag.Field).Addr().Interface())
			continue
		}

		// If the field is a slice, scrape the slice.
		if tag.FieldType.Kind() == reflect.Slice {
			strings, err := g.scrapeSlice(doc, tag)
			if err != nil {
				continue
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
				continue
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
func New() *Grabber {
	return &Grabber{
		Timeout:      10,
		MaxRedirects: 10,
		UserAgent:    "Mozilla/5.0 (compatible; WebGrab/1.0;) Go",
	}
}
