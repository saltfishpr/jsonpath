package jsonpath_test

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

var benchmarkSizes = []int{10, 100, 1000}

func benchmarkDocument(size int) ([]byte, map[string]int) {
	rng := rand.New(rand.NewSource(int64(0xC0FFEE ^ size)))
	categories := []string{"reference", "fiction", "biography", "technical", "poetry"}
	tags := []string{"sale", "new", "rare", "signed", "used"}
	counts := map[string]int{
		"root":             1,
		"nested_field":     1,
		"child":            1,
		"negative_index":   1,
		"slice":            10,
		"slice_step":       (size + 1) / 2,
		"index_union":      3,
		"objects":          size,
		"author_wildcard":  size,
		"descendant_price": size + 1,
		"nested_wildcard":  0,
		"filter_cheap":     0,
		"filter_selective": 0,
		"filter_search":    0,
		"filter_compound":  0,
		"filter_length":    0,
		"filter_match":     size,
		"missing_field":    0,
		"filter_no_match":  0,
	}
	var out strings.Builder
	out.WriteString(`{"store":{"book":[`)
	for i := 0; i < size; i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		reviewCount := rng.Intn(4)
		author := rng.Intn(500) + 1
		title := fmt.Sprintf("Book Number %d", i)
		isbn := fmt.Sprintf("%d-%d-%06d-%d", rng.Intn(10), rng.Intn(900)+100, i, rng.Intn(10))
		price := math.RoundToEven((1+rng.Float64()*49)*100) / 100
		inStock := rng.Float64() > 0.2
		fmt.Fprintf(&out, `{"category":%q,"author":"Author %d","title":%q,"isbn":%q,"price":%.2f,"in_stock":%t,"tags":[`, categories[i%len(categories)], author, title, isbn, price, inStock)
		tagCount := rng.Intn(4)
		selection := rng.Perm(len(tags))
		for j := 0; j < tagCount; j++ {
			if j > 0 {
				out.WriteByte(',')
			}
			fmt.Fprintf(&out, `%q`, tags[selection[j]])
		}
		out.WriteString(`],"reviews":[`)
		for j := 0; j < reviewCount; j++ {
			if j > 0 {
				out.WriteByte(',')
			}
			reviewer := rng.Intn(9999) + 1
			rating := rng.Intn(5) + 1
			fmt.Fprintf(&out, `{"reviewer":"User %d","rating":%d}`, reviewer, rating)
			if i == 0 && rating >= 4 {
				counts["filter_selective"]++
			}
		}
		out.WriteString(`]}`)
		counts["nested_wildcard"] += reviewCount
		if price < 10 {
			counts["filter_cheap"]++
			if inStock {
				counts["filter_compound"]++
			}
		}
		if tagCount == 2 {
			counts["filter_length"]++
		}
		if strings.Contains(title, "Number 1") {
			counts["filter_search"]++
		}
	}
	out.WriteString(`],"bicycle":{"color":"red","price":399.0}}}`)
	return []byte(out.String()), counts
}

func BenchmarkParse(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Books%d", size), func(b *testing.B) {
			data, _ := benchmarkDocument(size)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := jsonpath.Parse(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

var benchmarkQueries = []struct {
	name string
	expr string
}{
	{"root", `$`},
	{"nested_field", `$.store.bicycle.color`},
	{"child", `$.store.book[0].title`},
	{"negative_index", `$.store.book[-1].title`},
	{"slice", `$.store.book[0:10].title`},
	{"slice_step", `$.store.book[::2].title`},
	{"index_union", `$.store.book[0,1,-1].title`},
	{"objects", `$.store.book[*]`},
	{"author_wildcard", `$.store.book[*].author`},
	{"descendant_price", `$..price`},
	{"nested_wildcard", `$.store.book[*].reviews[*].rating`},
	{"filter_cheap", `$.store.book[?@.price < 10]`},
	{"filter_selective", `$.store.book[0].reviews[?@.rating >= 4]`},
	{"filter_search", `$.store.book[?search(@.title, "Number 1")]`},
	{"filter_compound", `$.store.book[?@.price < 10 && @.in_stock == true].title`},
	{"filter_length", `$.store.book[?length(@.tags) == 2].title`},
	{"filter_match", `$.store.book[?match(@.title, "Book Number [0-9]+")].title`},
	{"missing_field", `$.store.missing`},
	{"filter_no_match", `$.store.book[?@.price < 0].title`},
}

func BenchmarkCompile(b *testing.B) {
	for _, tc := range benchmarkQueries {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := jsonpath.Compile(tc.expr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Exclude fixture generation, document parsing, and query compilation to match
// upstream's eval benchmarks. Values includes this API's result-copying cost.
func BenchmarkQuery(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Books%d", size), func(b *testing.B) {
			data, counts := benchmarkDocument(size)
			doc, err := jsonpath.Parse(data)
			if err != nil {
				b.Fatal(err)
			}
			for _, tc := range benchmarkQueries {
				b.Run(tc.name, func(b *testing.B) {
					want, ok := counts[tc.name]
					if !ok {
						b.Fatalf("missing expected count for %s", tc.name)
					}
					query, err := jsonpath.Compile(tc.expr)
					if err != nil {
						b.Fatal(err)
					}
					values, err := query.Values(doc)
					if err != nil || len(values) != want {
						b.Fatalf("selection returned %d matches, want %d; error: %v", len(values), want, err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if _, err := query.Values(doc); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
