package json

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const TRUE_STRING = "true"
const FALSE_STRING = "false"
const NULL_STRING = "null"

var UNICODE_INSUFFICIENT_BYTES = errors.New("failed reading all 4 hex chars for unicode")

func UnmarshalValue(reader *bufio.Reader) (value interface{}, err error) {
	// Unmarshal leading whitespace
	if err = UnmarshalWhitespace(reader); err != nil {
		return nil, fmt.Errorf("failed to Unmarshal leading whitespace: %w", err)
	}
	// Peek at the first rune
	r, _, err := reader.ReadRune()
	if err != nil {
		return nil, fmt.Errorf("failed to read rune: %w", err)
	}
	if err = reader.UnreadRune(); err != nil {
		return nil, fmt.Errorf("failed to unread rune: %w", err)
	}
	// Call correct parsing function depending on the first rune
	if r == '"' {
		value, err = UnmarshalString(reader)
	} else if unicode.IsDigit(r) || r == '-' {
		value, err = UnmarshalNumber(reader)
	} else if r == '{' {
		value, err = UnmarshalObject(reader)
	} else if r == '[' {
		value, err = UnmarshalArray(reader)
	} else if r == 't' {
		value, err = UnmarshalTrue(reader)
	} else if r == 'f' {
		value, err = UnmarshalFalse(reader)
	} else if r == 'n' {
		value, err = UnmarshalNull(reader)
	} else {
		return nil, fmt.Errorf("failed to match value given first char: %c", r)
	}
	// Unmarshal trailing whitespace
	if err := UnmarshalWhitespace(reader); err != nil {
		return nil, fmt.Errorf("failed to Unmarshal trailing whitespace: %w", err)
	}
	return value, err
}

func isJsonWhitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\r' || r == '\t'
}

func UnmarshalWhitespace(reader *bufio.Reader) error {
	eof := false
	for {
		r, _, err := reader.ReadRune()
		if err == io.EOF {
			eof = true
			break
		} else if err != nil {
			return fmt.Errorf("failed to read rune: %w", err)
		}
		if !isJsonWhitespace(r) {
			break
		}
	}
	if !eof {
		if err := reader.UnreadRune(); err != nil {
			return fmt.Errorf("failed to unread rune: %w", err)
		}
	}
	return nil
}

func UnmarshalObject(reader *bufio.Reader) (map[string]interface{}, error) {
	// States
	// 0 start
	// 1 {
	// 2 { ... key
	// 3 { ... key:
	// 4 { ... key:value
	// 5 { ... key:value,
	state := 0
	key := ""
	object := make(map[string]interface{})
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return nil, fmt.Errorf("failed to read rune: %w", err)
		}
		if state == 0 {
			if r == '{' {
				state = 1
			} else {
				return nil, fmt.Errorf("failed to Unmarshal object: no opening {")
			}
		} else if state == 1 || state == 5 {
			if isJsonWhitespace(r) {
				// stay in state 1
			} else if state == 1 && r == '}' {
				break
			} else {
				if err = reader.UnreadRune(); err != nil {
					return nil, fmt.Errorf("failed to unread rune: %w", err)
				}
				key, err = UnmarshalString(reader)
				if err != nil {
					return nil, fmt.Errorf("failed to Unmarshal object key: %w", err)
				}
				state = 2
			}
		} else if state == 2 {
			if isJsonWhitespace(r) {
				// stay in state 2
			} else if r == ':' {
				state = 3
			} else {
				return nil, fmt.Errorf("failed to find matching value for object key: %s", key)
			}
		} else if state == 3 {
			value, err := UnmarshalValue(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to Unmarshal value for object key: %s", key)
			}
			object[key] = value
			state = 4
		} else if state == 4 {
			if r == '}' {
				break
			} else if r == ',' {
				state = 5
			}
		}
	}
	return object, nil
}

func UnmarshalArray(reader *bufio.Reader) ([]interface{}, error) {
	// States
	// 0 start
	// 1 start -> [
	// 2 start -> [ -> 1+ values
	state := 0
	var values []interface{}
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return nil, fmt.Errorf("failed to read rune: %w", err)
		}

		if state == 0 {
			if r == '[' {
				state = 1
			} else {
				return nil, fmt.Errorf("failed to Unmarshal array: no opening [")
			}
		} else if state == 1 {
			if isJsonWhitespace(r) {
				// stay in state 1
			} else if r == ']' {
				break
			} else {
				if err = reader.UnreadRune(); err != nil {
					return nil, fmt.Errorf("failed to unread rune: %w", err)
				}
				value, err := UnmarshalValue(reader)
				if err != nil {
					return nil, fmt.Errorf("failed to Unmarshal array: %w", err)
				}
				values = append(values, value)
				state = 2
			}
		} else if state == 2 {
			if r == ',' {
				state = 1
			} else if r == ']' {
				break
			} else {
				return nil, fmt.Errorf("failed to Unmarshal array: no , or ]")
			}
		}
	}
	return values, nil
}

func UnmarshalNull(reader *bufio.Reader) (interface{}, error) {
	var value [4]byte
	n, err := reader.Read(value[:])
	if err != nil {
		return false, fmt.Errorf("failed to read chars while parsing null: %w", err)
	} else if n != 4 {
		return false, fmt.Errorf("failed to read all 4 chars while parsing null, could only read %d chars", n)
	}
	if string(value[:]) != NULL_STRING {
		return nil, fmt.Errorf("could not Unmarshal null, found: %s", value)
	}
	return nil, nil
}

func UnmarshalTrue(reader *bufio.Reader) (bool, error) {
	var value [4]byte
	n, err := reader.Read(value[:])
	if err != nil {
		return false, fmt.Errorf("failed to read 4 chars while parsing true: %w", err)
	} else if n != len(value) {
		return false, fmt.Errorf("failed to read all 4 chars while parsing true, could only read %d chars", n)
	}
	if string(value[:]) != TRUE_STRING {
		return false, fmt.Errorf("could not Unmarshal true, found: %s", value)
	}
	return true, nil
}

func UnmarshalFalse(reader *bufio.Reader) (bool, error) {
	var value [5]byte
	n, err := reader.Read(value[:])
	if err != nil {
		return false, fmt.Errorf("failed to read 5 chars while parsing false: %w", err)
	} else if n != len(value) {
		return false, fmt.Errorf("failed to read all 5 chars while parsing false, could only read %d chars", n)
	}
	if string(value[:]) != FALSE_STRING {
		return false, fmt.Errorf("could not Unmarshal false, found: %s", value)
	}
	return false, nil
}

func UnmarshalNumber(reader *bufio.Reader) (interface{}, error) {
	// States (https://www.json.org/json-en.html)
	// 0 start
	// 1 start -> -
	// 2 start -> 0
	// 3 start -> digit 1-9
	// 4 digit 1-9 -> digit
	// 5 {2,3,4} -> .
	// 6 5 -> digit
	// 7 {2,3,4,6} -> e|E
	// 8 7 -> -|+
	// 9 exponential digit
	// end with {2,3,4,6,9} -> eof|*
	state := 0
	eof := false
	validEnd := false
	invalidTransition := false
	var numberBuf strings.Builder
	for {
		r, _, err := reader.ReadRune()
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return 0, fmt.Errorf("failed to read rune: %w", err)
		}
		isDigit := unicode.IsDigit(r)

		switch state {
		case 0:
			if eof {
				invalidTransition = true
			} else if r == '-' {
				state = 1
			} else if r == '0' {
				state = 2
			} else if isDigit {
				state = 3
			} else {
				invalidTransition = true
			}
		case 1:
			if eof {
				invalidTransition = true
			} else if r == '0' {
				state = 2
			} else if isDigit {
				state = 3
			} else {
				invalidTransition = true
			}
		case 2:
			if eof {
				validEnd = true
			} else if r == '.' {
				state = 5
			} else if r == 'e' || r == 'E' {
				state = 7
			} else {
				validEnd = true
			}
		case 3:
			fallthrough
		case 4:
			if eof {
				validEnd = true
			} else if isDigit {
				state = 4
			} else if r == '.' {
				state = 5
			} else if r == 'e' || r == 'E' {
				state = 7
			} else {
				validEnd = true
			}
		case 5:
			if eof {
				invalidTransition = true
			} else if isDigit {
				state = 6
			} else {
				invalidTransition = true
			}
		case 6:
			if eof {
				validEnd = true
			} else if isDigit {
				state = 6
			} else if r == 'e' || r == 'E' {
				state = 7
			} else {
				validEnd = true
			}
		case 7:
			if eof {
				invalidTransition = true
			} else if r == '-' || r == '+' {
				state = 8
			} else if isDigit {
				state = 9
			} else {
				invalidTransition = true
			}
		case 8:
			if eof {
				invalidTransition = true
			} else if isDigit {
				state = 9
			} else {
				invalidTransition = true
			}
		case 9:
			if eof || !isDigit {
				validEnd = true
			}
		}
		if !validEnd {
			numberBuf.WriteRune(r)
		}
		if validEnd || invalidTransition {
			break
		}
	}
	if invalidTransition {
		return 0, fmt.Errorf("invalid char in number: %s", numberBuf.String())
	}
	if !eof {
		if err := reader.UnreadRune(); err != nil {
			return 0, fmt.Errorf("failed to unread rune: %w", err)
		}
	}
	return convertToNumber(numberBuf.String())
}

func convertToNumber(numberString string) (interface{}, error) {
	if strings.ContainsAny(numberString, ".eE") {
		float64Value, err := strconv.ParseFloat(numberString, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to Unmarshal float number: %w", err)
		}
		return float64Value, nil
	}
	int64Value, err := strconv.ParseInt(numberString, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to Unmarshal integer number: %w", err)
	}
	return int64Value, nil
}

// serializeUnicode returns the unicode character given the code points in reader. Expects 4 hex digits.
func convertHexToUnicode(reader *bufio.Reader) (rune, error) {
	var hexChars [4]byte
	n, err := reader.Read(hexChars[:])
	if n != 4 {
		return 0, UNICODE_INSUFFICIENT_BYTES
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read hex chars for unicode: %w", err)
	}
	hexString := string(hexChars[:])
	hexValue, err := strconv.ParseInt(hexString, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to Unmarshal hex string: %s", hexString)
	}
	return rune(hexValue), nil
}

func UnmarshalString(reader *bufio.Reader) (string, error) {
	// Verify that the first char is a double quote
	r, _, err := reader.ReadRune()
	if err != nil {
		return "", fmt.Errorf("failed to read rune: %w", err)
	}
	if r != '"' {
		return "", fmt.Errorf("cannot match string, no opening double quote found: %c", r)
	}
	var b strings.Builder
	backslash := false
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return "", fmt.Errorf("failed to read rune: %w", err)
		}
		// Handle escaped characters
		if backslash {
			switch r {
			case '"':
				b.WriteRune('"')
			case '\\':
				b.WriteRune('\\')
			case '/':
				b.WriteRune('/')
			case 'b':
				b.WriteRune('\b')
			case 'f':
				b.WriteRune('\f')
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case 'u':
				unicodeChar, err := convertHexToUnicode(reader)
				if err != nil {
					return "", fmt.Errorf("failed to Unmarshal unicode character")
				}
				b.WriteRune(unicodeChar)
			default:
				return "", fmt.Errorf("error: unexpected escape character %c", r)
			}
			backslash = false
		} else {
			if r == '\\' {
				backslash = true
			} else if r == '"' {
				break
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String(), nil
}

func Serialize(reader bufio.Reader) (interface{}, error) {

	return nil, nil
}
