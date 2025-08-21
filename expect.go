package expect

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type editedLine struct {
	file       string
	line       int
	addedLines int
}

var editedLines []editedLine = make([]editedLine, 0)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// Validates the output as marshalled JSON.
func JsonEqual(t *testing.T, args ...any) {
	if len(args) == 0 {
		t.Fatalf("Expected at least one value, got none")
		return
	}
	file, line, err := getCurrentFileAndLine()
	if err != nil {
		t.Fatalf("Could not retrieve caller information: %v", err)
		return
	}

	// Normal test!
	if len(args) == 2 {
		switch s := args[1].(type) {
		case string:
			ok, err := jsonEqual(args[0], s)
			if err != nil {
				t.Fatalf("Could not check equality: %v", err)
			}
			if !ok {
				expected, err := jsonString(args[0])
				if err != nil {
					t.Fatalf("Failed to marshal expected value to JSON: %v", err)
				}
				t.Fatalf("%v\ndoes not equal %s", expected, s)
			}
			return
		default:
			t.Fatalf("Expected a string as the second argument, got: %T", s)
		}
	}

	if len(args) == 1 {
		js, err := jsonString(args[0])
		if err != nil {
			t.Fatalf("Failed to marshal value to JSON: %v", err)
			return
		}
		addValueToFile(t, file, line, fmt.Sprintf("`\n%s`", js))
		return
	}

	t.Fatalf("Expected at most two values, got %d", len(args))
}

func Equal(t *testing.T, args ...any) {
	if len(args) == 0 {
		t.Fatalf("Expected at least one value, got none")
		return
	}
	file, line, err := getCurrentFileAndLine()
	if err != nil {
		t.Fatalf("Could not retrieve caller information: %v", err)
		return
	}

	if len(args) == 2 {
		// Normal test! nice this should be easy!
		// But using require here kind of ruins it, as the output of a failing
		// test here will not make sense for the user.
		equal(t, args[0], args[1])
		return
	}

	if len(args) == 1 {
		addValueToFile(t, file, line, valueString(args[0]))
		return
	}

	t.Fatalf("Expected at most two values, got %d", len(args))
}

// Fields expects a struct or a map[string|int]any as the second argument.
//
// Fields will then generate multiple tests for each field in the struct or map.
func Fields(t *testing.T, arg any) {
	file, line, err := getCurrentFileAndLine()
	if err != nil {
		t.Fatalf("Could not retrieve caller information: %v", err)
		return
	}

	// TODO: support nested structs and maps
	fields := make([]FieldValue, 0)
	for _, key := range getStructKeys(arg) {
		fields = append(fields, FieldValue{
			FieldName: fmt.Sprintf(".%s", key),
			Value:     valueString(reflect.ValueOf(arg).FieldByName(key).Interface()),
		})
	}
	for _, key := range getMapKeys(arg) {
		fields = append(fields, FieldValue{
			FieldName: fmt.Sprintf("[\"%s\"]", key),
			Value:     valueString(reflect.ValueOf(arg).MapIndex(reflect.ValueOf(key)).Interface()),
		})
	}

	addLinesToFile(t, file, line, fields)
}

func getStructKeys(input any) []string {
	v := reflect.ValueOf(input)
	t := v.Type()
	var keys []string
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			keys = append(keys, t.Field(i).Name)
		}
	}
	return keys
}

func getMapKeys(input any) []string {
	v := reflect.ValueOf(input)
	var keys []string
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			keys = append(keys, fmt.Sprintf("%v", k.Interface()))
		}
	}
	return keys
}

func equal(t *testing.T, actual any, expected any) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected values do not match:\nExpected: %v\nActual: %v", expected, actual)
	}
}

// Expects expected to be json output
func jsonEqual(actual any, expected string) (bool, error) {
	j, err := jsonString(actual)
	if err != nil {
		return false, fmt.Errorf("failed to marshal actual value to JSON: %w", err)
	}
	return jsonStringEqual(j, expected), nil
}

func jsonStringEqual(a, b string) bool {
	aa := strings.Trim(a, "`")
	bb := strings.Trim(b, "`")
	aa = strings.TrimSpace(aa)
	bb = strings.TrimSpace(bb)
	return reflect.DeepEqual(aa, bb)
}

func addValueToFile(t *testing.T, file string, line int, value string) {
	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	if line-1 < 0 || line-1 >= len(lines) {
		t.Fatalf("Invalid line number: %d", line)
		return
	}

	addedLines := len(strings.Split(value, "\n")) - 1

	// Insert the value as a comment at the current line
	// TODO: We might have a comment here, so we need to handle that!
	// lines[line-1] = strings.TrimSpace(lines[line-1])
	lines[line-1] = strings.TrimSuffix(lines[line-1], ")")
	lines[line-1] += fmt.Sprintf(", %s)", value)

	// Write back to the file
	err = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
		return
	}
	if addedLines > 0 {
		editedLines = append(editedLines, editedLine{
			file:       file,
			line:       line,
			addedLines: addedLines,
		})
	}
}

type FieldValue struct {
	FieldName string // either ["sdf"] or ".Sdf"
	Value     string
}

// Used by Fields. Will write over the current line.
func addLinesToFile(t *testing.T, file string, line int, values []FieldValue) {
	if len(values) == 0 {
		t.Fatalf("Expected at least one field value, got none")
		return
	}
	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	if line-1 < 0 || line-1 >= len(lines) {
		t.Fatalf("Invalid line number: %d", line)
		return
	}

	// Insert the value as a comment at the current line
	// TODO: We might have a comment here, so we need to handle that!
	// lines[line-1] = strings.TrimSpace(lines[line-1])
	firstValue := values[0]
	template := ""
	lines[line-1] = strings.TrimSuffix(lines[line-1], ")")                 // remove the closing parenthesis
	lines[line-1] = strings.Replace(lines[line-1], "Fields(", "Equal(", 1) // change Multi to Equal
	template = strings.TrimRight(lines[line-1], " ")

	lines[line-1] = template + firstValue.FieldName
	lines[line-1] += fmt.Sprintf(", %s)", firstValue.Value)

	for _, field := range values[1:] {
		lines[line-1] += "\n"
		lines[line-1] += template + field.FieldName
		lines[line-1] += fmt.Sprintf(", %s)", field.Value)
	}

	addedLines := len(strings.Split(lines[line-1], "\n"))

	// Write back to the file
	err = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
		return
	}
	if addedLines > 0 {
		editedLines = append(editedLines, editedLine{
			file:       file,
			line:       line,
			addedLines: addedLines,
		})
	}

	// TODO: run gofmt on the file after this!
}

func getCurrentFileAndLine() (string, int, error) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "", 0, fmt.Errorf("could not retrieve caller information")
	}
	return file, updateLines(file, line), nil
}

// Update line number based on edited lines
func updateLines(file string, line int) int {
	if len(editedLines) == 0 {
		return line
	}
	for _, el := range editedLines {
		if el.file == file {
			if el.line <= line {
				line += el.addedLines
			}
		}
	}
	return line
}

func valueString(value any) string {
	switch v := value.(type) {
	case string:
		multiline := len(strings.Split(v, "\n")) > 1
		if multiline {
			return fmt.Sprintf("`%s`", v)
		} else {
			return fmt.Sprintf("\"%s\"", v)
		}
	case int, int64, float64, float32, uint, uint64:
		return fmt.Sprintf("%v", v)
	case nil:
		return "nil"
	case map[string]any:
		// TODO: Should return error to indicate that this is not supported and use JsonEqual instead.
		return "map[string]any{" + fmt.Sprintf("%v", v) + "}"
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("\"%+v\"", v) // TODO: should probably just fail here
	}
}

func jsonString(value any) (string, error) {
	v, err := json.MarshalIndent(value, " ", " ")
	if err != nil {
		return "", err
	}
	return string(v), nil
}
