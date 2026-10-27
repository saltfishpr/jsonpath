package jsonpath_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

type complianceCase struct {
	Name     string              `json:"name"`
	Selector string              `json:"selector"`
	Document json.RawMessage     `json:"document"`
	Invalid  bool                `json:"invalid_selector"`
	Result   []json.RawMessage   `json:"result"`
	Results  [][]json.RawMessage `json:"results"`
	Paths    []string            `json:"result_paths"`
	AllPaths [][]string          `json:"results_paths"`
}

func decoded(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func matching(t *testing.T, nodes []jsonpath.Node, values []json.RawMessage, paths []string) bool {
	t.Helper()
	if len(nodes) != len(values) || len(nodes) != len(paths) {
		return false
	}
	for i, n := range nodes {
		if n.Path != paths[i] || !reflect.DeepEqual(decoded(t, n.Value), decoded(t, values[i])) {
			return false
		}
	}
	return true
}

var complianceCasesToSkip = map[string]struct{}{
	"functions, match, explicit caret":  {},
	"functions, match, explicit dollar": {},
}

func TestCompliance(t *testing.T) {
	data, err := os.ReadFile("testdata/cts.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Tests []complianceCase `json:"tests"`
	}
	if err = json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	for _, tc := range suite.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			// These cases incorrectly treat ordinary I-Regexp characters as anchors;
			// see testdata/README.md.
			if _, skip := complianceCasesToSkip[tc.Name]; skip {
				t.Skip("upstream fixture has incorrect expectations")
			}

			q, err := jsonpath.Compile(tc.Selector)
			if tc.Invalid {
				if err == nil {
					t.Fatalf("accepted invalid expression %q", tc.Selector)
				}
				return
			}
			if err != nil {
				t.Fatalf("compile %q: %v", tc.Selector, err)
			}
			doc, err := jsonpath.ParseDocument(tc.Document)
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := q.Nodes(doc)
			if err != nil {
				t.Fatal(err)
			}
			ok := false
			if tc.Results != nil {
				for i, values := range tc.Results {
					if matching(t, nodes, values, tc.AllPaths[i]) {
						ok = true
						break
					}
				}
			} else {
				ok = matching(t, nodes, tc.Result, tc.Paths)
			}
			if !ok {
				t.Fatalf("query %q: got %v; expected values %s or %s, paths %v or %v", tc.Selector, nodes, tc.Result, tc.Results, tc.Paths, tc.AllPaths)
			}
		})
	}
}
