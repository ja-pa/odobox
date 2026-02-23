package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (b *Backend) getSettings() (SettingsResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return SettingsResponse{}, err
	}
	if _, err := b.loadCleanerConfig(cfg); err != nil {
		return SettingsResponse{}, err
	}
	if _, err := b.loadParserConfig(cfg); err != nil {
		return SettingsResponse{}, err
	}
	if _, err := b.loadSMSParserConfig(cfg); err != nil {
		return SettingsResponse{}, err
	}
	if _, err := b.defaultTranscriptVersion(cfg); err != nil {
		return SettingsResponse{}, err
	}

	poll := cfg.get("app", "poll_interval_minutes", "5")
	pollInt, err := strconv.Atoi(strings.TrimSpace(poll))
	pollValue := any(strings.TrimSpace(poll))
	if err == nil {
		pollValue = pollInt
	}

	settings := map[string]map[string]any{
		"imap": {
			"host":     cfg.get("imap", "host", ""),
			"port":     cfg.get("imap", "port", "993"),
			"ssl":      cfg.get("imap", "ssl", "true"),
			"username": cfg.get("imap", "username", ""),
			"password": cfg.get("imap", "password", ""),
			"folder":   cfg.get("imap", "folder", "INBOX"),
		},
		"message_cleaner": {
			"keep_line_regex":      cfg.get("message_cleaner", "keep_line_regex", `^v\d+:\s*.+$`),
			"version_v1_regex":     cfg.get("message_cleaner", "version_v1_regex", `(?is)(?:^|\n)\s*v1:\s*(?P<content>.*?)(?=\n\s*v2:\s*|\Z)`),
			"version_v2_regex":     cfg.get("message_cleaner", "version_v2_regex", `(?is)(?:^|\n)\s*v2:\s*(?P<content>.*?)(?=\n\s*v1:\s*|\Z)`),
			"remove_regexes":       cfg.get("message_cleaner", "remove_regexes", ""),
			"collapse_blank_lines": cfg.get("message_cleaner", "collapse_blank_lines", "true"),
		},
		"voicemail_parser": {
			"caller_phone_regex": cfg.get("voicemail_parser", "caller_phone_regex", `^Hlasova zprava\s+(\+?\d+)\s+[-=]+>\s+\d+`),
		},
		"sms_parser": {
			"text_extract_regex": cfg.get("sms_parser", "text_extract_regex", `(?is)TEXT:\s*["“]?(?:Message)?(?P<content>[^"\r\n]+?)["”]`),
		},
		"app": {
			"poll_interval_minutes":      pollValue,
			"default_transcript_version": cfg.get("app", "default_transcript_version", "both"),
			"sms_identity_text":          cfg.get("app", "sms_identity_text", ""),
		},
		"odorik": {
			"pin":        cfg.get("odorik", "pin", ""),
			"user":       cfg.get("odorik", "user", ""),
			"password":   cfg.get("odorik", "password", ""),
			"account_id": cfg.get("odorik", "account_id", ""),
			"api_pin":    cfg.get("odorik", "api_pin", ""),
			"sender_id":  cfg.get("odorik", "sender_id", ""),
		},
	}

	return SettingsResponse{Settings: settings, EditableSections: []string{"app", "imap", "message_cleaner", "odorik", "voicemail_parser", "sms_parser"}}, nil
}

func (b *Backend) patchSettings(req PatchSettingsRequest) (PatchSettingsResponse, error) {
	if req.Settings == nil {
		return PatchSettingsResponse{}, fmt.Errorf("settings must be a JSON object")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return PatchSettingsResponse{}, err
	}
	editable := map[string]bool{"imap": true, "message_cleaner": true, "voicemail_parser": true, "sms_parser": true, "app": true, "odorik": true}
	restricted := map[string]map[string]bool{
		"app": {"poll_interval_minutes": true, "default_transcript_version": true, "sms_identity_text": true},
		"odorik": {
			"pin": true, "user": true, "password": true, "sender_id": true,
			"account_id": true, "api_pin": true,
		},
	}

	for section, updates := range req.Settings {
		section = strings.ToLower(strings.TrimSpace(section))
		if !editable[section] {
			return PatchSettingsResponse{}, fmt.Errorf("only these sections can be updated: app,imap,message_cleaner,odorik,voicemail_parser,sms_parser")
		}
		for key, value := range updates {
			k := strings.ToLower(strings.TrimSpace(key))
			if k == "" {
				return PatchSettingsResponse{}, fmt.Errorf("invalid key in section '%s'", section)
			}
			if allowed, ok := restricted[section]; ok {
				if !allowed[k] {
					return PatchSettingsResponse{}, fmt.Errorf("only these keys can be updated in '%s'", section)
				}
			}
			stringValue, err := anyToINIValue(value)
			if err != nil {
				return PatchSettingsResponse{}, err
			}
			cfg.set(section, k, stringValue)
		}
	}

	if _, err := b.loadIMAPConfig(cfg); err != nil {
		return PatchSettingsResponse{}, err
	}
	if _, err := b.loadCleanerConfig(cfg); err != nil {
		return PatchSettingsResponse{}, err
	}
	if _, err := b.loadParserConfig(cfg); err != nil {
		return PatchSettingsResponse{}, err
	}
	if _, err := b.loadSMSParserConfig(cfg); err != nil {
		return PatchSettingsResponse{}, err
	}
	pollRaw := strings.TrimSpace(cfg.get("app", "poll_interval_minutes", "5"))
	pollInterval, err := strconv.Atoi(pollRaw)
	if err != nil || pollInterval < 1 {
		return PatchSettingsResponse{}, fmt.Errorf("app.poll_interval_minutes must be a positive integer")
	}
	transcriptVersion := strings.ToLower(strings.TrimSpace(cfg.get("app", "default_transcript_version", "both")))
	if transcriptVersion != "v1" && transcriptVersion != "v2" && transcriptVersion != "both" {
		return PatchSettingsResponse{}, fmt.Errorf("app.default_transcript_version must be one of: v1,v2,both")
	}
	smsIdentityText := strings.TrimSpace(cfg.get("app", "sms_identity_text", ""))
	if len([]rune(smsIdentityText)) > 80 {
		return PatchSettingsResponse{}, fmt.Errorf("app.sms_identity_text must be at most 80 characters")
	}

	cfgPath := b.resolveConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return PatchSettingsResponse{}, err
	}
	if err := cfg.write(cfgPath); err != nil {
		return PatchSettingsResponse{}, err
	}
	fresh, err := b.getSettings()
	if err != nil {
		return PatchSettingsResponse{}, err
	}
	return PatchSettingsResponse{Status: "ok", Settings: fresh.Settings}, nil
}

func anyToINIValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("list values must contain only strings")
			}
			if strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}
		return strings.Join(parts, "\n"), nil
	case []string:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if strings.TrimSpace(item) != "" {
				parts = append(parts, strings.TrimSpace(item))
			}
		}
		return strings.Join(parts, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported value type")
	}
}
