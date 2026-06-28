package runner

import "strings"

func MaskSecrets(chunk string, secretValues []string) string {
	for _, value := range secretValues {
		if value == "" {
			continue
		}
		chunk = strings.ReplaceAll(chunk, value, "***")
	}
	return chunk
}
