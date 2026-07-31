package automation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"

	expletives "github.com/Hard-Problems-Group-LLC/expletives"
	"github.com/rivo/uniseg"
)

var (
	errLineTooLong = errors.New("JSON line exceeds configured limit")
	errTruncated   = errors.New("JSON line is not newline terminated")
)

type decodedRequest struct {
	operation string
	requestID string
	value     any
}

type decodeError struct {
	requestID string
	code      string
	message   string
	retryable bool
}

func (e *decodeError) Error() string {
	return e.message
}

func readLine(conn net.Conn, reader *bufio.Reader, maxBytes int) ([]byte, error) {
	if maxBytes < 0 {
		return nil, errLineTooLong
	}

	var line []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(line) == 0 && err == nil {
			line = fragment
			break
		}

		maxBuffered := maxBytes
		if err == nil {
			// A complete line has one additional delimiter byte.
			maxBuffered++
		}
		if len(line) > maxBuffered-len(fragment) {
			return nil, errLineTooLong
		}
		line = append(line, fragment...)

		switch {
		case err == nil:
			break
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(line) == 0 {
				return nil, io.EOF
			}
			return nil, errTruncated
		default:
			return nil, err
		}
		break
	}

	if len(line)-1 > maxBytes {
		return nil, errLineTooLong
	}
	line = line[:len(line)-1]
	if len(line) == 0 || len(bytes.TrimSpace(line)) == 0 {
		return nil, errors.New("blank JSON line")
	}
	return line, nil
}

func decodeRequest(line []byte, limits Limits) (decodedRequest, error) {
	if len(line) > limits.RequestLineBytes {
		return decodedRequest{}, &decodeError{
			code:    "line_too_long",
			message: "request line exceeds configured limit",
		}
	}
	if !utf8.Valid(line) {
		return decodedRequest{}, &decodeError{
			code:    "invalid_utf8",
			message: "request is not valid UTF-8",
		}
	}
	if err := validateJSONStructure(line, limits.JSONDepth); err != nil {
		return decodedRequest{}, &decodeError{
			code:    "malformed_json",
			message: boundedMessage(err.Error()),
		}
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return decodedRequest{}, &decodeError{
			code:    "malformed_json",
			message: boundedMessage(err.Error()),
		}
	}

	var header Header
	if err := json.Unmarshal(line, &header); err != nil {
		return decodedRequest{}, &decodeError{
			code:    "invalid_envelope",
			message: boundedMessage(err.Error()),
		}
	}
	requestID := rawString(fields["request_id"])
	if !validIdentifier(requestID, limits.RequestIDBytes) {
		requestID = ""
	}
	if header.Protocol != Protocol {
		return decodedRequest{}, &decodeError{
			requestID: requestID,
			code:      "unsupported_protocol",
			message:   "protocol must be expletives.automation",
		}
	}
	if header.Version != Version {
		return decodedRequest{}, &decodeError{
			requestID: requestID,
			code:      "unsupported_version",
			message:   "protocol version must be 1",
		}
	}

	switch header.Type {
	case TypeObserve:
		var request observeRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		if isJSONNull(fields["frame_sequence"]) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("frame_sequence cannot be null"))
		}
		if request.FrameSequence != nil && *request.FrameSequence == 0 {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("frame_sequence must be positive"))
		}
		return decodedRequest{operation: TypeObserve, requestID: request.RequestID, value: request}, nil

	case TypeWaitSnapshot:
		var request waitSnapshotRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		if _, exists := fields["after_sequence"]; !exists {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("after_sequence is required"))
		}
		if isJSONNull(fields["after_sequence"]) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("after_sequence cannot be null"))
		}
		if isJSONNull(fields["timeout_ms"]) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("timeout_ms cannot be null"))
		}
		if request.TimeoutMillis < 0 || request.TimeoutMillis > limits.ReadTimeoutMillis {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("timeout_ms exceeds the advertised wait bound"))
		}
		return decodedRequest{operation: TypeWaitSnapshot, requestID: request.RequestID, value: request}, nil

	case TypeInjectInput:
		var request injectInputRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		if err := validateKeyEvent(request.Event, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		return decodedRequest{operation: TypeInjectInput, requestID: request.RequestID, value: request}, nil

	case TypeInvokeCommand:
		var request invokeCommandRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		if !validIdentifier(request.Command, limits.IdentifierBytes) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("command is not a valid bounded identifier"))
		}
		if isJSONNull(fields["target_key"]) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("target_key cannot be null"))
		}
		if request.TargetKey != "" && !validIdentifier(request.TargetKey, limits.IdentifierBytes) {
			return decodedRequest{}, invalidRequest(request.RequestID, errors.New("target_key is not a valid bounded identifier"))
		}
		return decodedRequest{operation: TypeInvokeCommand, requestID: request.RequestID, value: request}, nil

	case TypeQueryResult:
		var request queryResultRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		if err := validateRequestID(request.TargetRequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, fmt.Errorf("target_request_id: %w", err))
		}
		return decodedRequest{operation: TypeQueryResult, requestID: request.RequestID, value: request}, nil

	case TypeResetInput, TypeShutdown:
		var request simpleRequest
		if err := decodeStrict(line, &request); err != nil {
			return decodedRequest{}, invalidRequest(requestID, err)
		}
		if err := validateRequestID(request.RequestID, limits); err != nil {
			return decodedRequest{}, invalidRequest(request.RequestID, err)
		}
		return decodedRequest{operation: header.Type, requestID: request.RequestID, value: request}, nil

	default:
		return decodedRequest{}, &decodeError{
			requestID: requestID,
			code:      "unknown_type",
			message:   "unknown request type",
		}
	}
}

func decodeStrict(line []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values in one record")
	}
	return err
}

func validateJSONStructure(data []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder, 0, maxDepth); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder, depth, maxDepth int) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}

	if delim != '{' && delim != '[' {
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	if depth+1 > maxDepth {
		return fmt.Errorf("JSON nesting exceeds depth %d", maxDepth)
	}

	switch delim {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			keys[key] = struct{}{}
			if err := consumeJSONValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	closeToken, err := decoder.Token()
	if err != nil {
		return err
	}
	closeDelim, ok := closeToken.(json.Delim)
	if !ok || (delim == '{' && closeDelim != '}') || (delim == '[' && closeDelim != ']') {
		return errors.New("mismatched JSON delimiter")
	}
	return nil
}

func validateRequestID(requestID string, limits Limits) error {
	if !validIdentifier(requestID, limits.RequestIDBytes) {
		return errors.New("request_id is not a valid bounded identifier")
	}
	return nil
}

func validateKeyEvent(event KeyEvent, limits Limits) error {
	switch event.Kind {
	case KeyDown, KeyUp, KeyPress:
	default:
		return errors.New("event kind must be key_down, key_up, or key_press")
	}
	if len(event.Key) > limits.IdentifierBytes || !validLogicalKey(event.Key) {
		return errors.New("event key is not a supported logical key")
	}
	return nil
}

func validLogicalKey(key string) bool {
	switch key {
	case "control", "alt", "shift", "meta",
		"space", "enter", "escape", "tab", "backspace",
		"up", "down", "left", "right",
		"home", "end", "page_up", "page_down", "insert", "delete",
		"f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12":
		return true
	default:
	}
	if key == "" || len(key) > expletives.MaxCellBytes ||
		!utf8.ValidString(key) {
		return false
	}
	for _, current := range key {
		if unicode.IsControl(current) {
			return false
		}
	}
	graphemes := uniseg.NewGraphemes(key)
	if !graphemes.Next() {
		return false
	}
	return !graphemes.Next()
}

func invalidRequest(requestID string, err error) *decodeError {
	if !validIdentifier(requestID, DefaultLimits().RequestIDBytes) {
		requestID = ""
	}
	return &decodeError{
		requestID: requestID,
		code:      "invalid_request",
		message:   boundedMessage(err.Error()),
	}
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func isJSONNull(raw json.RawMessage) bool {
	return len(raw) != 0 && bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func boundedMessage(message string) string {
	message = strings.ToValidUTF8(message, "\uFFFD")
	message = strings.ReplaceAll(message, "\x00", "\uFFFD")
	message = strings.TrimSpace(message)
	if len(message) <= maxErrorMessageBytes {
		return message
	}
	message = message[:maxErrorMessageBytes]
	for !utf8.ValidString(message) {
		message = message[:len(message)-1]
	}
	return message
}

func encodeLine(value any, maxBytes int) ([]byte, error) {
	line, err := marshalBoundedJSONLine(value, maxBytes)
	if errors.Is(err, errJSONRecordTooLong) {
		return nil, fmt.Errorf("JSON response exceeds %d-byte limit", maxBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("encode JSON record: %w", err)
	}
	return line, nil
}
