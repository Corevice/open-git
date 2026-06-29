package compat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	checkStatusCode            = "status_code"
	checkRateLimitHeaders      = "rate_limit_headers"
	checkLinkPagination        = "link_pagination"
	checkPerPageCap            = "per_page_cap"
	checkErrorFormat           = "error_format"
	checkValidationErrorFormat = "validation_error_format"
	checkSnakeCaseFields       = "snake_case_fields"
	checkDatetimeFields        = "datetime_fields"
	checkIntegerID             = "integer_id"
	checkRequiredFields        = "required_fields"
	checkHTTPSURLField         = "https_url_field"
	checkDocumentationURL      = "documentation_url"
)

// CheckStatusCode verifies the response status matches expected.
func CheckStatusCode(expected int) CheckFunc {
	return func(resp *http.Response, _ []byte) CheckResult {
		passed := resp.StatusCode == expected
		diff := ""
		if !passed {
			diff = fmt.Sprintf("expected status %d, got %d", expected, resp.StatusCode)
		}
		return CheckResult{Name: checkStatusCode, Passed: passed, Diff: diff}
	}
}

// CheckRateLimitHeaders verifies all GitHub-style rate limit headers are present.
func CheckRateLimitHeaders() CheckFunc {
	required := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
		"X-RateLimit-Used",
	}
	return func(resp *http.Response, _ []byte) CheckResult {
		var missing []string
		for _, h := range required {
			if resp.Header.Get(h) == "" {
				missing = append(missing, h)
			}
		}
		if len(missing) == 0 {
			return CheckResult{Name: checkRateLimitHeaders, Passed: true}
		}
		return CheckResult{
			Name:   checkRateLimitHeaders,
			Passed: false,
			Diff:   fmt.Sprintf("missing headers: %s", strings.Join(missing, ", ")),
		}
	}
}

// CheckLinkPagination verifies JSON array responses include a Link header.
func CheckLinkPagination() CheckFunc {
	return func(resp *http.Response, body []byte) CheckResult {
		var arr []json.RawMessage
		if err := json.Unmarshal(body, &arr); err != nil {
			return CheckResult{Name: checkLinkPagination, Passed: true}
		}
		if resp.Header.Get("Link") == "" {
			return CheckResult{
				Name:   checkLinkPagination,
				Passed: false,
				Diff:   "JSON array response missing Link header",
			}
		}
		return CheckResult{Name: checkLinkPagination, Passed: true}
	}
}

// CheckPerPageCap verifies responses respect the 100-item per_page cap.
func CheckPerPageCap() CheckFunc {
	return func(resp *http.Response, body []byte) CheckResult {
		if resp.Request == nil || resp.Request.URL == nil {
			return CheckResult{Name: checkPerPageCap, Passed: true}
		}
		perPageStr := resp.Request.URL.Query().Get("per_page")
		if perPageStr == "" {
			return CheckResult{Name: checkPerPageCap, Passed: true}
		}
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage <= 100 {
			return CheckResult{Name: checkPerPageCap, Passed: true}
		}

		var arr []json.RawMessage
		if err := json.Unmarshal(body, &arr); err != nil {
			return CheckResult{Name: checkPerPageCap, Passed: true}
		}
		if len(arr) > 100 {
			return CheckResult{
				Name:   checkPerPageCap,
				Passed: false,
				Diff:   fmt.Sprintf("expected at most 100 items, got %d", len(arr)),
			}
		}
		return CheckResult{Name: checkPerPageCap, Passed: true}
	}
}

// CheckErrorFormat verifies non-2xx responses include message and documentation_url.
func CheckErrorFormat() CheckFunc {
	return func(resp *http.Response, body []byte) CheckResult {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return CheckResult{Name: checkErrorFormat, Passed: true}
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkErrorFormat,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		var problems []string
		if msg, ok := payload["message"].(string); !ok || msg == "" {
			problems = append(problems, "missing or non-string field: message")
		}
		if doc, ok := payload["documentation_url"].(string); !ok || doc == "" {
			problems = append(problems, "missing or non-string field: documentation_url")
		}
		if len(problems) == 0 {
			return CheckResult{Name: checkErrorFormat, Passed: true}
		}
		return CheckResult{
			Name:   checkErrorFormat,
			Passed: false,
			Diff:   strings.Join(problems, "; "),
		}
	}
}

// CheckValidationErrorFormat verifies 422 responses include a structured errors array.
func CheckValidationErrorFormat() CheckFunc {
	return func(resp *http.Response, body []byte) CheckResult {
		if resp.StatusCode != http.StatusUnprocessableEntity {
			return CheckResult{Name: checkValidationErrorFormat, Passed: true}
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkValidationErrorFormat,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		errorsRaw, ok := payload["errors"]
		if !ok {
			return CheckResult{
				Name:   checkValidationErrorFormat,
				Passed: false,
				Diff:   "missing errors array",
			}
		}

		errorsArr, ok := errorsRaw.([]any)
		if !ok {
			return CheckResult{
				Name:   checkValidationErrorFormat,
				Passed: false,
				Diff:   "errors is not an array",
			}
		}

		for i, item := range errorsArr {
			obj, ok := item.(map[string]any)
			if !ok {
				return CheckResult{
					Name:   checkValidationErrorFormat,
					Passed: false,
					Diff:   fmt.Sprintf("errors[%d] is not an object", i),
				}
			}
			for _, field := range []string{"resource", "field", "code"} {
				if val, ok := obj[field].(string); !ok || val == "" {
					return CheckResult{
						Name:   checkValidationErrorFormat,
						Passed: false,
						Diff:   fmt.Sprintf("errors[%d] missing or non-string field: %s", i, field),
					}
				}
			}
		}

		return CheckResult{Name: checkValidationErrorFormat, Passed: true}
	}
}

// CheckSnakeCaseFields verifies all JSON object keys use snake_case.
func CheckSnakeCaseFields() CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var data any
		if err := json.Unmarshal(body, &data); err != nil {
			return CheckResult{Name: checkSnakeCaseFields, Passed: true}
		}

		var invalid []string
		collectNonSnakeCaseKeys(data, "", &invalid)
		if len(invalid) == 0 {
			return CheckResult{Name: checkSnakeCaseFields, Passed: true}
		}
		return CheckResult{
			Name:   checkSnakeCaseFields,
			Passed: false,
			Diff:   fmt.Sprintf("non-snake_case keys: %s", strings.Join(invalid, ", ")),
		}
	}
}

// CheckDatetimeFields verifies *_at and *_on values are RFC 3339 parseable.
func CheckDatetimeFields() CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var data any
		if err := json.Unmarshal(body, &data); err != nil {
			return CheckResult{Name: checkDatetimeFields, Passed: true}
		}

		var invalid []string
		collectInvalidDatetimeFields(data, "", &invalid)
		if len(invalid) == 0 {
			return CheckResult{Name: checkDatetimeFields, Passed: true}
		}
		return CheckResult{
			Name:   checkDatetimeFields,
			Passed: false,
			Diff:   fmt.Sprintf("invalid datetime values: %s", strings.Join(invalid, ", ")),
		}
	}
}

func collectNonSnakeCaseKeys(v any, prefix string, invalid *[]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			for _, r := range k {
				if unicode.IsUpper(r) {
					*invalid = append(*invalid, path)
					break
				}
			}
			collectNonSnakeCaseKeys(child, path, invalid)
		}
	case []any:
		for i, child := range val {
			collectNonSnakeCaseKeys(child, fmt.Sprintf("%s[%d]", prefix, i), invalid)
		}
	}
}

func collectInvalidDatetimeFields(v any, prefix string, invalid *[]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			if isDatetimeKey(k) {
				if s, ok := child.(string); ok {
					if _, err := time.Parse(time.RFC3339, s); err != nil {
						*invalid = append(*invalid, fmt.Sprintf("%s=%q", path, s))
					}
				}
			}
			collectInvalidDatetimeFields(child, path, invalid)
		}
	case []any:
		for i, child := range val {
			collectInvalidDatetimeFields(child, fmt.Sprintf("%s[%d]", prefix, i), invalid)
		}
	}
}

func isDatetimeKey(key string) bool {
	return strings.HasSuffix(key, "_at") || strings.HasSuffix(key, "_on")
}

// CheckIntegerID verifies the top-level id field is a JSON number, not a string.
func CheckIntegerID() CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkIntegerID,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		idRaw, ok := payload["id"]
		if !ok {
			return CheckResult{
				Name:   checkIntegerID,
				Passed: false,
				Diff:   "missing field: id",
			}
		}

		switch idRaw.(type) {
		case float64, json.Number:
			return CheckResult{Name: checkIntegerID, Passed: true}
		default:
			return CheckResult{
				Name:   checkIntegerID,
				Passed: false,
				Diff:   fmt.Sprintf("id is not a JSON number, got %T", idRaw),
			}
		}
	}
}

// CheckRequiredFields verifies every named field is present and non-null in the top-level JSON object.
func CheckRequiredFields(fields ...string) CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkRequiredFields,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		var missing []string
		for _, field := range fields {
			val, ok := payload[field]
			if !ok {
				missing = append(missing, field)
				continue
			}
			if val == nil {
				missing = append(missing, field+"(null)")
			}
		}

		if len(missing) == 0 {
			return CheckResult{Name: checkRequiredFields, Passed: true}
		}
		return CheckResult{
			Name:   checkRequiredFields,
			Passed: false,
			Diff:   fmt.Sprintf("missing or null fields: %s", strings.Join(missing, ", ")),
		}
	}
}

// CheckHTTPSURLField verifies the named field exists and starts with "https://".
func CheckHTTPSURLField(field string) CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkHTTPSURLField,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		val, ok := payload[field]
		if !ok {
			return CheckResult{
				Name:   checkHTTPSURLField,
				Passed: false,
				Diff:   fmt.Sprintf("missing field: %s", field),
			}
		}

		urlStr, ok := val.(string)
		if !ok {
			return CheckResult{
				Name:   checkHTTPSURLField,
				Passed: false,
				Diff:   fmt.Sprintf("field %s is not a string", field),
			}
		}

		if !strings.HasPrefix(urlStr, "https://") {
			return CheckResult{
				Name:   checkHTTPSURLField,
				Passed: false,
				Diff:   fmt.Sprintf("field %s does not start with https://, got %q", field, urlStr),
			}
		}

		return CheckResult{Name: checkHTTPSURLField, Passed: true}
	}
}

// CheckDocumentationURL verifies documentation_url is present in the response body.
func CheckDocumentationURL() CheckFunc {
	return func(_ *http.Response, body []byte) CheckResult {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return CheckResult{
				Name:   checkDocumentationURL,
				Passed: false,
				Diff:   fmt.Sprintf("invalid JSON: %v", err),
			}
		}

		doc, ok := payload["documentation_url"]
		if !ok {
			return CheckResult{
				Name:   checkDocumentationURL,
				Passed: false,
				Diff:   "missing field: documentation_url",
			}
		}

		if doc == nil {
			return CheckResult{
				Name:   checkDocumentationURL,
				Passed: false,
				Diff:   "field documentation_url is null",
			}
		}

		return CheckResult{Name: checkDocumentationURL, Passed: true}
	}
}
