# jsonpath

面向 Go 1.18+ 的 RFC 9535 JSONPath 包，无运行时第三方依赖。输入为原始 JSON，查询结果为保留原文的 `json.RawMessage`，可同时返回 Normalized Path。

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

一次性查询使用 `QueryValues(expr, data)` 或 `QueryNodes(expr, data)`。需要复用时分别调用 `Compile` 和 `Parse`，然后使用 `query.Values(doc)` 或 `query.Nodes(doc)`。`Values` 不构建路径。调用方可以用 `encoding/json.Unmarshal` 将返回片段解码到业务类型。

当前 module path 是本地占位名称 `jsonpath`。发布到远程仓库前，应替换 `go.mod` 及内部 import 中的 module path。

## 行为约定

- 支持 root、child/descendant、name、wildcard、index、slice、filter，以及 `length`、`count`、`value`、`match`、`search` 五个标准函数。
- 编译阶段检查语法、函数参数类型及 singular query。不存在 JavaScript 求值、隐式 truthiness 或自定义函数注册。
- `Document` 和 `Query` 不可变，可并发复用；零值和 nil 不能用于求值，会返回错误。
- `Parse` 复制输入；返回的每个 JSON 片段独立分配，修改片段不会修改文档或其他结果。
- 结果保留重复节点、数组和 selector 顺序；对象按原文成员顺序遍历。无匹配返回非 nil 空 slice；匹配 `null` 返回 JSON 片段 `null`。
- 片段保留数值写法、转义和容器内部空白，但不包含值外围空白。数值比较使用精确十进制语义，不转换为 `float64`。例如 `0.1` 与 `1e-1` 相等，大于 2^53 的相邻整数仍可区分。
- JSON 输入必须包含且仅包含一个值，拒绝重复成员名、非法 UTF-8 和未配对 surrogate。重复成员名按解码后的字符串判断。
- `CompileError` 与 `DocumentError` 包含零起始字节偏移；`LimitError` 表示资源限制。使用 `errors.As` 区分错误。

## 正则与资源限制

`match` 是全字符串匹配，`search` 是子串匹配。正则使用 RFC 9485 的 checking I-Regexp 实现：`^`、`$` 是普通字符；`.` 排除 CR 和 LF；不接受 `\d`、`\w`、inline flags、lookaround 或反向引用。非法模式返回 false。

Unicode categories 跟随构建工具链的 `unicode.Version`，补齐 `Cn` 和 `C`。不同 Go 版本的 Unicode 数据可能对新增字符产生不同分类。

JSON 容器、JSONPath parser 和 I-Regexp 括号的嵌套深度限制为 1024。超限返回 `LimitError`；JSONPath parser 的计数包含 bracketed selection 和嵌套过滤表达式。

I-Regexp 使用位置集合和记忆化，支持超过 1000 次的重复以及嵌套重复，不展开巨大量词。它不保证所有模式的线性时间或空间；复杂模式、大文档和大量查询结果仍可能消耗较多资源。当前 API 没有 context 取消或执行预算。查询在内存中构建完整文档及完整结果，不是流式处理。


最低版本验收使用 Go 1.18.10 实际运行测试。32 位兼容检查：

```sh
GOOS=linux GOARCH=386 go test -c -o /tmp/jsonpath-linux-386.test .
```

## 性能测试

benchmark 分别测量查询编译（`Compile`）、JSON 解析（`Parse`）和查询求值（`Values`）三个阶段，共 59 个用例。运行全部用例：

```sh
go test -run '^$' -bench '^Benchmark(Compile|Parse|Query)$' -benchmem -benchtime=1s -count=3 .
```

以下为 Apple M4 Pro、macOS darwin/arm64、Go 1.26.1、GOMAXPROCS=14 下的实测结果。每个用例运行 3 次，全部通过；表中各指标分别取 3 次结果的中位数。benchmark 串行执行，不代表并发吞吐量。

耗时统一使用 ns/op（1 ms = 1,000,000 ns）；B/op 和 allocs/op 分别表示每次操作分配的字节数和分配次数；解析吞吐量 MB/s 按输入字节数计算。

编译和查询使用相同的 19 种 JSONPath 表达式。解析和查询分别使用 25,000 和 100,000 本书的 JSON 数据，由 `jsonpath_benchmark_test.go` 使用固定随机种子生成，包含书籍的分类、作者、标题、ISBN、价格、库存状态、标签、0–3 条评论，以及一辆自行车。数据生成均不计时。

### 查询编译

`BenchmarkCompile` 对每种表达式重复调用 `jsonpath.Compile`，计时包含语法解析、类型检查及编译阶段的正则处理，不包含 JSON 解析和查询求值。

| JSONPath                                                    | ns/op | B/op | allocs/op |
| ----------------------------------------------------------- | ----: | ---: | --------: |
| `$`                                                         | 19.14 |   64 |         1 |
| `$.store.bicycle.color`                                     | 148.9 |  480 |         7 |
| `$.store.book[0].title`                                     | 182.7 |  552 |         9 |
| `$.store.book[-1].title`                                    | 185.0 |  552 |         9 |
| `$.store.book[0:10].title`                                  | 236.5 |  560 |        10 |
| `$.store.book[::2].title`                                   | 191.7 |  552 |         9 |
| `$.store.book[0,1,-1].title`                                | 281.8 |  952 |        13 |
| `$.store.book[*]`                                           | 163.5 |  480 |         7 |
| `$.store.book[*].author`                                    | 168.1 |  544 |         8 |
| `$..price`                                                  | 60.93 |  160 |         3 |
| `$.store.book[*].reviews[*].rating`                         | 265.9 |  928 |        11 |
| `$.store.book[?@.price < 10]`                               | 352.5 | 1144 |        14 |
| `$.store.book[0].reviews[?@.rating >= 4]`                   | 439.7 | 1528 |        17 |
| `$.store.book[?search(@.title, "Number 1")]`                |  1215 | 3408 |        57 |
| `$.store.book[?@.price < 10 && @.in_stock == true].title`   | 539.5 | 1992 |        22 |
| `$.store.book[?length(@.tags) == 2].title`                  | 472.0 | 1336 |        16 |
| `$.store.book[?match(@.title, "Book Number [0-9]+")].title` |  1623 | 4864 |        82 |
| `$.store.missing`                                           | 102.9 |  288 |         5 |
| `$.store.book[?@.price < 0].title`                          | 338.8 | 1200 |        14 |

### JSON 解析

`BenchmarkParse` 复用生成的 JSON 字节数据，每次调用 `jsonpath.Parse`。计时包含输入复制和文档解析，不包含查询编译和查询求值。

| 书籍数量 |     ns/op |   MB/s |      B/op | allocs/op |
| -------: | --------: | -----: | --------: | --------: |
|   25,000 |  36065802 | 148.34 |  94047589 |    802799 |
|  100,000 | 136982531 | 156.23 | 375946102 |   3204411 |

### 查询求值

`BenchmarkQuery` 复用已编译的 Query 和已解析的 Document，调用 `query.Values(doc)`，并在计时前校验命中数量。计时包含查询求值和结果片段复制，不包含查询编译、JSON 解析和结果数量校验，也不构建 normalized path。根节点查询会复制整个 JSON 文档，返回整本书的查询也会复制对应对象。

#### 25,000 本书

| JSONPath                                                    |    ns/op |     B/op | allocs/op |
| ----------------------------------------------------------- | -------: | -------: | --------: |
| `$`                                                         |   336216 |  5357600 |         3 |
| `$.store.bicycle.color`                                     |    54.27 |       40 |         3 |
| `$.store.book[0].title`                                     |    60.53 |       48 |         3 |
| `$.store.book[-1].title`                                    |    59.83 |       56 |         3 |
| `$.store.book[0:10].title`                                  |   503.70 |      920 |        24 |
| `$.store.book[::2].title`                                   |   832741 |  1452888 |     12542 |
| `$.store.book[0,1,-1].title`                                |   230.20 |      272 |        13 |
| `$.store.book[*]`                                           |  1320385 |  6382216 |     25005 |
| `$.store.book[*].author`                                    |  1249196 |  2209552 |     25027 |
| `$..price`                                                  |  5608487 |  2803256 |     25046 |
| `$.store.book[*].reviews[*].rating`                         |  6216803 |  5797648 |     38013 |
| `$.store.book[?@.price < 10]`                               |  4142649 |  1297792 |      4630 |
| `$.store.book[0].reviews[?@.rating >= 4]`                   |   156.10 |      120 |         8 |
| `$.store.book[?search(@.title, "Number 1")]`                |  2499438 |  3060448 |     11133 |
| `$.store.book[?@.price < 10 && @.in_stock == true].title`   |  2502613 |   390632 |      3760 |
| `$.store.book[?length(@.tags) == 2].title`                  |  5195815 |  5145312 |     31257 |
| `$.store.book[?match(@.title, "Book Number [0-9]+")].title` | 35803276 | 68012488 |   1117718 |
| `$.store.missing`                                           |    19.77 |        0 |         0 |
| `$.store.book[?@.price < 0].title`                          |  3051732 |       24 |         3 |

#### 100,000 本书

| JSONPath                                                    |     ns/op |      B/op | allocs/op |
| ----------------------------------------------------------- | --------: | --------: | --------: |
| `$`                                                         |   1235800 |  21405728 |         3 |
| `$.store.bicycle.color`                                     |     55.80 |        40 |         3 |
| `$.store.book[0].title`                                     |     66.94 |        48 |         3 |
| `$.store.book[-1].title`                                    |     61.71 |        56 |         3 |
| `$.store.book[0:10].title`                                  |    518.60 |       920 |        24 |
| `$.store.book[::2].title`                                   |   7253788 |   6743800 |     50054 |
| `$.store.book[0,1,-1].title`                                |       226 |       272 |        13 |
| `$.store.book[*]`                                           |   5577186 |  25481016 |    100005 |
| `$.store.book[*].author`                                    |   8334330 |   9299600 |    100033 |
| `$..price`                                                  |  18314710 |  12193272 |    100058 |
| `$.store.book[*].reviews[*].rating`                         |  20798896 |  21517152 |    150551 |
| `$.store.book[?@.price < 10]`                               |  16779194 |   5083376 |     18336 |
| `$.store.book[0].reviews[?@.rating >= 4]`                   |     99.49 |        40 |         5 |
| `$.store.book[?search(@.title, "Number 1")]`                |  18093436 |   3062016 |     11133 |
| `$.store.book[?@.price < 10 && @.in_stock == true].title`   |  18184241 |   1847856 |     14651 |
| `$.store.book[?length(@.tags) == 2].title`                  |  29809409 |  20807608 |    125236 |
| `$.store.book[?match(@.title, "Book Number [0-9]+")].title` | 151465530 | 276002501 |   4567730 |
| `$.store.missing`                                           |     20.54 |         0 |         0 |
| `$.store.book[?@.price < 0].title`                          |  14538698 |        24 |         3 |
