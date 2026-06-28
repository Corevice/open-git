package secret

import "strings"

func MaskSecretValues(line string, secrets []string) string {
	masked := line
	for _, value := range secrets {
		if value == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, value, "***")
	}
	return masked
}
