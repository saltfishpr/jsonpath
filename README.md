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

运行全部 benchmark（编译、解析及 14 种查询场景）：

```sh
go test -run '^$' -bench . -benchmem -benchtime=100ms -count=3
```

以下为 Apple M4 Pro、macOS darwin/arm64、Go 1.26.1、GOMAXPROCS=14 下的实测结果。128 个 benchmark 各运行 3 次，全部通过，总耗时约 65 秒；表中各指标取 3 次结果的中位数。100ms 是短时测量，结果仅供参考；稳定对比建议使用 `-benchtime=1s -count=5`，并保持机器负载一致。这些 benchmark 为串行执行，不代表并发吞吐量。

测试数据由 `jsonpath_benchmark_test.go` 确定性生成，包含文档元信息和 10 或 1000 本书，每本书包含标题、价格、可用状态、作者及标签。数据生成不计入耗时。

### 编译与解析

| 操作              | 耗时（µs/op） |    B/op | allocs/op |
| ----------------- | ------------: | ------: | --------: |
| Compile：字段查询 |         0.089 |     288 |         5 |
| Compile：组合过滤 |         0.508 |    1928 |        21 |
| Compile：正则过滤 |         1.009 |    2968 |        50 |
| Parse：10 本书    |         7.356 |   25160 |       219 |
| Parse：1000 本书  |       745.722 | 2382233 |     20115 |

### 查询

以下均使用 1000 本书的数据。`OneShot` 调用 `QueryValues` 或 `QueryNodes`，包含编译和解析；`Reuse` 在计时前完成 Compile 和 Parse，仅测量求值及结果构造。Values 返回 JSON 片段，Nodes 额外构造 normalized path。

| 场景       | 模式    | 结果   | 耗时（µs/op） |    B/op | allocs/op |
| ---------- | ------- | ------ | ------------: | ------: | --------: |
| 字段读取   | OneShot | Values |       772.231 | 2382554 |     20123 |
| 字段读取   | Reuse   | Values |         0.046 |      40 |         3 |
| 字段读取   | Reuse   | Nodes  |         0.120 |     120 |         6 |
| 批量标题   | OneShot | Values |       827.599 | 2448939 |     21137 |
| 批量标题   | OneShot | Nodes  |       928.323 | 2528524 |     25037 |
| 批量标题   | Reuse   | Values |        31.009 |   66232 |      1015 |
| 批量标题   | Reuse   | Nodes  |       124.945 |  145817 |      4915 |
| 组合过滤   | OneShot | Values |       847.849 | 2388242 |     20203 |
| 组合过滤   | Reuse   | Values |        38.712 |    4088 |        67 |
| 正则过滤   | OneShot | Values |      1467.095 | 3414767 |     44860 |
| 正则过滤   | Reuse   | Values |       680.476 | 1029573 |     24695 |
| 递归查找   | OneShot | Values |       894.306 | 2458020 |     21141 |
| 递归查找   | Reuse   | Values |        82.115 |   75632 |      1023 |
| 递归查找   | Reuse   | Nodes  |       210.502 |  212818 |      5823 |
| 过滤无匹配 | Reuse   | Values |        29.413 |      16 |         2 |

对应表达式：

- 字段读取：`$.meta.total`
- 批量标题：`$.books[*].title`
- 组合过滤：`$.books[?@.price < 10 && @.available == true].title`
- 正则过滤：`$.books[?match(@.title, "Book [0-9]+")].title`
- 递归查找：`$..name`
- 过滤无匹配：`$.books[?@.price < 0].title`

重复查询同一文档时，复用 Query 和 Document 可以避免重复编译与解析；不需要节点路径时优先使用 Values。即使复用，结果片段仍会独立分配，批量返回和路径构造的成本不会被消除。
