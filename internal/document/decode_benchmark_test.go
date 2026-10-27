package document

import (
	"fmt"
	"strings"
	"testing"
)

// 各 fixture 都是确定性生成的，避免基准之间因为数据差异互相干扰。
func fixtureObject(members int) []byte {
	var sb strings.Builder
	sb.Grow(members * 16)
	sb.WriteByte('{')
	for i := 0; i < members; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"key-%04d":%d`, i, i*7)
	}
	sb.WriteByte('}')
	return []byte(sb.String())
}

func fixtureNumberArray(n int) []byte {
	var sb strings.Builder
	sb.Grow(n * 8)
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `%d.%02d`, i, i%100)
	}
	sb.WriteByte(']')
	return []byte(sb.String())
}

func fixtureStringArray(n int) []byte {
	var sb strings.Builder
	sb.Grow(n * 24)
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"item number %d"`, i)
	}
	sb.WriteByte(']')
	return []byte(sb.String())
}

func fixtureEscapedStringArray(n int) []byte {
	var sb strings.Builder
	sb.Grow(n * 32)
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"a\nb\tc\u00e9d\\%d"`, i)
	}
	sb.WriteByte(']')
	return []byte(sb.String())
}

func fixtureNested(depth int) []byte {
	return []byte(strings.Repeat(`{"k":`, depth) + "1" + strings.Repeat("}", depth))
}

func fixtureMixed(n int) []byte {
	colors := []string{`"red"`, `"green"`, `"blue"`}
	var sb strings.Builder
	sb.Grow(n * 160)
	sb.WriteString(`{"store":{"book":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb,
			`{"id":%d,"category":%s,"title":"Book Number %d","isbn":"%d-%06d-x",`+
				`"price":%d.99,"in_stock":%t,"tags":["sale","new"],`+
				`"reviews":[{"user":"u%d","rating":%d}]}`,
			i, colors[i%len(colors)], i, i%10, i, i%50, i%3 != 0, i%97, i%5+1)
	}
	sb.WriteString(`],"bicycle":{"color":"red","price":399.00}}}`)
	return []byte(sb.String())
}

// BenchmarkParse 覆盖不同文档形态：解析器的瓶颈随标量种类、转义、嵌套深度
// 而不同，单一形态的数字不足以判断优化是否有效。
func BenchmarkParse(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{name: "scalar", data: []byte(`1`)},
		{name: "object/1000", data: fixtureObject(1000)},
		{name: "array/numbers/1000", data: fixtureNumberArray(1000)},
		{name: "array/strings/1000", data: fixtureStringArray(1000)},
		{name: "array/escapes/1000", data: fixtureEscapedStringArray(1000)},
		{name: "nested/100", data: fixtureNested(100)},
		{name: "nested/1000", data: fixtureNested(1000)},
		{name: "mixed/100", data: fixtureMixed(100)},
		{name: "mixed/1000", data: fixtureMixed(1000)},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.data)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Parse(tc.data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
