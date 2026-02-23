package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func (b *Backend) resolveConfigPath() string {
	if env := strings.TrimSpace(os.Getenv("ODORIK_CONFIG")); env != "" {
		return env
	}
	if b.configPath != "" {
		return b.configPath
	}
	if _, err := os.Stat(defaultCfgPath); err == nil {
		return defaultCfgPath
	}
	fallback := "../../odorik-backend/config.ini"
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return defaultCfgPath
}

func (b *Backend) loadConfig() (*appConfig, error) {
	cfgPath := b.resolveConfigPath()
	cfg := &appConfig{sections: map[string]map[string]string{}}
	f, err := os.Open(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	parseINI(data, cfg)
	return cfg, nil
}

func parseINI(data []byte, cfg *appConfig) {
	lines := strings.Split(string(data), "\n")
	section := ""
	lastKey := ""

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToLower(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			if _, ok := cfg.sections[section]; !ok {
				cfg.sections[section] = map[string]string{}
			}
			lastKey = ""
			continue
		}
		if section == "" {
			continue
		}
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && lastKey != "" {
			prev := cfg.sections[section][lastKey]
			part := strings.TrimSpace(line)
			if prev == "" {
				cfg.sections[section][lastKey] = part
			} else {
				cfg.sections[section][lastKey] = prev + "\n" + part
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.TrimSpace(v)
		cfg.sections[section][key] = val
		lastKey = key
	}
}

func (cfg *appConfig) section(name string) map[string]string {
	if sec, ok := cfg.sections[strings.ToLower(name)]; ok {
		return sec
	}
	return map[string]string{}
}

func (cfg *appConfig) get(section, key, fallback string) string {
	sec := cfg.section(section)
	if val, ok := sec[strings.ToLower(key)]; ok {
		return val
	}
	return fallback
}

func (cfg *appConfig) set(section, key, value string) {
	secName := strings.ToLower(section)
	if _, ok := cfg.sections[secName]; !ok {
		cfg.sections[secName] = map[string]string{}
	}
	cfg.sections[secName][strings.ToLower(key)] = value
}

func (cfg *appConfig) write(path string) error {
	var out strings.Builder
	sections := make([]string, 0, len(cfg.sections))
	for name := range cfg.sections {
		sections = append(sections, name)
	}
	sort.Strings(sections)

	for _, section := range sections {
		out.WriteString("[")
		out.WriteString(section)
		out.WriteString("]\n")

		keys := make([]string, 0, len(cfg.sections[section]))
		for key := range cfg.sections[section] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := cfg.sections[section][key]
			if strings.Contains(value, "\n") {
				out.WriteString(key)
				out.WriteString(" =\n")
				for _, line := range strings.Split(value, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					out.WriteString("    ")
					out.WriteString(line)
					out.WriteString("\n")
				}
			} else {
				out.WriteString(key)
				out.WriteString(" = ")
				out.WriteString(value)
				out.WriteString("\n")
			}
		}
		out.WriteString("\n")
	}

	return os.WriteFile(path, []byte(out.String()), 0o600)
}

func (b *Backend) resolveDBPath(cfg *appConfig) string {
	if env := strings.TrimSpace(os.Getenv("ODORIK_DB")); env != "" {
		return env
	}
	if db := strings.TrimSpace(cfg.get("app", "db", "")); db != "" {
		return db
	}
	if _, err := os.Stat(defaultDBPath); err == nil {
		return defaultDBPath
	}
	fallback := "../../odorik-backend/voicemail.db"
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return defaultDBPath
}

func parseBoolLoose(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func (b *Backend) loadIMAPConfig(cfg *appConfig) (imapConfig, error) {
	host := strings.TrimSpace(cfg.get("imap", "host", ""))
	username := strings.TrimSpace(cfg.get("imap", "username", ""))
	password := strings.TrimSpace(cfg.get("imap", "password", ""))
	if host == "" || username == "" || password == "" {
		return imapConfig{}, fmt.Errorf("config [imap] missing required keys: host, username, password")
	}
	port := 993
	if raw := strings.TrimSpace(cfg.get("imap", "port", "993")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return imapConfig{}, fmt.Errorf("imap.port must be positive integer")
		}
		port = parsed
	}
	folder := strings.TrimSpace(cfg.get("imap", "folder", "INBOX"))
	if folder == "" {
		folder = "INBOX"
	}
	return imapConfig{
		Host:     host,
		Port:     port,
		SSL:      parseBoolLoose(cfg.get("imap", "ssl", "true"), true),
		Username: username,
		Password: password,
		Folder:   folder,
	}, nil
}

func (b *Backend) loadCleanerConfig(cfg *appConfig) (cleanerConfig, error) {
	cc := cleanerConfig{
		KeepLineRegex:      cfg.get("message_cleaner", "keep_line_regex", `^v\d+:\s*.+$`),
		CollapseBlankLines: parseBoolLoose(cfg.get("message_cleaner", "collapse_blank_lines", "true"), true),
		VersionV1Regex:     cfg.get("message_cleaner", "version_v1_regex", `(?is)(?:^|\n)\s*v1:\s*(?P<content>.*?)(?=\n\s*v2:\s*|\Z)`),
		VersionV2Regex:     cfg.get("message_cleaner", "version_v2_regex", `(?is)(?:^|\n)\s*v2:\s*(?P<content>.*?)(?=\n\s*v1:\s*|\Z)`),
	}
	removeRaw := cfg.get("message_cleaner", "remove_regexes", "")
	for _, line := range strings.Split(removeRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cc.RemoveRegexes = append(cc.RemoveRegexes, trimmed)
		}
	}
	patterns := append([]string{cc.KeepLineRegex}, cc.RemoveRegexes...)
	for _, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return cleanerConfig{}, fmt.Errorf("invalid message_cleaner regex: %w", err)
		}
	}
	return cc, nil
}

func (cc cleanerConfig) normalizePattern(pattern string) string {
	return strings.ReplaceAll(pattern, `\Z`, `$`)
}

func (b *Backend) loadParserConfig(cfg *appConfig) (parserConfig, error) {
	pc := parserConfig{CallerPhoneRegex: cfg.get("voicemail_parser", "caller_phone_regex", `^Hlasova zprava\s+(\+?\d+)\s+[-=]+>\s+\d+`)}
	if _, err := regexp.Compile(pc.CallerPhoneRegex); err != nil {
		return parserConfig{}, fmt.Errorf("invalid voicemail_parser regex: %w", err)
	}
	return pc, nil
}

func (b *Backend) loadSMSParserConfig(cfg *appConfig) (smsParserConfig, error) {
	spc := smsParserConfig{
		TextExtractRegex: cfg.get("sms_parser", "text_extract_regex", `(?is)TEXT:\s*["“]?(?:Message)?(?P<content>[^"\r\n]+?)["”]`),
	}
	if _, err := regexp.Compile(spc.TextExtractRegex); err != nil {
		return smsParserConfig{}, fmt.Errorf("invalid sms_parser regex: %w", err)
	}
	return spc, nil
}

func (b *Backend) loadSMSConfig(cfg *appConfig) smsConfig {
	user := strings.TrimSpace(cfg.get("odorik", "user", ""))
	if user == "" {
		user = strings.TrimSpace(cfg.get("odorik", "account_id", ""))
	}
	if user == "" {
		user = strings.TrimSpace(cfg.get("imap", "username", ""))
	}
	password := strings.TrimSpace(cfg.get("odorik", "password", ""))
	if password == "" {
		password = strings.TrimSpace(cfg.get("odorik", "api_pin", ""))
	}
	if password == "" {
		password = strings.TrimSpace(cfg.get("odorik", "pin", ""))
	}
	defaultID := strings.TrimSpace(cfg.get("odorik", "sender_id", ""))
	if defaultID == "" {
		defaultID = strings.TrimSpace(cfg.get("odorik", "default_sender", ""))
	}
	return smsConfig{User: user, Password: password, DefaultID: defaultID}
}

func (b *Backend) defaultTranscriptVersion(cfg *appConfig) (string, error) {
	v := strings.ToLower(strings.TrimSpace(cfg.get("app", "default_transcript_version", "both")))
	switch v {
	case "both":
		return "all", nil
	case "v1", "v2":
		return v, nil
	default:
		return "", fmt.Errorf("app.default_transcript_version must be one of: v1,v2,both")
	}
}
