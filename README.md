# jsonpath

[![Go Reference](https://pkg.go.dev/badge/github.com/saltfishpr/jsonpath.svg)](https://pkg.go.dev/github.com/saltfishpr/jsonpath)

RFC 9535 JSONPath for Go，无第三方运行时依赖。

## 安装

```sh
go get github.com/saltfishpr/jsonpath
```

需要 Go 1.18 或更高版本。

## 使用

```go
package main

import (
    "fmt"
    "log"

    "github.com/saltfishpr/jsonpath"
)

func main() {
    doc, err := jsonpath.Parse([]byte(`{"books":[{"title":"Go","price":8},{"title":"Rust","price":12}]}`))
    if err != nil {
        log.Fatal(err)
    }
    query, err := jsonpath.Compile(`$.books[?@.price < 10].title`)
    if err != nil {
        log.Fatal(err)
    }
    nodes, err := query.Nodes(doc)
    if err != nil {
        log.Fatal(err)
    }
    for _, node := range nodes {
        fmt.Printf("%s: %s\n", node.Path, node.Value)
        // $['books'][0]['title']: "Go"
    }
}
```

用 `QueryValues` 查询值：

```go
values, err := jsonpath.QueryValues(`$.books[*].title`, data)
// values[0] = "Go", values[1] = "Rust"
```

配合 `encoding/json` 解码到结构体：

```go
type Book struct {
    Title string `json:"title"`
    Price int    `json:"price"`
}
values, _ := jsonpath.QueryValues(`$.books[?@.price < 10]`, data)
var book Book
_ = json.Unmarshal(values[0], &book)
```

## 表达式示例

```
$.store.book[0].title                                   // 指定索引
$.store.book[-1].title                                  // 负索引
$.store.book[0:10].title                                // 切片
$.store.book[*].author                                  // wildcard
$..price                                                // descendant
$.store.book[?@.price < 10 && @.in_stock].title         // 过滤；数值每次都按精确十进制比较
$.store.book[?length(@.tags) == 2].title                // length / count / value
$.store.book[?match(@.title, "Book [0-9]+")].title      // RFC 9485 I-Regexp 全匹配
$.store.book[?search(@.title, "Go")].title              // RFC 9485 I-Regexp 子串匹配
```

完整 API 和行为约定见 [pkg.go.dev](https://pkg.go.dev/github.com/saltfishpr/jsonpath)。

## License

MIT
