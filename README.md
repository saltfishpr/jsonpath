# jsonpath

[![Go Reference](https://pkg.go.dev/badge/github.com/saltfishpr/jsonpath.svg)](https://pkg.go.dev/github.com/saltfishpr/jsonpath)

Go 语言实现的 JSONPath 查询语法解析器和求值器，提供类似 gjson 的 API 风格。

完全遵循 [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535.html) 标准。

## 特性

- RFC 9535 标准实现
- 类似 gjson 的简洁 API
- 支持过滤表达式 `?(@.price < 10)`
- 支持递归 descent `..`
- 支持数组切片 `[start:end:step]`
- 内置函数：`length()`, `count()`, `match()`, `search()`, `value()`
- 支持自定义函数注册
- 零依赖，纯 Go 实现

## 安装

```bash
go get github.com/saltfishpr/jsonpath
```

## 快速开始

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

    // 获取所有书籍的作者
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

## API 用法

### Get / GetMany

```go
// Get 获取第一个匹配结果
result := jsonpath.Get(json, "$.store.book[0].author")
fmt.Println(result.String()) // Nigel Rees

// GetMany 获取所有匹配结果
results := jsonpath.GetMany(json, "$..author")
for _, r := range results {
    fmt.Println(r.String())
}
```

### 结果类型转换

```go
result := jsonpath.Get(json, "$.store.book[0].price")

// 转换为各种类型
fmt.Println(result.Float())  // 8.95
fmt.Println(result.Int())    // 8
fmt.Println(result.String()) // 8.95
fmt.Println(result.Bool())   // true

// 检查类型
result.IsArray()  // false
result.IsObject() // false
result.IsBool()   // false
result.Exists()   // true
```

### 数组和对象操作

```go
// 获取数组
arr := jsonpath.Get(json, "$.store.book").Array()
for i, elem := range arr {
    fmt.Printf("[%d] %s\n", i, elem.String())
}

// 获取对象
obj := jsonpath.Get(json, "$.store.bicycle").Map()
for k, v := range obj {
    fmt.Printf("%s: %v\n", k, v.Value())
}
```

### 链式查询

```go
// 在已有结果上继续查询
result := jsonpath.Get(json, "$.store")
bicycle := result.Get("$.bicycle")
color := bicycle.Get("$.color")
fmt.Println(color.String()) // red
```

## JSONPath 语法示例

| 表达式                   | 描述                   |
| ------------------------ | ---------------------- |
| `$.store.book[*].author` | 所有书籍的作者         |
| `$..author`              | 所有作者（递归查找）   |
| `$.store.*`              | store 下的所有值       |
| `$.store..price`         | 所有价格值             |
| `$..book[2]`             | 第三本书               |
| `$..book[-1]`            | 最后一本书             |
| `$..book[:2]`            | 前两本书               |
| `$..book[0,1]`           | 前两本书（另一种写法） |
| `$..book[?(@.isbn)]`     | 有 ISBN 的书           |
| `$..book[?(@.price<10)]` | 价格小于 10 的书       |
| `$..*`                   | 所有成员值和数组元素   |

## 许可证

MIT License
