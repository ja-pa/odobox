package core

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func validateContact(fullName, phone string) (string, string, error) {
	name := strings.TrimSpace(fullName)
	phoneDisplay := strings.TrimSpace(phone)
	if name == "" {
		return "", "", fmt.Errorf("full_name is required")
	}
	if phoneDisplay == "" {
		return "", "", fmt.Errorf("phone is required")
	}
	normalized := normalizePhoneLoose(phoneDisplay)
	if len(normalized) < 6 || len(normalized) > 15 {
		return "", "", fmt.Errorf("phone must contain 6 to 15 digits")
	}
	return name, phoneDisplay, nil
}

type parsedVCard struct {
	FullName string
	Phones   []string
	Email    string
	Org      string
	Note     string
	Raw      string
}

func unfoldVCardLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(out) > 0 {
			out[len(out)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		out = append(out, line)
	}
	return out
}

func unescapeVCardValue(v string) string {
	repl := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return strings.TrimSpace(repl.Replace(v))
}

func parseVCFContacts(content string) []parsedVCard {
	normalized := strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	lines = unfoldVCardLines(lines)

	contacts := []parsedVCard{}
	var current *parsedVCard
	var rawLines []string

	flush := func() {
		if current == nil {
			return
		}
		current.Raw = strings.TrimSpace(strings.Join(rawLines, "\n"))
		if current.FullName != "" && len(current.Phones) > 0 {
			contacts = append(contacts, *current)
		}
		current = nil
		rawLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "BEGIN:VCARD") {
			flush()
			current = &parsedVCard{}
			rawLines = append(rawLines, line)
			continue
		}
		if current == nil {
			continue
		}
		rawLines = append(rawLines, line)
		if strings.EqualFold(trimmed, "END:VCARD") {
			flush()
			continue
		}
		keyPart, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(strings.Split(keyPart, ";")[0]))
		value = unescapeVCardValue(value)
		switch key {
		case "FN":
			if current.FullName == "" {
				current.FullName = value
			}
		case "N":
			if current.FullName == "" {
				current.FullName = strings.TrimSpace(strings.ReplaceAll(value, ";", " "))
			}
		case "TEL":
			if value != "" {
				current.Phones = append(current.Phones, value)
			}
		case "EMAIL":
			if current.Email == "" {
				current.Email = value
			}
		case "ORG":
			if current.Org == "" {
				current.Org = value
			}
		case "NOTE":
			if current.Note == "" {
				current.Note = value
			}
		}
	}
	flush()
	return contacts
}

func (b *Backend) listContacts() ([]ContactInfo, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, full_name, phone_display, COALESCE(email, ''), COALESCE(org, ''), COALESCE(note, ''), COALESCE(vcard, ''), updated_at FROM contacts ORDER BY full_name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ContactInfo{}
	for rows.Next() {
		var c ContactInfo
		if err := rows.Scan(&c.ID, &c.FullName, &c.Phone, &c.Email, &c.Org, &c.Note, &c.VCard, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (b *Backend) createContact(req CreateContactRequest) (ContactInfo, error) {
	name, phoneDisplay, err := validateContact(req.FullName, req.Phone)
	if err != nil {
		return ContactInfo{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return ContactInfo{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ContactInfo{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	normalized := normalizePhoneLoose(phoneDisplay)
	res, err := db.Exec(
		`INSERT INTO contacts(full_name, phone_display, phone_normalized, email, org, note, vcard, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'manual', ?, ?)`,
		name, phoneDisplay, normalized, strings.TrimSpace(req.Email), strings.TrimSpace(req.Org), strings.TrimSpace(req.Note), "", now, now,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ContactInfo{}, fmt.Errorf("contact with this phone already exists")
		}
		return ContactInfo{}, err
	}
	id64, _ := res.LastInsertId()
	return ContactInfo{
		ID:        int(id64),
		FullName:  name,
		Phone:     phoneDisplay,
		Email:     strings.TrimSpace(req.Email),
		Org:       strings.TrimSpace(req.Org),
		Note:      strings.TrimSpace(req.Note),
		VCard:     "",
		UpdatedAt: now,
	}, nil
}

func (b *Backend) updateContact(req UpdateContactRequest) (ContactInfo, error) {
	if req.ID < 1 {
		return ContactInfo{}, fmt.Errorf("contact id must be positive")
	}
	name, phoneDisplay, err := validateContact(req.FullName, req.Phone)
	if err != nil {
		return ContactInfo{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return ContactInfo{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ContactInfo{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	normalized := normalizePhoneLoose(phoneDisplay)
	res, err := db.Exec(
		`UPDATE contacts SET full_name=?, phone_display=?, phone_normalized=?, email=?, org=?, note=?, source='manual', updated_at=? WHERE id=?`,
		name, phoneDisplay, normalized, strings.TrimSpace(req.Email), strings.TrimSpace(req.Org), strings.TrimSpace(req.Note), now, req.ID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ContactInfo{}, fmt.Errorf("contact with this phone already exists")
		}
		return ContactInfo{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ContactInfo{}, fmt.Errorf("contact not found")
	}
	var vcard string
	_ = db.QueryRow(`SELECT COALESCE(vcard, '') FROM contacts WHERE id = ?`, req.ID).Scan(&vcard)
	return ContactInfo{
		ID:        req.ID,
		FullName:  name,
		Phone:     phoneDisplay,
		Email:     strings.TrimSpace(req.Email),
		Org:       strings.TrimSpace(req.Org),
		Note:      strings.TrimSpace(req.Note),
		VCard:     vcard,
		UpdatedAt: now,
	}, nil
}

func (b *Backend) deleteContact(id int) (DeleteContactResponse, error) {
	if id < 1 {
		return DeleteContactResponse{}, fmt.Errorf("contact id must be positive")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return DeleteContactResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return DeleteContactResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return DeleteContactResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return DeleteContactResponse{}, fmt.Errorf("contact not found")
	}
	return DeleteContactResponse{Status: "ok", ID: id}, nil
}

func (b *Backend) importVCF(req ImportVCFRequest) (ImportVCFResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return ImportVCFResponse{}, fmt.Errorf("vcf content is required")
	}
	entries := parseVCFContacts(content)
	if len(entries) == 0 {
		return ImportVCFResponse{}, fmt.Errorf("no valid contacts found in VCF")
	}

	cfg, err := b.loadConfig()
	if err != nil {
		return ImportVCFResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return ImportVCFResponse{}, err
	}
	defer db.Close()

	imported := 0
	updated := 0
	skipped := 0
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	for _, entry := range entries {
		if entry.FullName == "" || len(entry.Phones) == 0 {
			skipped++
			continue
		}
		for _, phone := range entry.Phones {
			phoneDisplay := strings.TrimSpace(phone)
			normalized := normalizePhoneLoose(phoneDisplay)
			if len(normalized) < 6 {
				skipped++
				continue
			}
			var existingID int
			err := db.QueryRow(`SELECT id FROM contacts WHERE phone_normalized = ?`, normalized).Scan(&existingID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return ImportVCFResponse{}, err
			}
			if errors.Is(err, sql.ErrNoRows) {
				_, insErr := db.Exec(
					`INSERT INTO contacts(full_name, phone_display, phone_normalized, email, org, note, vcard, source, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?, 'vcf', ?, ?)`,
					entry.FullName, phoneDisplay, normalized, entry.Email, entry.Org, entry.Note, entry.Raw, now, now,
				)
				if insErr != nil {
					skipped++
					continue
				}
				imported++
				continue
			}
			_, updErr := db.Exec(
				`UPDATE contacts SET full_name=?, phone_display=?, email=?, org=?, note=?, vcard=?, source='vcf', updated_at=? WHERE id=?`,
				entry.FullName, phoneDisplay, entry.Email, entry.Org, entry.Note, entry.Raw, now, existingID,
			)
			if updErr != nil {
				skipped++
				continue
			}
			updated++
		}
	}

	return ImportVCFResponse{
		Status:    "ok",
		Imported:  imported,
		Updated:   updated,
		Skipped:   skipped,
		Processed: len(entries),
	}, nil
}

func escapeVCardValue(v string) string {
	repl := strings.NewReplacer(`\`, `\\`, "\n", `\n`, ",", `\,`, ";", `\;`)
	return repl.Replace(strings.TrimSpace(v))
}

func (b *Backend) exportVCF() (ExportVCFResponse, error) {
	contacts, err := b.listContacts()
	if err != nil {
		return ExportVCFResponse{}, err
	}
	var out strings.Builder
	for _, c := range contacts {
		if strings.TrimSpace(c.FullName) == "" || strings.TrimSpace(c.Phone) == "" {
			continue
		}
		out.WriteString("BEGIN:VCARD\r\n")
		out.WriteString("VERSION:3.0\r\n")
		out.WriteString("FN:")
		out.WriteString(escapeVCardValue(c.FullName))
		out.WriteString("\r\n")
		out.WriteString("TEL;TYPE=CELL:")
		out.WriteString(escapeVCardValue(c.Phone))
		out.WriteString("\r\n")
		if strings.TrimSpace(c.Email) != "" {
			out.WriteString("EMAIL:")
			out.WriteString(escapeVCardValue(c.Email))
			out.WriteString("\r\n")
		}
		if strings.TrimSpace(c.Org) != "" {
			out.WriteString("ORG:")
			out.WriteString(escapeVCardValue(c.Org))
			out.WriteString("\r\n")
		}
		if strings.TrimSpace(c.Note) != "" {
			out.WriteString("NOTE:")
			out.WriteString(escapeVCardValue(c.Note))
			out.WriteString("\r\n")
		}
		out.WriteString("END:VCARD\r\n")
	}
	return ExportVCFResponse{Status: "ok", Content: out.String(), Count: len(contacts)}, nil
}
