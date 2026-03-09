package core

import "testing"

func TestExtractVersion_V2GoogleVoicemailBlock(t *testing.T) {
	t.Parallel()

	cfg := cleanerConfig{
		KeepLineRegex:      `^v\d+:\s*.+$`,
		CollapseBlankLines: true,
		VersionV1Regex:     `(?is)(?:^|\n)\s*v1:\s*(?P<content>.*?)(?=\n\s*v2:\s*|\Z)`,
		VersionV2Regex:     `(?is)(?:^|\n)(?:\s*v2:\s*|---\s*Přepis hlasové zprávy\s*\(google_v2\)\s*---\s*)(?P<content>.*?)(?:\n\s*v1:\s*|\nVíce informací o přepisu nahrávky na text:|$)`,
		RemoveRegexes: []string{
			`^Prehrana zprava.*$`,
			`^--- Přepis hlasové zprávy.*$`,
			`^Více informací o přepisu nahrávky na text:$`,
			`^https?://\S+$`,
		},
	}

	text := "Prehrana zprava cislo 1-Hlasova schranka,voicemail v priloze.\n\n--- Přepis hlasové zprávy (google_v2) ---\nJo, jenom jsem chtěl říct, že v pondělí nedojdu.\n\nVíce informací o přepisu nahrávky na text:\nhttps://forum.odorik.cz/viewtopic.php?p=46775#p46775\n"

	got := extractVersion(text, cfg, "v2")
	want := "v2: Jo, jenom jsem chtěl říct, že v pondělí nedojdu."
	if got != want {
		t.Fatalf("unexpected extracted version:\nwant: %q\ngot:  %q", want, got)
	}
}
