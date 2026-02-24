package core

import (
	"regexp"
	"strings"
)

func normalizeVersion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "v1":
		return "v1"
	case "2", "v2":
		return "v2"
	case "both", "all":
		return "all"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func cleanMessageText(text string, cfg cleanerConfig) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	keepRe := regexp.MustCompile(cfg.KeepLineRegex)
	lines := strings.Split(normalized, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if keepRe.MatchString(line) {
			kept = append(kept, strings.TrimSpace(line))
		}
	}
	if len(kept) > 0 {
		return strings.Join(kept, "\n")
	}
	cleaned := normalized
	for _, pattern := range cfg.RemoveRegexes {
		re := regexp.MustCompile(ensureMultilinePattern(pattern))
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	cleaned = stripOdorikFooterNoise(cleaned)
	if cfg.CollapseBlankLines {
		re := regexp.MustCompile(`\n{3,}`)
		cleaned = re.ReplaceAllString(cleaned, "\n\n")
	}
	return strings.TrimSpace(cleaned)
}

func cleanupWithRegexes(text string, cfg cleanerConfig) string {
	cleaned := text
	for _, pattern := range cfg.RemoveRegexes {
		re := regexp.MustCompile(ensureMultilinePattern(pattern))
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	cleaned = stripOdorikFooterNoise(cleaned)
	if cfg.CollapseBlankLines {
		re := regexp.MustCompile(`\n{3,}`)
		cleaned = re.ReplaceAllString(cleaned, "\n\n")
	}
	return strings.TrimSpace(cleaned)
}

func ensureMultilinePattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "(?m)") || strings.HasPrefix(trimmed, "(?im)") || strings.HasPrefix(trimmed, "(?mi)") {
		return trimmed
	}
	return "(?m)" + trimmed
}

func stripOdorikFooterNoise(text string) string {
	cleaned := text
	footerLine := regexp.MustCompile(`(?mi)^Více informací o přepisu nahrávky na text:.*$`)
	cleaned = footerLine.ReplaceAllString(cleaned, "")
	urlLine := regexp.MustCompile(`(?m)^https?://\S+$`)
	cleaned = urlLine.ReplaceAllString(cleaned, "")
	return cleaned
}

func extractVersion(text string, cfg cleanerConfig, version string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	if version == "all" {
		parts := []string{}
		if v1 := extractVersion(normalized, cfg, "v1"); v1 != "" {
			parts = append(parts, v1)
		}
		if v2 := extractVersion(normalized, cfg, "v2"); v2 != "" {
			parts = append(parts, v2)
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
		return cleanMessageText(normalized, cfg)
	}
	if version != "v1" && version != "v2" {
		return cleanMessageText(normalized, cfg)
	}
	pattern := cfg.normalizePattern(cfg.VersionV1Regex)
	if version == "v2" {
		pattern = cfg.normalizePattern(cfg.VersionV2Regex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		if extracted, ok := extractVersionByMarkers(normalized, cfg, version); ok {
			return extracted
		}
		return ""
	}
	m := re.FindStringSubmatch(normalized)
	if m == nil {
		if extracted, ok := extractVersionByMarkers(normalized, cfg, version); ok {
			return extracted
		}
		return ""
	}
	idx := re.SubexpIndex("content")
	if idx < 0 {
		if len(m) < 2 {
			return ""
		}
		idx = 1
	}
	content := cleanupWithRegexes(strings.TrimSpace(m[idx]), cfg)
	if content == "" {
		return ""
	}
	return version + ": " + content
}

func extractVersionByMarkers(text string, cfg cleanerConfig, version string) (string, bool) {
	targetPrefix := version + ":"
	lines := strings.Split(text, "\n")
	collecting := false
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowered := strings.ToLower(trimmed)
		if strings.HasPrefix(lowered, targetPrefix) {
			collecting = true
			content := strings.TrimSpace(trimmed[len(targetPrefix):])
			if content != "" {
				out = append(out, content)
			}
			continue
		}
		if strings.HasPrefix(lowered, "v1:") || strings.HasPrefix(lowered, "v2:") {
			if collecting {
				break
			}
			continue
		}
		if collecting {
			out = append(out, line)
		}
	}

	if len(out) == 0 {
		return "", false
	}
	content := cleanupWithRegexes(strings.TrimSpace(strings.Join(out, "\n")), cfg)
	if content == "" {
		return "", false
	}
	return version + ": " + content, true
}
