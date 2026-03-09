package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCleanerConfig_MultilineRemoveRegexes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.ini")
	content := `[message_cleaner]
keep_line_regex = ^v\d+:\s*.+$
remove_regexes =
    ^Prehrana zprava.*$
    ^https?://\S+$
version_v1_regex = (?is)(?:^|\n)\s*v1:\s*(?P<content>.*?)(?=\n\s*v2:\s*|\Z)
version_v2_regex = (?is)(?:^|\n)(?:\s*v2:\s*|---\s*Přepis hlasové zprávy\s*\(google_v2\)\s*---\s*)(?P<content>.*?)(?:\n\s*v1:\s*|\nVíce informací o přepisu nahrávky na text:|$)
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	b := NewBackend(cfgPath)
	cfg, err := b.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	cc, err := b.loadCleanerConfig(cfg)
	if err != nil {
		t.Fatalf("loadCleanerConfig: %v", err)
	}
	if len(cc.RemoveRegexes) != 2 {
		t.Fatalf("expected 2 remove regexes, got %d: %#v", len(cc.RemoveRegexes), cc.RemoveRegexes)
	}
	if cc.RemoveRegexes[0] != "^Prehrana zprava.*$" {
		t.Fatalf("unexpected first regex: %q", cc.RemoveRegexes[0])
	}
	if cc.RemoveRegexes[1] != "^https?://\\S+$" {
		t.Fatalf("unexpected second regex: %q", cc.RemoveRegexes[1])
	}
}

func TestConfigWriteRead_RoundTripMultilineAndSMSRegex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.ini")
	b := NewBackend(cfgPath)

	cfg, err := b.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	cfg.set("message_cleaner", "remove_regexes", "^A$\n^B$")
	cfg.set("sms_parser", "text_extract_regex", `(?is)TEXT:\s*["“]?(?:Message)?(?P<content>[^"\r\n]+?)["”]`)

	if err := cfg.write(cfgPath); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg2, err := b.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig roundtrip: %v", err)
	}
	cc, err := b.loadCleanerConfig(cfg2)
	if err != nil {
		t.Fatalf("loadCleanerConfig: %v", err)
	}
	if len(cc.RemoveRegexes) != 2 || cc.RemoveRegexes[0] != "^A$" || cc.RemoveRegexes[1] != "^B$" {
		t.Fatalf("unexpected remove regexes after roundtrip: %#v", cc.RemoveRegexes)
	}

	spc, err := b.loadSMSParserConfig(cfg2)
	if err != nil {
		t.Fatalf("loadSMSParserConfig: %v", err)
	}
	expected := `(?is)TEXT:\s*["“]?(?:Message)?(?P<content>[^"\r\n]+?)["”]`
	if spc.TextExtractRegex != expected {
		t.Fatalf("unexpected sms parser regex after roundtrip: %q", spc.TextExtractRegex)
	}
}
