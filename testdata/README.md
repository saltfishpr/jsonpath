# 官方 compliance suite

本目录的 `cts.json` 来自 [jsonpath-compliance-test-suite](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite)，固定在上游提交 `20ebdcce5008c8a777ee9c80ad853ecf606950b5`：

https://raw.githubusercontent.com/jsonpath-standard/jsonpath-compliance-test-suite/20ebdcce5008c8a777ee9c80ad853ecf606950b5/cts.json

许可证与版权声明见同目录的 `CTS-LICENSE` 和 `CTS-NOTICE`。

## 与 RFC 9485 的两处偏差

RFC 9485 §3 的 `NormalChar` 把 `$`（U+0024）和 `^`（U+005E）列为普通字符。§4 规定 `match()` 和 `search()` 采用 XSD 正则语义：`^` 和 `$` 不是锚点，`match()` 的整个模式隐式锚定到完整字符串。

- RFC 9485 §3：https://www.rfc-editor.org/rfc/rfc9485.html#section-3
- RFC 9485 §4：https://www.rfc-editor.org/rfc/rfc9485.html#section-4

上游有两条案例把 `^` 和 `$` 当作锚点，与上述规则冲突：

| 上游名称                          | 表达式                  | 上游期望        | RFC 期望 |
| --------------------------------- | ----------------------- | --------------- | -------- |
| functions, match, explicit caret  | `$[?match(@, '^ab.*')]` | `"abc"`, `"ab"` | 空       |
| functions, match, explicit dollar | `$[?match(@, '.*bc$')]` | `"abc"`         | 空       |

本包按 RFC 实现，这两条在上游属于错误期望。上游 [issue #122](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite/issues/122) 跟踪该问题，RFC 9485 作者已在[评论](https://github.com/treetrail/treetrail/issues/46#issuecomment-6084436765)中确认 `^` 和 `$` 不是锚点，工作组同意修正 CTS。截至 2026-10，issue 仍未关闭，固定提交 `20ebdcce` 仍然包含这两条案例。

## 测试的处理方式

`compliance_test.go` 按案例名称跳过这两条案例，上游数据保持原样。`jsonpath_test.go` 另有公开 API 案例覆盖同一语义：`^ab.*` 匹配 `^abc`、不匹配 `abc`，`.*bc$` 匹配 `abc$`。上游修复后删除这个 skip 即可。

其余上游案例由 runner 完整执行：结果值与结果路径按索引一一配对校验，`Values` 与 `Nodes` 的结果保持一致，`invalid_selector` 案例必须在编译阶段失败。
