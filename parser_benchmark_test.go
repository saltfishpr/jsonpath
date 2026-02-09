package jsonpath

import (
	"testing"
)

// BenchmarkParseRoot 测试根标识符解析
func BenchmarkParseRoot(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$")
	}
}

// BenchmarkParseSimpleDot 测试简单点表示法
func BenchmarkParseSimpleDot(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.foo")
	}
}

// BenchmarkParseNestedDot 测试嵌套点表示法
func BenchmarkParseNestedDot(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.store.book.author")
	}
}

// BenchmarkParseDeepNesting 测试深度嵌套路径
func BenchmarkParseDeepNesting(b *testing.B) {
	path := "$.a.b.c.d.e.f.g.h.i.j"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(path)
	}
}

// BenchmarkParseBracketName 测试括号表示法名称
func BenchmarkParseBracketName(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$['foo']")
	}
}

// BenchmarkParseBracketMultipleNames 测试多个名称选择器
func BenchmarkParseBracketMultipleNames(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$['foo','bar','baz']")
	}
}

// BenchmarkParseBracketManyNames 测试大量名称选择器
func BenchmarkParseBracketManyNames(b *testing.B) {
	path := "$['a','b','c','d','e','f','g','h','i','j']"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(path)
	}
}

// BenchmarkParseWildcard 测试通配符选择器
func BenchmarkParseWildcard(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.*")
	}
}

// BenchmarkParseIndex 测试索引选择器
func BenchmarkParseIndex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[0]")
	}
}

// BenchmarkParseNegativeIndex 测试负索引选择器
func BenchmarkParseNegativeIndex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[-1]")
	}
}

// BenchmarkParseMultipleIndexes 测试多个索引选择器
func BenchmarkParseMultipleIndexes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[0,1,2,3,4]")
	}
}

// BenchmarkParseSlice 测试切片选择器
func BenchmarkParseSlice(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[0:10]")
	}
}

// BenchmarkParseSliceWithStep 测试带步长的切片选择器
func BenchmarkParseSliceWithStep(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[0:10:2]")
	}
}

// BenchmarkParseSliceReverse 测试反向切片
func BenchmarkParseSliceReverse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[::-1]")
	}
}

// BenchmarkParseDescendant 测试后代段
func BenchmarkParseDescendant(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$..foo")
	}
}

// BenchmarkParseDescendantWildcard 测试后代通配符
func BenchmarkParseDescendantWildcard(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$..*")
	}
}

// BenchmarkParseDescendantDeep 测试深度后代查询
func BenchmarkParseDescendantDeep(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$..book..author")
	}
}

// BenchmarkParseFilterSimpleExistence 测试简单存在性过滤器
func BenchmarkParseFilterSimpleExistence(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.foo]")
	}
}

// BenchmarkParseFilterComparison 测试比较过滤器
func BenchmarkParseFilterComparison(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.price < 10]")
	}
}

// BenchmarkParseFilterLogicalAnd 测试逻辑与过滤器
func BenchmarkParseFilterLogicalAnd(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.age > 18 && @.age < 65]")
	}
}

// BenchmarkParseFilterLogicalOr 测试逻辑或过滤器
func BenchmarkParseFilterLogicalOr(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.type == 'A' || @.type == 'B']")
	}
}

// BenchmarkParseFilterLogicalNot 测试逻辑非过滤器
func BenchmarkParseFilterLogicalNot(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?!@.deleted]")
	}
}

// BenchmarkParseFilterComplexLogical 测试复杂逻辑表达式
func BenchmarkParseFilterComplexLogical(b *testing.B) {
	path := "$[?(@.x > 0 && @.y < 10) || (@.z == 'test' && !@.disabled)]"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(path)
	}
}

// BenchmarkParseFilterNested 测试嵌套过滤器
func BenchmarkParseFilterNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.items[?@.values[?@.active == true]]")
	}
}

// BenchmarkParseFunction 测试函数表达式
func BenchmarkParseFunction(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?length(@.name) > 5]")
	}
}

// BenchmarkParseFunctionMultipleArgs 测试多参数函数
func BenchmarkParseFunctionMultipleArgs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?match(@.date, '^[0-9]{4}-[0-9]{2}')]")
	}
}

// BenchmarkParseFunctionNested 测试嵌套函数调用
func BenchmarkParseFunctionNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?length(trim(@.name)) > 0]")
	}
}

// BenchmarkParseMixedSelectors 测试混合选择器
func BenchmarkParseMixedSelectors(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$['name',*,0:5,-1,?@.active]")
	}
}

// BenchmarkParseRFCExample1 测试 RFC 示例: $.store.book[*].author
func BenchmarkParseRFCExample1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.store.book[*].author")
	}
}

// BenchmarkParseRFCExample2 测试 RFC 示例: $..author
func BenchmarkParseRFCExample2(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$..author")
	}
}

// BenchmarkParseRFCExample3 测试 RFC 示例: $.store..price
func BenchmarkParseRFCExample3(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$.store..price")
	}
}

// BenchmarkParseRFCExample4 测试 RFC 示例: $..book[?@.price<10]
func BenchmarkParseRFCExample4(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$..book[?@.price<10]")
	}
}

// BenchmarkParseSingularQueryRelative 测试相对单值查询
func BenchmarkParseSingularQueryRelative(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.x == @.y]")
	}
}

// BenchmarkParseSingularQueryAbsolute 测试绝对单值查询
func BenchmarkParseSingularQueryAbsolute(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.x == $.max.value]")
	}
}

// BenchmarkParseSingularQueryNested 测试嵌套单值查询
func BenchmarkParseSingularQueryNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[?@.items[0].price > $.threshold]")
	}
}

// BenchmarkParseComplexRealWorld 测试真实世界复杂查询
func BenchmarkParseComplexRealWorld(b *testing.B) {
	path := "$.data.items[?(@.status == 'active' && @.value > 100)].details[*].name"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(path)
	}
}

// BenchmarkParseWithWhitespace 测试带空格的路径
func BenchmarkParseWithWhitespace(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("$[ ? ( @.x > 0 ) && ( @.y < 10 ) ]")
	}
}
