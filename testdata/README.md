# 官方 compliance suite

来源：https://github.com/jsonpath-standard/jsonpath-compliance-test-suite

固定提交：`20ebdcce5008c8a777ee9c80ad853ecf606950b5`

数据：https://raw.githubusercontent.com/jsonpath-standard/jsonpath-compliance-test-suite/20ebdcce5008c8a777ee9c80ad853ecf606950b5/cts.json

## 两项上游 bug

RFC 9485 §3 的 `NormalChar` 把 `$`（U+0024）和 `^`（U+005E）列为普通字符；§4 以 XSD 正则为语义基准，`^` 和 `$` 没有锚点含义，整个模式隐式锚定到整串。

- RFC 9485 §3：https://www.rfc-editor.org/rfc/rfc9485.html#section-3
- RFC 9485 §4：https://www.rfc-editor.org/rfc/rfc9485.html#section-4

以下两条案例把 `^` 和 `$` 当锚点，与 RFC 不符：

| 上游名称                          | 表达式                  | 上游期望        | RFC 期望 |
| --------------------------------- | ----------------------- | --------------- | -------- |
| functions, match, explicit caret  | `$[?match(@, '^ab.*')]` | `"abc"`, `"ab"` | 空       |
| functions, match, explicit dollar | `$[?match(@, '.*bc$')]` | `"abc"`         | 空       |

这是 CTS 上游的错误期望，不是本包与标准冲突。[issue #122](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite/issues/122) 正在跟踪此问题；RFC 9485 作者 [确认](https://github.com/treetrail/treetrail/issues/46#issuecomment-6084436765) `^` 和 `$` 不是锚点，工作组同意修正 CTS。截至 2025-10，issue 仍未关闭，固定版本 `20ebdcce` 仍包含这两条错误期望。

`compliance_test.go` 按案例名称跳过这两条错误案例，不修改上游数据；公开 API 集成案例另验证 `^abc` 和 `abc$` 能正向匹配。上游修复后即可移除此 skip。runner 还按索引配对校验结果和路径，并要求 `invalid_selector` 在编译阶段失败。
