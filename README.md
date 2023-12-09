# WebGrab

WebGrab is a simple Go library which allows for easy scraping of web pages. It is built on top of the [GoQuery](http://github.com/PuerkitoBio/goquery) library.

## Installation

```bash
go get github.com/glitchruk/webgrab
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/glitchruk/webgrab"
)

type Page struct {
    Title    string `grab:"title"`
    Body     string `grab:"body"`
    Keywords string `grab:"meta[name=keywords]" attr:"content"`
}

func main() {
    page := Page{}
    
    grabber := webgrab.New()
    grabber.Timeout = 30
    grabber.MaxRedirects = 10
    grabber.Grab("http://example.com", &page)

    fmt.Println(page.Title)
    fmt.Println(page.Body)
    fmt.Println(page.Keywords)
}
```

### Tag Syntax

The defined tags are:

* `grab:"selector"` - The selector to use to grab the value.
* `attr:"attribute"` - The attribute of the selected element to grab.
* `extract:"regexp"` - A regular expression to extract a value from a string.
* `filter:"regexp"` - A regular expression to filter the value of a field.

The selector is a [GoQuery](http://godoc.org/github.com/PuerkitoBio/goquery) selector. The attribute is an
optional attribute of the selected element to grab. If no attribute is
specified, the text of the selected element will be grabbed.

### Arrays

If the field is an array, the selector will be applied to each element of the
array. For example:

```go
type Page struct {
    Links []string `grab:"a[href]" attr:"href"`
}
```

### Nested Structs

It is possible to use nested structs to grab values from the page. For example,
to grab the title and meta keywords from a page:

```go
type Page struct {
    Title string `grab:"title"`
    Meta  struct {
        Keywords string `grab:"meta[name=keywords]" attr:"content"`
        Author   string `grab:"meta[name=author]" attr:"content"`
    }
}
```

### Extract

The `extract` tag can be used to extract a value from a string using a regular
expression. For example, to extract the title from a Wikipedia page:

```go
type Page struct {
    Title string `grab:"title" extract:"(.+) - Wikipedia"`
}
```

### Filter

The `filter` tag can be used to filter the value of a field. For example, to
get all links that end with `.html`:

```go
type Page struct {
    Links []string `grab:"a[href]" attr:"href" filter:".*\.html$"`
}
```
