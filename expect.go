package expect

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// TODO: some function to update tests (jsonEqual) with updated values.

// TODO: handle nested structs and maps in Fields

type editedLine struct {
	file       string
	line       int
	addedLines int
}

// If the test fail, we can override the expected value by setting the actual value
var override = os.Getenv("EXPECT_OVERRIDE") == "true"

var editedLines []editedLine = make([]editedLine, 0)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func CensorDigits(s any) string {
	switch s := s.(type) {
	case string:
		return strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return 'X'
			}
			return r
		}, s)
	case int, int64, float64, float32, uint, uint64:
		return CensorDigits(fmt.Sprintf("%v", s))
	default:
		return fmt.Sprintf("%v", s) // Just return the string representation
	}
}

func CensorChars(s any) string {
	switch s := s.(type) {
	case string:
		return strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return 'X'
			}
			return r
		}, s)
	default:
		return fmt.Sprintf("%v", s) // Just return the string representation
	}
}

func CensorAlphanumeric(s any) string {
	switch s := s.(type) {
	case string:
		return strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return 'X'
			}
			return r
		}, s)
	case int, int64, float64, float32, uint, uint64:
		return CensorAlphanumeric(fmt.Sprintf("%v", s))
	default:
		return fmt.Sprintf("%v", s) // Just return the string representation
	}
}

// Validates the output as marshalled JSON.
func JsonEqual(t *testing.T, args ...any) {
	if len(args) == 0 {
		t.Fatalf("Expected at least one value, got none")
		return
	}
	file, line, _, err := getCurrentFileAndLine()
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
				if override {
					// TODO: allow for overriding json value (multiline)
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
	file, line, packageName, err := getCurrentFileAndLine()
	if err != nil {
		t.Fatalf("Could not retrieve caller information: %v", err)
		return
	}

	if len(args) == 2 {
		e := equal(args[0], args[1])
		if !e && override {
			// Overide the value
			changeValueToFile(t, file, line, valueString(packageName, args[0]))
		} else if !e {
			t.Fatalf("Expected values do not match:\nExpected: %+v\nActual:   %+v", valueString("", args[1]), valueString("", args[0]))
		}
		return
	}

	if len(args) == 1 {
		addValueToFile(t, file, line, valueString(packageName, args[0]))
		return
	}

	t.Fatalf("Expected at most two values, got %d", len(args))
}

// Fields expects a struct or a map[string|int]any as the second argument.
//
// Fields will then generate multiple tests for each field in the struct or map.
func Fields(t *testing.T, arg any) {
	file, line, packageName, err := getCurrentFileAndLine()
	if err != nil {
		t.Fatalf("Could not retrieve caller information: %v", err)
		return
	}

	// TODO: support nested structs and maps
	fields := make([]FieldValue, 0)
	for _, key := range getStructKeys(arg) {
		fields = append(fields, FieldValue{
			FieldName: fmt.Sprintf(".%s", key),
			Value:     valueString(packageName, reflect.ValueOf(arg).FieldByName(key).Interface()),
		})
	}
	for _, key := range getMapKeys(arg) {
		fields = append(fields, FieldValue{
			FieldName: fmt.Sprintf("[\"%s\"]", key),
			Value:     valueString(packageName, reflect.ValueOf(arg).MapIndex(reflect.ValueOf(key)).Interface()),
		})
	}

	// If we have no fields, we can still add the value as a single field!
	if len(fields) == 0 {
		fields = append(fields, FieldValue{
			FieldName: "",
			Value:     valueString(packageName, arg),
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
		// TODO: somehow handle pointers..
		// case reflect.Pointer:
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

func equal(actual any, expected any) bool {
	return reflect.DeepEqual(expected, actual) || reflect.DeepEqual(valueString("", expected), valueString("", actual))
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

func changeValueToFile(t *testing.T, file string, line int, value string) {
	if strings.Contains(value, "\n") {
		t.Fatalf("Expected a single line value, got a multiline value. Currently not supported.")
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
	lines[line-1] = strings.TrimRight(lines[line-1], " ") // remove trailing spaces
	// last character is a closing parenthesis
	if !strings.HasSuffix(lines[line-1], ")") {
		t.Fatalf("Expected a single line value to edit, got: %s. Multiple value edit not supported.", lines[line-1])
		return
	}

	lines[line-1] = strings.TrimSuffix(lines[line-1], ")") // remove the ')'
	// find the index of the second comma:
	commaIndex := strings.LastIndex(lines[line-1], ",")
	if commaIndex == -1 {
		t.Fatalf("Expected a single line value to edit, got: %s. Multiple value edit not supported.", lines[line-1])
		return
	}
	lines[line-1] = lines[line-1][:commaIndex+1] // keep everything
	lines[line-1] += fmt.Sprintf(" %s)", value)  // add the new value

	// Write back to the file
	err = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
		return
	}
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

func getCurrentFileAndLine() (string, int, string, error) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "", 0, "", fmt.Errorf("could not retrieve caller information")
	}
	packageName := strings.Split(path.Base(runtime.FuncForPC(pc).Name()), ".")[0]
	return file, updateLines(file, line), packageName, nil
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

// Make sure " is handle as \"
func sanitizeString(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "`", "\\`") // Handle backticks as well
	return s
}

// TODO: better output if the aliastype("one") != "one"
func valueString(packageName string, value any) string {
	switch v := value.(type) {
	case string:
		multiline := len(strings.Split(v, "\n")) > 1
		if multiline {
			return fmt.Sprintf("`%s`", sanitizeString(v))
		} else {
			return fmt.Sprintf("\"%s\"", sanitizeString(v))
		}
	case int, int64, float64, float32, uint, uint64:
		return fmt.Sprintf("%v", v)
	case nil:
		return "nil"
	case map[string]any:
		// TODO: Should probably return error to indicate that this is not supported and use JsonEqual instead.
		return "map[string]any{" + fmt.Sprintf("%v", v) + "}"
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		// Might be a type alias. Support int or string aliases.
		t := reflect.TypeOf(v)
		packagePath := t.PkgPath()
		pkgName := path.Base(packagePath)
		pkgPrefix := ""
		if packageName != "" && packageName != pkgName {
			pkgPrefix = pkgName + "."
		}
		switch t.Kind() {
		case reflect.String:
			return fmt.Sprintf("%s%s(\"%s\")", pkgPrefix, t.Name(), sanitizeString(reflect.ValueOf(v).String()))
		case reflect.Int, reflect.Int64, reflect.Float64, reflect.Float32, reflect.Uint, reflect.Uint64:
			number, ok := numberFromAny(v)
			if !ok {
				panic(fmt.Sprintf("could not convert number: %T, %s", v, reflect.TypeOf(v).Name()))
			}
			return fmt.Sprintf("%s(%s)", t.Name(), number)
		case reflect.Pointer:
			// If it's a pointer, we can dereference it to get the value.
			if reflect.ValueOf(v).IsNil() {
				return "nil"
			}
			// Dereference the pointer to get the value.
			dereferencedValue := reflect.ValueOf(v).Elem().Interface()
			return valueString(packageName, dereferencedValue)
		default:
			// not alias type
		}

		switch value.(type) {
		case fmt.Stringer:
			// If the value implements fmt.Stringer, we can use its String method.
			return fmt.Sprintf("\"%s\"", sanitizeString(v.(fmt.Stringer).String()))
		case fmt.GoStringer:
			// If the value implements fmt.Stringer, we can use its String method.
			return fmt.Sprintf("\"%s\"", sanitizeString(v.(fmt.GoStringer).GoString()))
		}
		return fmt.Sprintf("\"%+v\"", v)
	}
}

func numberFromAny(v any) (string, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", rv.Float()), true
	}
	return "0", false
}

func jsonString(value any) (string, error) {
	v, err := json.MarshalIndent(value, " ", " ")
	if err != nil {
		return "", err
	}
	return string(v), nil
}
