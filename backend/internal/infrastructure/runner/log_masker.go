package runner

import (
	"net/url"
	"strings"
)

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
	variants := []string{value}
	if encoded := url.QueryEscape(value); encoded != value {
		variants = append(variants, encoded)
	}
	if pathEncoded := url.PathEscape(value); pathEncoded != value {
		variants = append(variants, pathEncoded)
	}

	seen := make(map[string]struct{}, len(variants)*3)
	for _, variant := range variants {
		for _, candidate := range []string{variant, strings.ToLower(variant), strings.ToUpper(variant)} {
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			chunk = strings.ReplaceAll(chunk, candidate, "***")
		}
	}
	return chunk
}
