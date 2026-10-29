package document

import (
	"math"
	"math/big"
	"reflect"
	"testing"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		digits       string
		negative     bool
		magnitude    int64
		bigMagnitude string
	}{
		{name: "zero", text: "0"},
		{name: "negative zero", text: "-0"},
		{name: "fractional zero", text: "-0.000"},
		{name: "zero with huge exponent", text: "0e999999999999999999999"},
		{name: "zero with tiny exponent", text: "-0.00e-999999999999999999999"},
		{name: "integer", text: "123", digits: "123", magnitude: 3},
		{name: "negative integer", text: "-123", digits: "123", negative: true, magnitude: 3},
		{name: "integer trailing zeros", text: "12000", digits: "12", magnitude: 5},
		{name: "fraction trailing zeros", text: "1.2300", digits: "123", magnitude: 1},
		{name: "fraction leading zeros", text: "0.0012300", digits: "123", magnitude: -2},
		{name: "internal zeros", text: "100.0200", digits: "10002", magnitude: 3},
		{name: "positive exponent", text: "1.2300e+02", digits: "123", magnitude: 3},
		{name: "negative exponent", text: "-12.30e-2", digits: "123", negative: true, magnitude: 0},
		{name: "uppercase exponent", text: "1E3", digits: "1", magnitude: 4},
		{name: "exponent leading zeros", text: "1e+0002", digits: "1", magnitude: 3},
		{name: "zero exponent", text: "1e-0", digits: "1", magnitude: 1},
		{
			name: "long coefficient", text: "1234567890123456789012345678901234567890",
			digits: "123456789012345678901234567890123456789", magnitude: 40,
		},
		{
			name: "maximum int64 magnitude", text: "1e9223372036854775806",
			digits: "1", magnitude: math.MaxInt64,
		},
		{
			name: "minimum int64 magnitude", text: "0.01e-9223372036854775807",
			digits: "1", magnitude: math.MinInt64,
		},
		{
			name: "positive adjustment overflows", text: "1e9223372036854775807",
			digits: "1", bigMagnitude: "9223372036854775808",
		},
		{
			name: "trailing zeros cause overflow", text: "100e9223372036854775807",
			digits: "1", bigMagnitude: "9223372036854775810",
		},
		{
			name: "negative adjustment overflows", text: "0.01e-9223372036854775808",
			digits: "1", bigMagnitude: "-9223372036854775809",
		},
		{
			name: "huge positive exponent", text: "1e999999999999999999999",
			digits: "1", bigMagnitude: "1000000000000000000000",
		},
		{
			name: "huge negative exponent", text: "1e-999999999999999999999",
			digits: "1", bigMagnitude: "-999999999999999999998",
		},
		{
			name: "huge exponent without adjustment", text: "0.1e999999999999999999999",
			digits: "1", bigMagnitude: "999999999999999999999",
		},
		{
			name: "huge exponent with negative adjustment", text: "0.001e999999999999999999999",
			digits: "1", bigMagnitude: "999999999999999999997",
		},
		{
			name: "positive exponent demotes to int64", text: "0.001e9223372036854775809",
			digits: "1", magnitude: math.MaxInt64,
		},
		{
			name: "negative exponent demotes to int64", text: "1e-9223372036854775809",
			digits: "1", magnitude: math.MinInt64,
		},
		{
			name: "minimum exponent with positive adjustment", text: "1e-9223372036854775808",
			digits: "1", magnitude: math.MinInt64 + 1,
		},
		{
			name: "maximum exponent without adjustment", text: "0.1e9223372036854775807",
			digits: "1", magnitude: math.MaxInt64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseNumber(tt.text)
			if got.negative != tt.negative || got.digits != tt.digits {
				t.Errorf("ParseNumber(%q): negative = %t, digits = %q; want %t, %q",
					tt.text, got.negative, got.digits, tt.negative, tt.digits)
			}
			if tt.bigMagnitude == "" {
				if got.bigMagnitude != nil {
					t.Fatalf("bigMagnitude = %v, want nil", got.bigMagnitude)
				}
				if got.magnitude != tt.magnitude {
					t.Errorf("magnitude = %d, want %d", got.magnitude, tt.magnitude)
				}
			} else {
				if got.bigMagnitude == nil {
					t.Fatalf("bigMagnitude = nil, want %s", tt.bigMagnitude)
				}
				if got.bigMagnitude.String() != tt.bigMagnitude {
					t.Errorf("bigMagnitude = %s, want %s", got.bigMagnitude, tt.bigMagnitude)
				}
				if got.bigMagnitude.IsInt64() {
					t.Error("bigMagnitude fits in int64; want canonical int64 representation")
				}
			}
		})
	}
}

func TestParseNumberEmptyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("ParseNumber(\"\") did not panic")
		}
	}()
	ParseNumber("")
}

func TestNumberZeroValue(t *testing.T) {
	var zero Number
	for _, text := range []string{"0", "-0", "0.000", "-0e999999999999999999999"} {
		n := ParseNumber(text)
		if zero.Cmp(n) != 0 || n.Cmp(zero) != 0 {
			t.Errorf("zero value and ParseNumber(%q) compare unequally", text)
		}
		if n != zero {
			t.Errorf("ParseNumber(%q) = %+v, want zero value", text, n)
		}
	}
	for _, tt := range []struct {
		text string
		want int
	}{
		{text: "-1e-999999999999999999999", want: 1},
		{text: "1e-999999999999999999999", want: -1},
	} {
		n := ParseNumber(tt.text)
		if got := zero.Cmp(n); got != tt.want {
			t.Errorf("zero.Cmp(%s) = %d, want %d", tt.text, got, tt.want)
		}
		if got := n.Cmp(zero); got != -tt.want {
			t.Errorf("%s.Cmp(zero) = %d, want %d", tt.text, got, -tt.want)
		}
	}
}

func TestNumberCmpOrdering(t *testing.T) {
	// Each group contains equivalent spellings; groups are in ascending order.
	groups := [][]string{
		{"-1e999999999999999999999", "-10e999999999999999999998"},
		{"-9e999999999999999999998"},
		{"-10"},
		{"-1.234"},
		{"-1.23", "-123e-2"},
		{"-1e-999999999999999999999"},
		{"0", "-0.00e+99"},
		{"1e-999999999999999999999"},
		{"9e-999999999999999999999"},
		{"1e-999999999999999999998"},
		{"0.1", "1e-1"},
		{"0.100001"},
		{"1.23", "1.2300", "123e-2"},
		{"1.230001"},
		{"1.234"},
		{"9.99"},
		{"10", "1e1"},
		{"9007199254740992"},
		{"9007199254740993"},
		{"1e9223372036854775806"},
		{"1e9223372036854775807"},
		{"1e999999999999999999999", "10e999999999999999999998"},
		{"1.01e999999999999999999999"},
		{"1e1000000000000000000000"},
	}
	for i, group := range groups {
		for j, otherGroup := range groups {
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			for _, a := range group {
				for _, b := range otherGroup {
					if got := ParseNumber(a).Cmp(ParseNumber(b)); got != want {
						t.Errorf("Cmp(%s, %s) = %d, want %d", a, b, got, want)
					}
				}
			}
		}
	}
}

func TestNumberCmpConcurrentCopies(t *testing.T) {
	texts := []string{
		"1e-999999999999999999999",
		"1",
		"1e999999999999999999999",
		"1.01e999999999999999999999",
		"1e1000000000000000000000",
	}
	numbers := make([]Number, len(texts))
	snapshots := make([]Number, len(texts))
	for i, text := range texts {
		numbers[i] = ParseNumber(text)
		snapshots[i] = numbers[i]
		if numbers[i].bigMagnitude != nil {
			snapshots[i].bigMagnitude = new(big.Int).Set(numbers[i].bigMagnitude)
		}
	}
	// Cleanup runs after the parallel subtests so mutations cannot escape the check.
	t.Cleanup(func() {
		if !reflect.DeepEqual(numbers, snapshots) {
			t.Error("Cmp mutated its operands")
		}
	})
	for i := range numbers {
		for j := range numbers {
			i, j := i, j
			t.Run(texts[i]+"/"+texts[j], func(t *testing.T) {
				t.Parallel()
				a, b := numbers[i], numbers[j]
				want := 0
				if i < j {
					want = -1
				} else if i > j {
					want = 1
				}
				for k := 0; k < 100; k++ {
					if got := a.Cmp(b); got != want {
						t.Fatalf("Cmp = %d, want %d", got, want)
					}
				}
				if !reflect.DeepEqual(a, snapshots[i]) || !reflect.DeepEqual(b, snapshots[j]) {
					t.Error("Cmp mutated a copied operand")
				}
			})
		}
	}
}
