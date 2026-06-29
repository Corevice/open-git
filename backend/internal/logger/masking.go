package logger

import (
	"net/http"
	"strings"
)

var sensitiveKeys = []string{"authorization", "token", "secret", "password", "x-auth-token"}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, sk := range sensitiveKeys {
		if lower == sk {
			return true
		}
	}
	return false
}

func MaskValue(key, val string) string {
	if isSensitiveKey(key) {
		return "***"
	}
	return val
}

func MaskHeaders(h http.Header) map[string]string {
	if h == nil {
		return nil
	}
	result := make(map[string]string, len(h))
	for k, vals := range h {
		result[k] = MaskValue(k, strings.Join(vals, ", "))
	}
	return result
}

func MaskMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = MaskValue(k, val)
		case map[string]interface{}:
			result[k] = MaskMap(val)
		default:
			result[k] = v
		}
	}
	return result
}
