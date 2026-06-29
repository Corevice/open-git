package actions

func MatchLabels(requested []string, available []string) bool {
	if len(requested) == 0 {
		return true
	}

	availableSet := make(map[string]struct{}, len(available))
	for _, label := range available {
		availableSet[label] = struct{}{}
	}
	for _, label := range requested {
		if _, ok := availableSet[label]; !ok {
			return false
		}
	}
	return true
}

func IsGitHubHosted(label string) bool {
	switch label {
	case "ubuntu-latest", "ubuntu-22.04", "ubuntu-20.04", "windows-latest", "macos-latest":
		return true
	default:
		return false
	}
}

func UsesActAdapter(labels []string) bool {
	if len(labels) == 0 {
		return true
	}
	for _, label := range labels {
		switch label {
		case "ubuntu-latest", "ubuntu-22.04", "ubuntu-20.04":
			return true
		}
	}
	return false
}
