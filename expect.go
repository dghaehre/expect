package expect

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	// "github.com/stretchr/testify/require"
)

// TODO: a function might run multiple times.. we should not modify that line multiple times. Probably just the first time!
// Not sure how relavant this really is though...
var EditedLines = make(map[string]int) // file -> line number

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func Test(t *testing.T, args ...any) {
	if len(args) == 0 {
		t.Fatalf("Expected at least one value, got none")
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
		addValueToFile(t, args[0])
		return
	}

	t.Fatalf("Expected at most two values, got %d", len(args))
}

// TODO: make this output a lot better!
func equal(t *testing.T, actual, expected any) {
	if !reflect.DeepEqual(expected, actual) {
		switch expected.(type) {
		case string:
			if !reflect.DeepEqual(expected, valueString(actual)) {
				t.Errorf("Expected string value does not match:\nExpected: %s\nActual: %s", expected, valueString(actual))
				t.FailNow()
			}
			return
		}

		t.Errorf("Expected values do not match:\nExpected: %v\nActual: %v", expected, actual)
		t.FailNow()
	}
}

func addValueToFile(t *testing.T, value any) {
	// This function is a placeholder for the actual implementation
	// that would add the value to a file.
	// For now, we just log the value.
	file, line := getCurrentFileAndLine() // EXPECTED: event.outdated.skipped

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
	lines[line-1] = strings.TrimSuffix(lines[line-1], ")")
	lines[line-1] += fmt.Sprintf(", %s)", valueString(value))

	// Write back to the file
	err = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
		return
	}
}

func getCurrentFileAndLine() (string, int) {
	_, file, line, ok := runtime.Caller(3) // yes!
	if !ok {
		return "unknown", 0
	}
	return file, line
}

func valueString(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case nil:
		return "nil"
	case map[string]any:
		return "map[string]any{" + fmt.Sprintf("%v", v) + "}"
	case bool:
		return fmt.Sprintf("%t", v)
	case fmt.Stringer:
		return fmt.Sprintf("\"%s\"", v.String())
	default:
		// TODO: Handle more types as needed
		// spit out json if we can
		// spit out json if we can
		if b, err := json.MarshalIndent(v, " ", " "); err == nil {
			return fmt.Sprintf("`\n	%s\n	`", b)
		}
		return fmt.Sprintf("\"%+v\"", v) // TODO: should probably just fail here
	}
}
