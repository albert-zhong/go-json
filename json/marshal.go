package json

import (
	"bufio"
	"fmt"
	"reflect"
	"strconv"
)

func MarshalValue(value interface{}, writer *bufio.Writer) error {
	// Handle null value
	if value == nil {
		if err := MarshalNull(writer); err != nil {
			return fmt.Errorf("failed to write null: %w", err)
		}
		return nil
	}
	// Handle non-null values
	valueType := reflect.TypeOf(value)
	switch valueType.Kind() {
	case reflect.String:
		valueString, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to cast value to string")
		}
		return MarshalString(valueString, writer)
	case reflect.Int64:
		fallthrough
	case reflect.Float64:
		return MarshalNumber(value, writer)
	case reflect.Map:
		object, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast object to map[string]interface{}")
		}
		return MarshalObject(object, writer)
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		reflectedValues := reflect.ValueOf(value)
		values := make([]interface{}, reflectedValues.Len())
		for i := 0; i < reflectedValues.Len(); i++ {
			values[i] = reflectedValues.Index(i).Interface()
		}
		return MarshalArray(values, writer)
	case reflect.Bool:
		value, ok := value.(bool)
		if !ok {
			return fmt.Errorf("failed to cast bool")
		}
		return MarshalBoolean(value, writer)
	default:
		return fmt.Errorf("cannot marshal value %v", value)
	}
}

func MarshalString(value string, writer *bufio.Writer) error {
	var err error
	if err = writer.WriteByte('"'); err != nil {
		return fmt.Errorf("failed to write opening \" in string %s: %w", value, err)
	}
	for _, c := range value {
		switch c {
		case '"':
			_, err = writer.WriteString(`\"`)
		case '\\':
			_, err = writer.WriteString(`\\`)
		case '\b':
			_, err = writer.WriteString(`\b`)
		case '\f':
			_, err = writer.WriteString(`\f`)
		case '\n':
			_, err = writer.WriteString(`\n`)
		case '\r':
			_, err = writer.WriteString(`\r`)
		case '\t':
			_, err = writer.WriteString(`\t`)
		default:
			_, err = writer.WriteRune(c)
		}
		if err != nil {
			return fmt.Errorf("failed to write rune %c from string %s: %w", c, value, err)
		}
	}
	if err = writer.WriteByte('"'); err != nil {
		return fmt.Errorf("failed to write closing \" in string %s: %w", value, err)
	}
	return nil
}

func MarshalNumber(value interface{}, writer *bufio.Writer) error {
	var valueString string
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int64:
		valueInt64, ok := value.(int64)
		if !ok {
			return fmt.Errorf("failed to cast number to int64")
		}
		valueString = strconv.FormatInt(valueInt64, 10)
	case reflect.Float64:
		valueFloat64, ok := value.(float64)
		if !ok {
			return fmt.Errorf("failed to cast number to float64")
		}
		valueString = strconv.FormatFloat(valueFloat64, 'f', -1, 64)
	default:
		return fmt.Errorf("number was not int64 or float64")
	}
	if _, err := writer.WriteString(valueString); err != nil {
		return fmt.Errorf("failed to write value %s: %w", valueString, err)
	}
	return nil
}

func MarshalBoolean(value bool, writer *bufio.Writer) error {
	var err error
	if value {
		_, err = writer.WriteString(TRUE_STRING)
	} else {
		_, err = writer.WriteString(FALSE_STRING)
	}
	if err != nil {
		return fmt.Errorf("failed to write boolean: %w", err)
	}
	return nil
}

func MarshalNull(writer *bufio.Writer) error {
	if _, err := writer.WriteString(NULL_STRING); err != nil {
		return fmt.Errorf("failed to write null: %w", err)
	}
	return nil
}

func MarshalArray(values []interface{}, writer *bufio.Writer) error {
	if err := writer.WriteByte('['); err != nil {
		return fmt.Errorf("failed to write [: %w", err)
	}
	for i, value := range values {
		if err := MarshalValue(value, writer); err != nil {
			return fmt.Errorf("failed to write array value at index %d: %w", i, err)
		}
		if i < len(values)-1 {
			if err := writer.WriteByte(','); err != nil {
				return fmt.Errorf("failed to write ,: %w", err)
			}
		}
	}
	if err := writer.WriteByte(']'); err != nil {
		return fmt.Errorf("failed to write ]: %w", err)
	}
	return nil
}

func MarshalObject(object map[string]interface{}, writer *bufio.Writer) error {
	if err := writer.WriteByte('{'); err != nil {
		return fmt.Errorf("failed to write {: %w", err)
	}
	i := 0
	for key, value := range object {
		if err := MarshalString(key, writer); err != nil {
			return fmt.Errorf("failed to write object key %s: %w", key, err)
		}
		if err := writer.WriteByte(':'); err != nil {
			return fmt.Errorf("failed to write ':': %w", err)
		}
		if err := MarshalValue(value, writer); err != nil {
			return fmt.Errorf("failed to write object value: %w", err)
		}
		if i < len(object)-1 {
			if err := writer.WriteByte(','); err != nil {
				return fmt.Errorf("failed to write ',': %w", err)
			}
		}
		i += 1
	}
	if err := writer.WriteByte('}'); err != nil {
		return fmt.Errorf("failed to write }: %w", err)
	}
	return nil
}
