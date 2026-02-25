package core

import "gopkg.in/ini.v1"

type appConfig struct {
	file *ini.File
}

type imapConfig struct {
	Host     string
	Port     int
	SSL      bool
	Username string
	Password string
	Folder   string
}

type cleanerConfig struct {
	KeepLineRegex      string
	RemoveRegexes      []string
	CollapseBlankLines bool
	VersionV1Regex     string
	VersionV2Regex     string
}

type parserConfig struct {
	CallerPhoneRegex string
}

type smsParserConfig struct {
	TextExtractRegex string
}

type smsConfig struct {
	User      string
	Password  string
	DefaultID string
}
