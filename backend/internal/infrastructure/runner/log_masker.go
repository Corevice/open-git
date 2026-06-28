package runner

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

const maskReplacement = "***"

func MaskSecrets(chunk string, secretValues []string) string {
	for _, value := range secretValues {
		if value == "" {
			continue
		}
		chunk = maskSecretValue(chunk, value)
	}
	return chunk
}

func maskSecretValue(chunk, value string) string {
	if value == "" || value == maskReplacement {
		return chunk
	}

	variants := []string{value}
	if encoded := url.QueryEscape(value); encoded != value {
		variants = append(variants, encoded)
	}
	if pathEncoded := url.PathEscape(value); pathEncoded != value {
		variants = append(variants, pathEncoded)
	}

	seen := make(map[string]struct{}, len(variants))
	for _, variant := range variants {
		if variant == "" || variant == maskReplacement {
			continue
		}
		if _, ok := seen[variant]; ok {
			continue
		}
		seen[variant] = struct{}{}
		chunk = strings.ReplaceAll(chunk, variant, maskReplacement)
		chunk = replaceCaseInsensitive(chunk, variant, maskReplacement)
	}
	return chunk
}

func replaceCaseInsensitive(chunk, needle, replacement string) string {
	if needle == "" || needle == maskReplacement {
		return chunk
	}
	var b strings.Builder
	b.Grow(len(chunk))
	for i := 0; i < len(chunk); {
		if i+len(needle) <= len(chunk) && strings.EqualFold(chunk[i:i+len(needle)], needle) {
			b.WriteString(replacement)
			i += len(needle)
			continue
		}
		r, size := utf8.DecodeRuneInString(chunk[i:])
		b.WriteRune(r)
		i += size
	}
	return b.String()
}
