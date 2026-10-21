package jsonpath

import "testing"

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		name, input, digits, exponent string
		negative                      bool
	}{
		{"zero", "0", "0", "0", false},
		{"negative_zero", "-0.00e999999999999999999999", "0", "0", false},
		{"integer", "123", "123", "0", false},
		{"negative", "-123", "123", "0", true},
		{"integer_trailing_zeros", "1200", "12", "2", false},
		{"fraction_leading_zeros", "0.00120", "12", "-4", false},
		{"fraction_trailing_zeros", "12.300", "123", "-1", false},
		{"positive_exponent", "1.2300e+04", "123", "2", false},
		{"negative_exponent", "-120.0E-3", "12", "-2", true},
		{"huge_exponent", "10e9223372036854775807", "1", "9223372036854775808", false},
		{"tiny_exponent", "0.1e-9223372036854775808", "1", "-9223372036854775809", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDecimal(tc.input)
			if got.negative != tc.negative || got.digits != tc.digits || got.exponent.String() != tc.exponent {
				t.Fatalf("parseDecimal(%q) = (negative=%t, digits=%q, exponent=%s), want (%t, %q, %s)",
					tc.input, got.negative, got.digits, &got.exponent, tc.negative, tc.digits, tc.exponent)
			}
		})
	}
}

func TestCompareNumber(t *testing.T) {
	cases := []struct {
		name, a, b string
		want       int
	}{
		{"zero_forms", "-0.0", "0e999999999999999999999", 0},
		{"zero_before_positive", "0", "0.001", -1},
		{"negative_before_zero", "-0.001", "0", -1},
		{"opposite_signs", "-100", "0.001", -1},
		{"positive_magnitude", "9", "10", -1},
		{"negative_magnitude", "-10", "-9", -1},
		{"same_magnitude_digits", "12", "13", -1},
		{"negative_same_magnitude", "-13", "-12", -1},
		{"implicit_trailing_zero", "1.2", "1.201", -1},
		{"negative_implicit_trailing_zero", "-1.201", "-1.2", -1},
		{"equivalent_decimal_forms", "0.10", "1e-1", 0},
		{"equivalent_integer_forms", "1000e-1", "100.0", 0},
		{"beyond_float64_precision", "9007199254740992", "9007199254740993", -1},
		{"fraction_precision", "0.1", "0.10000000000000001", -1},
		{"max_safe_magnitude", "1e9223372036854775806", "9e9223372036854775806", -1},
		{"magnitude_overflow", "1e9223372036854775807", "9e9223372036854775806", 1},
		{"digit_count_overflow", "12e9223372036854775806", "9e9223372036854775806", 1},
		{"both_magnitudes_overflow", "1e9223372036854775807", "2e9223372036854775807", -1},
		{"overflow_equal_magnitude", "12e9223372036854775806", "1e9223372036854775807", 1},
		{"minimum_int64_exponent", "1e-9223372036854775808", "2e-9223372036854775808", -1},
		{"below_int64_exponent", "1e-9223372036854775809", "1e-9223372036854775808", -1},
		{"above_int64_exponent", "1e9223372036854775808", "1e9223372036854775807", 1},
		{"huge_positive_exponent", "1e999999999999999999999", "9e999999999999999999998", 1},
		{"huge_negative_exponent", "1e-999999999999999999999", "9e-1000000000000000000000", 1},
		{"huge_equal_values", "10e999999999999999999998", "1e999999999999999999999", 0},
		{"negative_huge_values", "-1e999999999999999999999", "-9e999999999999999999998", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := parseDecimal(tc.a), parseDecimal(tc.b)
			aExponent, bExponent := a.exponent.String(), b.exponent.String()
			if got := compareNumber(a, b); got != tc.want {
				t.Fatalf("compareNumber(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
			if got := compareNumber(b, a); got != -tc.want {
				t.Fatalf("compareNumber(%s, %s) = %d, want %d", tc.b, tc.a, got, -tc.want)
			}
			if got := compareNumber(a, a); got != 0 {
				t.Fatalf("compareNumber(%s, itself) = %d, want 0", tc.a, got)
			}
			if a.exponent.String() != aExponent || b.exponent.String() != bExponent {
				t.Fatal("comparison modified an operand's exponent")
			}
		})
	}
}

func TestScanNumber(t *testing.T) {
	cases := []struct {
		name, input string
		start, end  int
		ok          bool
	}{
		{"zero", "0", 0, 1, true},
		{"negative_zero", "-0", 0, 2, true},
		{"integer", "123", 0, 3, true},
		{"fraction", "-12.34", 0, 6, true},
		{"exponent", "1e10", 0, 4, true},
		{"positive_exponent", "1E+2", 0, 4, true},
		{"negative_exponent", "1e-2", 0, 4, true},
		{"combined", "-1.25E+30", 0, 9, true},
		{"nonzero_offset", "[12.5,0]", 1, 5, true},
		{"delimiter", "123]", 0, 3, true},
		{"whitespace_suffix", "1 ", 0, 1, true},
		// The scanner consumes a number prefix; its caller rejects trailing input.
		{"leading_zero_prefix", "01", 0, 1, true},
		{"literal_suffix", "1true", 0, 1, true},
		{"empty", "", 0, 0, false},
		{"end_of_input", "[", 1, 1, false},
		{"sign_only", "-", 0, 1, false},
		{"leading_plus", "+1", 0, 0, false},
		{"leading_dot", ".1", 0, 0, false},
		{"negative_leading_dot", "-.1", 0, 1, false},
		{"missing_fraction", "1.", 0, 2, false},
		{"missing_fraction_before_exponent", "1.e2", 0, 2, false},
		{"missing_exponent", "1e", 0, 2, false},
		{"missing_positive_exponent", "1e+", 0, 3, false},
		{"missing_negative_exponent", "1e-", 0, 3, false},
		{"invalid_exponent", "1ex", 0, 2, false},
		{"leading_whitespace", " 1", 0, 0, false},
		{"literal", "null", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := tc.start
			got := scanNumber(tc.input, &pos)
			if got != tc.ok || pos != tc.end {
				t.Fatalf("scanNumber(%q, %d) = (%t, %d), want (%t, %d)",
					tc.input, tc.start, got, pos, tc.ok, tc.end)
			}
		})
	}
}
