package core

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func validateTemplateFields(name, body string) (string, string, error) {
	cleanName := strings.TrimSpace(name)
	cleanBody := strings.TrimSpace(body)
	if cleanName == "" {
		return "", "", fmt.Errorf("template name is required")
	}
	if cleanBody == "" {
		return "", "", fmt.Errorf("template body is required")
	}
	if utf8.RuneCountInString(cleanName) > 120 {
		return "", "", fmt.Errorf("template name is too long (max 120 chars)")
	}
	if utf8.RuneCountInString(cleanBody) > 2000 {
		return "", "", fmt.Errorf("template body is too long (max 2000 chars)")
	}
	return cleanName, cleanBody, nil
}

func (b *Backend) listSMSTemplates() ([]SMSTemplate, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, body, created_at, updated_at FROM sms_templates ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SMSTemplate{}
	for rows.Next() {
		var t SMSTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (b *Backend) createSMSTemplate(req CreateSMSTemplateRequest) (SMSTemplate, error) {
	name, body, err := validateTemplateFields(req.Name, req.Body)
	if err != nil {
		return SMSTemplate{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return SMSTemplate{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return SMSTemplate{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(`INSERT INTO sms_templates(name, body, created_at, updated_at) VALUES(?, ?, ?, ?)`, name, body, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return SMSTemplate{}, fmt.Errorf("template name already exists")
		}
		return SMSTemplate{}, err
	}
	id64, _ := res.LastInsertId()
	return SMSTemplate{ID: int(id64), Name: name, Body: body, CreatedAt: now, UpdatedAt: now}, nil
}

func (b *Backend) updateSMSTemplate(req UpdateSMSTemplateRequest) (SMSTemplate, error) {
	if req.ID < 1 {
		return SMSTemplate{}, fmt.Errorf("template id must be positive")
	}
	name, body, err := validateTemplateFields(req.Name, req.Body)
	if err != nil {
		return SMSTemplate{}, err
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return SMSTemplate{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return SMSTemplate{}, err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := db.Exec(`UPDATE sms_templates SET name = ?, body = ?, updated_at = ? WHERE id = ?`, name, body, now, req.ID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return SMSTemplate{}, fmt.Errorf("template name already exists")
		}
		return SMSTemplate{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return SMSTemplate{}, fmt.Errorf("template not found")
	}

	var createdAt string
	if err := db.QueryRow(`SELECT created_at FROM sms_templates WHERE id = ?`, req.ID).Scan(&createdAt); err != nil {
		return SMSTemplate{}, err
	}
	return SMSTemplate{ID: req.ID, Name: name, Body: body, CreatedAt: createdAt, UpdatedAt: now}, nil
}

func (b *Backend) deleteSMSTemplate(id int) (DeleteSMSTemplateResponse, error) {
	if id < 1 {
		return DeleteSMSTemplateResponse{}, fmt.Errorf("template id must be positive")
	}
	cfg, err := b.loadConfig()
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	db, err := openDB(b.resolveDBPath(cfg))
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM sms_templates WHERE id = ?`, id)
	if err != nil {
		return DeleteSMSTemplateResponse{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return DeleteSMSTemplateResponse{}, fmt.Errorf("template not found")
	}
	return DeleteSMSTemplateResponse{Status: "ok", ID: id}, nil
}
