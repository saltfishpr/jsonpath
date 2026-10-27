# 官方 compliance suite

`cts.json` 取自 [jsonpath-compliance-test-suite](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite)，固定在上游提交 `20ebdcce5008c8a777ee9c80ad853ecf606950b5`（[原始文件](https://raw.githubusercontent.com/jsonpath-standard/jsonpath-compliance-test-suite/20ebdcce5008c8a777ee9c80ad853ecf606950b5/cts.json)）。许可证与版权声明见同目录的 `CTS-LICENSE`、`CTS-NOTICE`。

## 与 RFC 9485 的两处偏差

- **无锚点**：[RFC 9485 §3](https://www.rfc-editor.org/rfc/rfc9485.html#section-3) 将 `^`（U+005E）和 `$`（U+0024）列为 `NormalChar`；[§4](https://www.rfc-editor.org/rfc/rfc9485.html#section-4) 规定 I-Regexp 遵循 XSD 正则语义，其中没有锚点。
- **匹配语义**：[RFC 9535 §2.4.6](https://www.rfc-editor.org/rfc/rfc9535.html#section-2.4.6) 要求 `match()` 匹配完整字符串，[§2.4.7](https://www.rfc-editor.org/rfc/rfc9535.html#section-2.4.7) 要求 `search()` 查找子串。

上游有两条案例把 `^`、`$` 当作锚点，与上述规则冲突：

| 上游名称                          | 表达式                  | 上游期望        | RFC 期望 |
| --------------------------------- | ----------------------- | --------------- | -------- |
| functions, match, explicit caret  | `$[?match(@, '^ab.*')]` | `"abc"`, `"ab"` | 空       |
| functions, match, explicit dollar | `$[?match(@, '.*bc$')]` | `"abc"`         | 空       |

本包按 RFC 实现，这两条属于上游的错误期望。上游 [issue #122](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite/issues/122) 引用了 RFC 9485 作者对二者不是锚点的确认，但截至 2026-10 仍未关闭，固定提交 `20ebdcce` 仍包含这两条案例。

## 测试的处理方式

- `jsonpath_compliance_test.go` 的 `complianceCasesToSkip` 按名称跳过这两条，上游数据保持原样。
- `jsonpath_test.go` 的 `TestQueryValues_Functions/match/literal_anchors` 通过公开 API `QueryValues` 覆盖同一语义：`^a$` 匹配 `"^a$"`，不匹配 `"a"`。
- 上游修复后，应先更新 `cts.json` 的固定版本和本文中的提交信息，核对修正内容，再移除对应的 skip。
