package jsonpath_test

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

func benchmarkDocument(size int) ([]byte, map[string]int) {
	rng := rand.New(rand.NewSource(int64(0xC0FFEE ^ size)))
	categories := []string{"reference", "fiction", "biography", "technical", "poetry"}
	tags := []string{"sale", "new", "rare", "signed", "used"}
	counts := map[string]int{
		"child":            1,
		"author_wildcard":  size,
		"descendant_price": size + 1,
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
		}
		if strings.Contains(title, "Number 1") {
			counts["filter_search"]++
		}
	}
	out.WriteString(`],"bicycle":{"color":"red","price":399.0}}}`)
	return []byte(out.String()), counts
}

var benchmarkQueries = []struct {
	name string
	expr string
}{
	{"child", `$.store.book[0].title`},
	{"author_wildcard", `$.store.book[*].author`},
	{"descendant_price", `$..price`},
	{"nested_wildcard", `$.store.book[*].reviews[*].rating`},
	{"filter_cheap", `$.store.book[?@.price < 10]`},
	{"filter_selective", `$.store.book[0].reviews[?@.rating >= 4]`},
	{"filter_search", `$.store.book[?search(@.title, "Number 1")]`},
}

// Exclude fixture generation, document parsing, and query compilation to match
// upstream's eval benchmarks. Values includes this API's result-copying cost.
func BenchmarkQuery(b *testing.B) {
	for _, size := range []int{25000, 100000} {
		b.Run(fmt.Sprintf("Books%d", size), func(b *testing.B) {
			data, counts := benchmarkDocument(size)
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
					values, err := query.Values(doc)
					if err != nil || len(values) != counts[tc.name] {
						b.Fatalf("selection returned %d matches, want %d; error: %v", len(values), counts[tc.name], err)
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
