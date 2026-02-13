# jsonpath

[![Go Reference](https://pkg.go.dev/badge/github.com/saltfishpr/jsonpath.svg)](https://pkg.go.dev/github.com/saltfishpr/jsonpath)

A JSONPath query syntax parser and evaluator implemented in Go, providing a gjson-like API style.

Fully compliant with [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535.html) standard.

## Features

- RFC 9535 standard implementation
- Clean gjson-like API
- Filter expressions support `?(@.price < 10)`
- Recursive descent support `..`
- Array slicing support `[start:end:step]`
- Built-in functions: `length()`, `count()`, `match()`, `search()`, `value()`
- Custom function registration support
- Zero dependencies, pure Go implementation

## Installation

```bash
go get github.com/saltfishpr/jsonpath
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/saltfishpr/jsonpath"
)

func main() {
    json := `{
        "store": {
            "book": [
                {"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
                {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
                {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99}
            ],
            "bicycle": {"color": "red", "price": 399}
        }
    }`

    // Get all book authors
    result := jsonpath.GetMany(json, "$.store.book[*].author")
    for _, r := range result {
        fmt.Println(r.String())
    }
    // Output:
    // Nigel Rees
    // Evelyn Waugh
    // Herman Melville
}
```

## API Usage

### Get / GetMany

```go
// Get retrieves the first matching result
result := jsonpath.Get(json, "$.store.book[0].author")
fmt.Println(result.String()) // Nigel Rees

// GetMany retrieves all matching results
results := jsonpath.GetMany(json, "$..author")
for _, r := range results {
    fmt.Println(r.String())
}
```

### Result Type Conversion

```go
result := jsonpath.Get(json, "$.store.book[0].price")

// Convert to various types
fmt.Println(result.Float())  // 8.95
fmt.Println(result.Int())    // 8
fmt.Println(result.String()) // 8.95
fmt.Println(result.Bool())   // true

// Type checking
result.IsArray()  // false
result.IsObject() // false
result.IsBool()   // false
result.Exists()   // true
```

### Array and Object Operations

```go
// Get array
arr := jsonpath.Get(json, "$.store.book").Array()
for i, elem := range arr {
    fmt.Printf("[%d] %s\n", i, elem.String())
}

// Get object
obj := jsonpath.Get(json, "$.store.bicycle").Map()
for k, v := range obj {
    fmt.Printf("%s: %v\n", k, v.Value())
}
```

### Chained Queries

```go
// Continue querying on existing results
result := jsonpath.Get(json, "$.store")
bicycle := result.Get("$.bicycle")
color := bicycle.Get("$.color")
fmt.Println(color.String()) // red
```

### Function Support

The following RFC 9535 standard functions are supported:

| Function                 | Description                                              | Example                     |
| ------------------------ | -------------------------------------------------------- | --------------------------- |
| `length(value)`          | Returns string length/array element count/object key count | `length(@.title)`           |
| `count(nodes)`           | Counts the number of nodes                                | `count(@.price[?(@ > 10)])` |
| `match(value, pattern)`  | Full match against regular expression                     | `match(@.category, "^ref")` |
| `search(value, pattern)` | Search regular expression                                 | `search(@.title, "Of")`     |
| `value(nodes)`           | Extract single value from nodes                           | `value(@..isbn)`            |

## JSONPath Syntax Examples

| Expression               | Description                              |
| ------------------------ | ---------------------------------------- |
| `$.store.book[*].author` | All book authors                        |
| `$..author`              | All authors (recursive search)           |
| `$.store.*`              | All values under store                  |
| `$.store..price`         | All price values                         |
| `$..book[2]`             | The third book                           |
| `$..book[-1]`            | The last book                            |
| `$..book[:2]`            | First two books                          |
| `$..book[0,1]`           | First two books (alternative syntax)     |
| `$..book[?(@.isbn)]`     | Books with ISBN                          |
| `$..book[?(@.price<10)]` | Books with price less than 10            |
| `$..*`                   | All member values and array elements      |

## License

MIT License
