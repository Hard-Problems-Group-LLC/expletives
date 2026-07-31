package automation

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var errJSONRecordTooLong = errors.New("JSON record exceeds configured limit")

const maxJSONEncodeDepth = 64

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// boundedJSONBuffer never allocates capacity beyond limit. In particular, an
// oversized string or collection is rejected as its encoding reaches the
// record limit rather than after an unbounded complete record is materialized.
type boundedJSONBuffer struct {
	data  []byte
	limit int
}

func (b *boundedJSONBuffer) writeString(value string) error {
	if len(value) > b.limit-len(b.data) {
		return errJSONRecordTooLong
	}
	b.grow(len(b.data) + len(value))
	b.data = append(b.data, value...)
	return nil
}

func (b *boundedJSONBuffer) writeByte(value byte) error {
	if len(b.data) == b.limit {
		return errJSONRecordTooLong
	}
	b.grow(len(b.data) + 1)
	b.data = append(b.data, value)
	return nil
}

func (b *boundedJSONBuffer) grow(needed int) {
	if needed <= cap(b.data) {
		return
	}
	capacity := cap(b.data) * 2
	if capacity < 512 {
		capacity = 512
	}
	if capacity < needed {
		capacity = needed
	}
	if capacity > b.limit {
		capacity = b.limit
	}
	grown := make([]byte, len(b.data), capacity)
	copy(grown, b.data)
	b.data = grown
}

// marshalBoundedJSONLine supports the package's protocol structs and their
// string-keyed map, collection, pointer, scalar, and interface components. It
// intentionally rejects byte slices, custom marshalers, and other shapes not
// used by the protocol instead of silently broadening this trust boundary.
func marshalBoundedJSONLine(value any, maxBytes int) ([]byte, error) {
	if maxBytes < 0 {
		return nil, errJSONRecordTooLong
	}
	writer := &boundedJSONBuffer{limit: maxBytes}
	if err := writeJSONValue(writer, reflect.ValueOf(value), 0); err != nil {
		return nil, err
	}
	// The framing LF is not part of the advertised JSON-byte limit.
	writer.limit = maxBytes + 1
	if err := writer.writeByte('\n'); err != nil {
		return nil, err
	}
	return writer.data, nil
}

func writeJSONValue(
	writer *boundedJSONBuffer,
	value reflect.Value,
	depth int,
) error {
	if depth > maxJSONEncodeDepth {
		return errors.New("maximum JSON encoding depth exceeded")
	}
	if !value.IsValid() {
		return writer.writeString("null")
	}
	if value.Type().Implements(jsonMarshalerType) ||
		(value.CanAddr() && value.Addr().Type().Implements(jsonMarshalerType)) {
		return fmt.Errorf(
			"custom JSON marshaler %s is not supported",
			value.Type(),
		)
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return writer.writeString("null")
		}
		return writeJSONValue(writer, value.Elem(), depth+1)
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return writer.writeString("null")
		}
		return writeJSONValue(writer, value.Elem(), depth+1)
	}

	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return writer.writeString("true")
		}
		return writer.writeString("false")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var encoded [32]byte
		return writer.writeString(string(strconv.AppendInt(
			encoded[:0],
			value.Int(),
			10,
		)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		var encoded [32]byte
		return writer.writeString(string(strconv.AppendUint(
			encoded[:0],
			value.Uint(),
			10,
		)))
	case reflect.Float32, reflect.Float64:
		number := value.Float()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			return fmt.Errorf("unsupported floating-point value %v", number)
		}
		var encoded [32]byte
		bits := value.Type().Bits()
		return writer.writeString(string(strconv.AppendFloat(
			encoded[:0],
			number,
			'g',
			-1,
			bits,
		)))
	case reflect.String:
		if value.Type() == reflect.TypeOf(json.Number("")) {
			return errors.New("json.Number is not used by the bounded protocol encoder")
		}
		return writeJSONString(writer, value.String())
	case reflect.Struct:
		return writeJSONObject(writer, value, depth+1)
	case reflect.Slice:
		if value.IsNil() {
			return writer.writeString("null")
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return errors.New("byte slices are not supported by bounded protocol encoding")
		}
		return writeJSONArray(writer, value, depth+1)
	case reflect.Array:
		return writeJSONArray(writer, value, depth+1)
	case reflect.Map:
		return writeJSONMap(writer, value, depth+1)
	default:
		return fmt.Errorf("unsupported JSON type %s", value.Type())
	}
}

func writeJSONObject(
	writer *boundedJSONBuffer,
	value reflect.Value,
	depth int,
) error {
	if err := writer.writeByte('{'); err != nil {
		return err
	}
	first := true
	if err := writeJSONFields(writer, value, depth, &first); err != nil {
		return err
	}
	return writer.writeByte('}')
}

func writeJSONFields(
	writer *boundedJSONBuffer,
	value reflect.Value,
	depth int,
	first *bool,
) error {
	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		fieldType := valueType.Field(index)
		if fieldType.PkgPath != "" {
			continue
		}
		name, omitEmpty, quoted := parseJSONTag(fieldType.Tag.Get("json"))
		if name == "-" {
			continue
		}
		field := value.Field(index)
		if fieldType.Anonymous && name == "" {
			embedded := field
			if embedded.Kind() == reflect.Pointer {
				if embedded.IsNil() {
					continue
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if err := writeJSONFields(
					writer,
					embedded,
					depth+1,
					first,
				); err != nil {
					return err
				}
				continue
			}
		}
		if omitEmpty && emptyJSONValue(field) {
			continue
		}
		if quoted {
			return errors.New("the JSON string tag option is not supported")
		}
		if name == "" {
			name = fieldType.Name
		}
		if !*first {
			if err := writer.writeByte(','); err != nil {
				return err
			}
		}
		*first = false
		if err := writeJSONString(writer, name); err != nil {
			return err
		}
		if err := writer.writeByte(':'); err != nil {
			return err
		}
		if err := writeJSONValue(writer, field, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONArray(
	writer *boundedJSONBuffer,
	value reflect.Value,
	depth int,
) error {
	if err := writer.writeByte('['); err != nil {
		return err
	}
	for index := 0; index < value.Len(); index++ {
		if index != 0 {
			if err := writer.writeByte(','); err != nil {
				return err
			}
		}
		if err := writeJSONValue(writer, value.Index(index), depth+1); err != nil {
			return err
		}
	}
	return writer.writeByte(']')
}

func writeJSONMap(
	writer *boundedJSONBuffer,
	value reflect.Value,
	depth int,
) error {
	if value.IsNil() {
		return writer.writeString("null")
	}
	if value.Type().Key().Kind() != reflect.String {
		return errors.New("only string-keyed JSON maps are supported")
	}
	if err := writer.writeByte('{'); err != nil {
		return err
	}
	// Even an empty string key and a one-byte scalar value need five bytes per
	// entry once commas and the closing brace are included. Reject impossible
	// map sizes before MapKeys allocates the sorting inventory.
	if value.Len() > (writer.limit-len(writer.data))/5 {
		return errJSONRecordTooLong
	}
	keys := value.MapKeys()
	sort.Slice(keys, func(left, right int) bool {
		return keys[left].String() < keys[right].String()
	})
	for index, key := range keys {
		if index != 0 {
			if err := writer.writeByte(','); err != nil {
				return err
			}
		}
		if err := writeJSONString(writer, key.String()); err != nil {
			return err
		}
		if err := writer.writeByte(':'); err != nil {
			return err
		}
		if err := writeJSONValue(
			writer,
			value.MapIndex(key),
			depth+1,
		); err != nil {
			return err
		}
	}
	return writer.writeByte('}')
}

func writeJSONString(writer *boundedJSONBuffer, value string) error {
	if err := writer.writeByte('"'); err != nil {
		return err
	}
	start := 0
	for index := 0; index < len(value); {
		current := value[index]
		if current < utf8.RuneSelf {
			if current >= 0x20 &&
				current != '\\' &&
				current != '"' &&
				current != '<' &&
				current != '>' &&
				current != '&' {
				index++
				continue
			}
			if err := writer.writeString(value[start:index]); err != nil {
				return err
			}
			var escaped string
			switch current {
			case '\\', '"':
				escaped = `\` + string(current)
			case '\b':
				escaped = `\b`
			case '\f':
				escaped = `\f`
			case '\n':
				escaped = `\n`
			case '\r':
				escaped = `\r`
			case '\t':
				escaped = `\t`
			default:
				const hexadecimal = "0123456789abcdef"
				escaped = `\u00` +
					string(hexadecimal[current>>4]) +
					string(hexadecimal[current&0x0f])
			}
			if err := writer.writeString(escaped); err != nil {
				return err
			}
			index++
			start = index
			continue
		}

		currentRune, size := utf8.DecodeRuneInString(value[index:])
		if currentRune == utf8.RuneError && size == 1 {
			if err := writer.writeString(value[start:index]); err != nil {
				return err
			}
			if err := writer.writeString(`\ufffd`); err != nil {
				return err
			}
			index++
			start = index
			continue
		}
		if currentRune == '\u2028' || currentRune == '\u2029' {
			if err := writer.writeString(value[start:index]); err != nil {
				return err
			}
			if currentRune == '\u2028' {
				if err := writer.writeString(`\u2028`); err != nil {
					return err
				}
			} else if err := writer.writeString(`\u2029`); err != nil {
				return err
			}
			index += size
			start = index
			continue
		}
		index += size
	}
	if err := writer.writeString(value[start:]); err != nil {
		return err
	}
	return writer.writeByte('"')
}

func parseJSONTag(tag string) (name string, omitEmpty bool, quoted bool) {
	name, optionText, found := strings.Cut(tag, ",")
	if !found {
		return tag, false, false
	}
	for optionText != "" {
		var option string
		option, optionText, _ = strings.Cut(optionText, ",")
		switch option {
		case "omitempty":
			omitEmpty = true
		case "string":
			quoted = true
		}
	}
	return name, omitEmpty, quoted
}

func emptyJSONValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Interface,
		reflect.Pointer:
		return value.IsZero()
	default:
		return false
	}
}
