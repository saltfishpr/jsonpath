package jsonpath_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

// Keep fixtures deterministic and build them outside the timed regions.
func benchmarkDocument(size int) []byte {
	var out strings.Builder
	out.WriteString(`{"meta":{"total":`)
	fmt.Fprintf(&out, `%d},"books":[`, size)
	for i := 0; i < size; i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(&out, `{"title":"Book %d","price":%d,"available":%t,"author":{"name":"Author %d"},"tags":["fiction","paperback"]}`, i, i%100, i%2 == 0, i%10)
	}
	out.WriteString(`]}`)
	return []byte(out.String())
}

var benchmarkQueries = []struct {
	name string
	expr string
	want func(int) int
}{
	{"Root", `$`, func(n int) int { return 1 }},
	{"Field", `$.meta.total`, func(n int) int { return 1 }},
	{"NestedField", `$.books[0].author.name`, func(n int) int { return 1 }},
	{"NegativeIndex", `$.books[-1].title`, func(n int) int { return 1 }},
	{"Slice", `$.books[0:10].title`, func(n int) int { return 10 }},
	{"Wildcard", `$.books[*].title`, func(n int) int { return n }},
	{"Objects", `$.books[*]`, func(n int) int { return n }},
	{"Filter", `$.books[?@.price < 10].title`, func(n int) int { return (n/100)*10 + minBenchmark(n%100, 10) }},
	{"CompoundFilter", `$.books[?@.price < 10 && @.available == true].title`, func(n int) int { return (n/100)*5 + (minBenchmark(n%100, 10)+1)/2 }},
	{"Length", `$.books[?length(@.tags) == 2].title`, func(n int) int { return n }},
	{"Regexp", `$.books[?match(@.title, "Book [0-9]+")].title`, func(n int) int { return n }},
	{"Descendant", `$..name`, func(n int) int { return n }},
	{"MissingField", `$.meta.missing`, func(n int) int { return 0 }},
	{"NoFilterMatches", `$.books[?@.price < 0].title`, func(n int) int { return 0 }},
}

func minBenchmark(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BenchmarkCompile(b *testing.B) {
	for _, tc := range benchmarkQueries {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := jsonpath.Compile(tc.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	for _, size := range []int{10, 1000} {
		b.Run(fmt.Sprintf("Books%d", size), func(b *testing.B) {
			data := benchmarkDocument(size)
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := jsonpath.Parse(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// OneShot includes compilation and parsing; Reuse evaluates an already compiled
// query against an already parsed document, isolating selection and result copying.
func BenchmarkQuery(b *testing.B) {
	for _, size := range []int{10, 1000} {
		b.Run(fmt.Sprintf("Books%d", size), func(b *testing.B) {
			data := benchmarkDocument(size)
			doc, err := jsonpath.Parse(data)
			if err != nil {
				b.Fatal(err)
			}
			for _, tc := range benchmarkQueries {
				b.Run(tc.name, func(b *testing.B) {
					query, err := jsonpath.Compile(tc.expr)
					if err != nil {
						b.Fatal(err)
					}
					want := tc.want(size)
					for _, mode := range []string{"OneShot", "Reuse"} {
						b.Run(mode, func(b *testing.B) {
							b.Run("Values", func(b *testing.B) {
								run := func() (int, error) {
									if mode == "OneShot" {
										values, err := jsonpath.QueryValues(tc.expr, data)
										return len(values), err
									}
									values, err := query.Values(doc)
									return len(values), err
								}
								benchmarkSelection(b, run, want)
							})
							b.Run("Nodes", func(b *testing.B) {
								run := func() (int, error) {
									if mode == "OneShot" {
										nodes, err := jsonpath.QueryNodes(tc.expr, data)
										return len(nodes), err
									}
									nodes, err := query.Nodes(doc)
									return len(nodes), err
								}
								benchmarkSelection(b, run, want)
							})
						})
					}
				})
			}
		})
	}
}

func benchmarkSelection(b *testing.B, run func() (int, error), want int) {
	b.Helper()
	if got, err := run(); err != nil || got != want {
		b.Fatalf("selection returned %d matches, want %d; error: %v", got, want, err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := run(); err != nil {
			b.Fatal(err)
		}
	}
}
