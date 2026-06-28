package compat

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// GoldenPath returns the path to a golden fixture file for method and path.
func GoldenPath(method, path string) string {
	escaped := strings.TrimPrefix(path, "/")
	escaped = strings.ReplaceAll(escaped, "/", "_")
	escaped = strings.ReplaceAll(escaped, "{", "")
	escaped = strings.ReplaceAll(escaped, "}", "")
	return fmt.Sprintf("testdata/golden/%s_%s.json", strings.ToUpper(method), escaped)
}

// LoadGolden reads the golden fixture for method and path.
func LoadGolden(method, path string) ([]byte, error) {
	return os.ReadFile(GoldenPath(method, path))
}

// DiffGolden compares expected and actual JSON payloads and returns a human-readable diff.
func DiffGolden(expected, actual []byte) string {
	var expectedMap map[string]any
	var actualMap map[string]any

	if err := json.Unmarshal(expected, &expectedMap); err != nil {
		return fmt.Sprintf("expected JSON invalid: %v", err)
	}
	if err := json.Unmarshal(actual, &actualMap); err != nil {
		return fmt.Sprintf("actual JSON invalid: %v", err)
	}

	var diffs []string
	compareGoldenMaps("", expectedMap, actualMap, &diffs)
	return strings.Join(diffs, "\n")
}

func compareGoldenMaps(prefix string, expected, actual map[string]any, diffs *[]string) {
	for key, expectedVal := range expected {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		actualVal, ok := actual[key]
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("missing field: %s (expected %s)", path, typeName(expectedVal)))
			continue
		}

		if !sameJSONType(expectedVal, actualVal) {
			*diffs = append(*diffs, fmt.Sprintf("type changed: %s (expected %s, got %s)", path, typeName(expectedVal), typeName(actualVal)))
			continue
		}

		expectedObj, expectedIsObj := expectedVal.(map[string]any)
		actualObj, actualIsObj := actualVal.(map[string]any)
		if expectedIsObj && actualIsObj {
			compareGoldenMaps(path, expectedObj, actualObj, diffs)
		}
	}

	for key := range actual {
		if _, ok := expected[key]; !ok {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			*diffs = append(*diffs, fmt.Sprintf("added field: %s (actual %s)", path, typeName(actual[key])))
		}
	}
}

func sameJSONType(a, b any) bool {
	return typeName(a) == typeName(b)
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}
