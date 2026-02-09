package jsonpath

import (
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// I-JSON 安全整数范围
const (
	MinSafeInteger = -(2<<52 - 1)
	MaxSafeInteger = 2<<52 - 1
)

// parseValue parses the json and returns a result.
//
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return back unexpected results.
// If you are consuming JSON from an unpredictable source then you may want to
// use the Valid function first.
func parseValue(json string) Result {
	var value Result
	i := 0
	for ; i < len(json); i++ {
		if json[i] == '{' || json[i] == '[' {
			value.Type = JSONTypeJSON
			value.Raw = json[i:] // just take the entire raw
			break
		}
		if json[i] <= ' ' {
			continue
		}
		switch json[i] {
		case '+', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'i', 'I', 'N':
			value.Type = JSONTypeNumber
			value.Raw, value.Num = tonum(json[i:])
		case 'n':
			if i+1 < len(json) && json[i+1] != 'u' {
				// nan
				value.Type = JSONTypeNumber
				value.Raw, value.Num = tonum(json[i:])
			} else {
				// null
				value.Type = JSONTypeNull
				value.Raw = tolit(json[i:])
			}
		case 't':
			value.Type = JSONTypeTrue
			value.Raw = tolit(json[i:])
		case 'f':
			value.Type = JSONTypeFalse
			value.Raw = tolit(json[i:])
		case '"':
			value.Type = JSONTypeString
			value.Raw, value.Str = tostr(json[i:])
		default:
			return Result{}
		}
		break
	}
	if value.Exists() {
		value.Index = i
	}
	return value
}

func tonum(json string) (raw string, num float64) {
	for i := 1; i < len(json); i++ {
		// less than dash might have valid characters
		if json[i] <= '-' {
			if json[i] <= ' ' || json[i] == ',' {
				// break on whitespace and comma
				raw = json[:i]
				num, _ = strconv.ParseFloat(raw, 64)
				return
			}
			// could be a '+' or '-'. let's assume so.
		} else if json[i] == ']' || json[i] == '}' {
			// break on ']' or '}'
			raw = json[:i]
			num, _ = strconv.ParseFloat(raw, 64)
			return
		}
	}
	raw = json
	num, _ = strconv.ParseFloat(raw, 64)
	return
}

func tolit(json string) (raw string) {
	for i := 1; i < len(json); i++ {
		if json[i] < 'a' || json[i] > 'z' {
			return json[:i]
		}
	}
	return json
}

func tostr(json string) (raw string, str string) {
	// expects that the lead character is a '"'
	for i := 1; i < len(json); i++ {
		if json[i] > '\\' {
			continue
		}
		if json[i] == '"' {
			return json[:i+1], json[1:i]
		}
		if json[i] == '\\' {
			i++
			for ; i < len(json); i++ {
				if json[i] > '\\' {
					continue
				}
				if json[i] == '"' {
					// look for an escaped slash
					if json[i-1] == '\\' {
						n := 0
						for j := i - 2; j > 0; j-- {
							if json[j] != '\\' {
								break
							}
							n++
						}
						if n%2 == 0 {
							continue
						}
					}
					return json[:i+1], unescape(json[1:i])
				}
			}
			var ret string
			if i+1 < len(json) {
				ret = json[:i+1]
			} else {
				ret = json[:i]
			}
			return ret, unescape(json[1:i])
		}
	}
	return json, json[1:]
}

// unescape unescapes a string
func unescape(json string) string {
	str := make([]byte, 0, len(json))
	for i := 0; i < len(json); i++ {
		switch {
		default:
			str = append(str, json[i])
		case json[i] < ' ':
			return string(str)
		case json[i] == '\\':
			i++
			if i >= len(json) {
				return string(str)
			}
			switch json[i] {
			default:
				return string(str)
			case '\\':
				str = append(str, '\\')
			case '/':
				str = append(str, '/')
			case 'b':
				str = append(str, '\b')
			case 'f':
				str = append(str, '\f')
			case 'n':
				str = append(str, '\n')
			case 'r':
				str = append(str, '\r')
			case 't':
				str = append(str, '\t')
			case '"':
				str = append(str, '"')
			case 'u':
				if i+5 > len(json) {
					return string(str)
				}
				r := runeit(json[i+1:])
				i += 5
				if utf16.IsSurrogate(r) {
					// need another code
					if len(json[i:]) >= 6 && json[i] == '\\' &&
						json[i+1] == 'u' {
						// we expect it to be correct so just consume it
						r = utf16.DecodeRune(r, runeit(json[i+2:]))
						i += 6
					}
				}
				// provide enough space to encode the largest utf8 possible
				str = append(str, 0, 0, 0, 0, 0, 0, 0, 0)
				n := utf8.EncodeRune(str[len(str)-8:], r)
				str = str[:len(str)-8+n]
				i-- // backtrack index by one
			}
		}
	}
	return string(str)
}

// runeit returns the rune from the the \uXXXX
func runeit(json string) rune {
	n, _ := strconv.ParseUint(json[:4], 16, 64)
	return rune(n)
}

// skipWhitespaceJSON skips JSON whitespace characters and returns the index of the first non-whitespace character.
func skipWhitespaceJSON(json string, i int) int {
	for i < len(json) {
		ch := json[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
		} else {
			break
		}
	}
	return i
}
