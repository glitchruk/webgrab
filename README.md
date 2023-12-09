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
    Title    string `webgrab:"title"`
    Body     string `webgrab:"body"`
    Keywords string `webgrab:"meta[name=keywords],content"`
}

func main() {
    page := Page{}
    
    grabber := webgrab.NewGrabber()
    grabber.Timeout = 30
    grabber.MaxRedirects = 10
    grabber.Grab("http://example.com", &page)

    fmt.Println(page.Title)
    fmt.Println(page.Body)
    fmt.Println(page.Keywords)
}
```

### Tag Syntax

The tag syntax is as follows:

```go
`webgrab:"selector[,attribute]"`
```

The selector is a [GoQuery](http://godoc.org/github.com/PuerkitoBio/goquery) selector. The attribute is an
optional attribute of the selected element to grab. If no attribute is
specified, the text of the selected element will be grabbed.

### Arrays

If the field is an array, the selector will be applied to each element of the
array. For example:

```go
type Page struct {
    Links []string `webgrab:"a[href],href"`
}
```

### Nested Structs

If the field is a struct, the selector will be applied to the struct. For
example:

```go
type Page struct {
    Title string `webgrab:"title"`
    Meta  struct {
        Keywords string `webgrab:"meta[name=keywords],content"`
    }
}
```