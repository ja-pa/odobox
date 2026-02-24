package core

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func normalizeRecipient(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	return s
}

func normalizePhoneLoose(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	re := regexp.MustCompile(`\D+`)
	s = re.ReplaceAllString(s, "")
	if strings.HasPrefix(s, "00") {
		s = strings.TrimPrefix(s, "00")
	}
	// Canonicalize Czech numbers so +420/00420 and local 9-digit form match.
	if strings.HasPrefix(s, "420") && len(s) == 12 {
		s = s[3:]
	}
	return s
}

func validateRecipient(raw string) (string, error) {
	recipient := normalizeRecipient(raw)
	if recipient == "" {
		return "", fmt.Errorf("recipient is required")
	}
	if strings.HasPrefix(recipient, "+") {
		recipient = "00" + strings.TrimPrefix(recipient, "+")
	}
	// Allow Czech local and country-prefixed forms entered without "00".
	if reLocalCZ := regexp.MustCompile(`^\d{9}$`); reLocalCZ.MatchString(recipient) {
		recipient = "00420" + recipient
	} else if reCZNoPrefix := regexp.MustCompile(`^420\d{9}$`); reCZNoPrefix.MatchString(recipient) {
		recipient = "00" + recipient
	}
	valid := regexp.MustCompile(`^00\d{6,15}$`)
	if !valid.MatchString(recipient) {
		return "", fmt.Errorf("recipient must be in international format starting with 00 (e.g. 00420123456789)")
	}
	return recipient, nil
}

func gsmCharUnits(message string) (int, bool) {
	// GSM-7 basic + extension table (extension chars consume 2 units).
	const gsmBasic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	const gsmExt = "^{}\\[~]|€"
	basic := map[rune]bool{}
	ext := map[rune]bool{}
	for _, r := range gsmBasic {
		basic[r] = true
	}
	for _, r := range gsmExt {
		ext[r] = true
	}
	units := 0
	for _, r := range message {
		if basic[r] {
			units++
			continue
		}
		if ext[r] {
			units += 2
			continue
		}
		return 0, false
	}
	return units, true
}

func singleSMSSegmentInfo(message string) (encoding string, used int, limit int) {
	if units, ok := gsmCharUnits(message); ok {
		return "GSM-7", units, 160
	}
	return "UCS-2", utf8.RuneCountInString(message), 70
}

func composeSMSMessage(identityText, body string) string {
	identity := strings.TrimSpace(identityText)
	message := strings.TrimSpace(body)
	if identity == "" {
		return message
	}
	if message == "" {
		return identity
	}
	return fmt.Sprintf("%s: %s", identity, message)
}

func (b *Backend) logSMS(
	db *sql.DB,
	recipient string,
	sender string,
	message string,
	encoding string,
	charsUsed int,
	maxChars int,
	providerResponse string,
	success bool,
	errMsg string,
) {
	_, _ = db.Exec(
		`INSERT INTO sms_outbox
		(created_at, recipient, sender_id, message_text, encoding, chars_used, max_chars, provider_response, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		recipient,
		sender,
		message,
		encoding,
		charsUsed,
		maxChars,
		providerResponse,
		boolToInt(success),
		errMsg,
	)
}

func (b *Backend) sendSMS(req SendSMSRequest) (SendSMSResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return SendSMSResponse{}, err
	}
	smsCfg := b.loadSMSConfig(cfg)
	if smsCfg.User == "" || smsCfg.Password == "" {
		return SendSMSResponse{}, fmt.Errorf("missing Odorik SMS credentials in settings (set odorik.user and odorik.password; account_id/api_pin also supported)")
	}

	recipient, err := validateRecipient(req.Recipient)
	if err != nil {
		return SendSMSResponse{}, err
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return SendSMSResponse{}, fmt.Errorf("message is required")
	}
	message = composeSMSMessage(cfg.get("app", "sms_identity_text", ""), message)

	encoding, used, limit := singleSMSSegmentInfo(message)
	if used > limit {
		return SendSMSResponse{}, fmt.Errorf(
			"message is too long for one SMS (%s %d/%d). Shorten text so it fits a single segment",
			encoding,
			used,
			limit,
		)
	}

	sender := strings.TrimSpace(req.Sender)
	if sender == "" {
		sender = smsCfg.DefaultID
	}

	form := url.Values{}
	form.Set("user", smsCfg.User)
	form.Set("password", smsCfg.Password)
	form.Set("recipient", recipient)
	form.Set("message", message)
	if sender != "" {
		form.Set("sender", sender)
	}
	form.Set("user_agent", "OdoBox-Wails")

	httpClient := &http.Client{Timeout: 35 * time.Second}
	resp, err := postFormWithRetry(httpClient, "https://www.odorik.cz/api/v1/sms", form, 3)
	if err != nil {
		if isTimeoutError(err) {
			return SendSMSResponse{}, fmt.Errorf("sms request failed: network timeout while connecting to Odorik (try again)")
		}
		return SendSMSResponse{}, fmt.Errorf("sms request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	provider := strings.TrimSpace(string(body))
	if provider == "" {
		provider = "empty_response"
	}

	db, dbErr := openDB(b.resolveDBPath(cfg))
	if dbErr == nil {
		defer db.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if dbErr == nil {
			b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, false, resp.Status)
		}
		return SendSMSResponse{}, fmt.Errorf("sms provider rejected request: %s (%s)", resp.Status, provider)
	}
	if strings.HasPrefix(strings.ToLower(provider), "error") || strings.HasPrefix(strings.ToLower(provider), "bad_") {
		if dbErr == nil {
			b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, false, provider)
		}
		if strings.Contains(strings.ToLower(provider), "authentication_failed") {
			return SendSMSResponse{}, fmt.Errorf(
				"sms provider error: %s. Verify Odorik Account ID and API password at https://www.odorik.cz/ucet/nastaveni_uctu.html?ucet_podmenu=api_heslo",
				provider,
			)
		}
		return SendSMSResponse{}, fmt.Errorf("sms provider error: %s", provider)
	}

	if dbErr == nil {
		b.logSMS(db, recipient, sender, message, encoding, used, limit, provider, true, "")
	}

	return SendSMSResponse{
		Status:           "ok",
		Recipient:        recipient,
		Sender:           sender,
		Encoding:         encoding,
		CharsUsed:        used,
		MaxSingleChars:   limit,
		ProviderResponse: provider,
		SentAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func extractBalanceValue(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}
	re := regexp.MustCompile(`[-+]?\d+(?:[.,]\d+)?`)
	value := re.FindString(clean)
	if value == "" {
		return clean
	}
	return strings.ReplaceAll(value, ",", ".")
}

func (b *Backend) getOdorikBalance() (OdorikBalanceResponse, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return OdorikBalanceResponse{}, err
	}
	smsCfg := b.loadSMSConfig(cfg)
	if smsCfg.User == "" || smsCfg.Password == "" {
		return OdorikBalanceResponse{}, fmt.Errorf("missing Odorik credentials in settings")
	}

	form := url.Values{}
	form.Set("user", smsCfg.User)
	form.Set("password", smsCfg.Password)
	form.Set("user_agent", "OdoBox-Wails")

	httpClient := &http.Client{Timeout: 20 * time.Second}
	endpoint := "https://www.odorik.cz/api/v1/balance?" + form.Encode()
	var resp *http.Response
	var reqErr error
	for i := 0; i < 3; i++ {
		req, buildErr := http.NewRequest(http.MethodGet, endpoint, nil)
		if buildErr != nil {
			return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: %w", buildErr)
		}
		resp, reqErr = httpClient.Do(req)
		if reqErr == nil {
			break
		}
		if !isTimeoutError(reqErr) || i == 2 {
			break
		}
		time.Sleep(time.Duration(i+1) * 700 * time.Millisecond)
	}
	if reqErr != nil {
		if isTimeoutError(reqErr) {
			return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: network timeout while connecting to Odorik")
		}
		return OdorikBalanceResponse{}, fmt.Errorf("balance request failed: %w", reqErr)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	provider := strings.TrimSpace(string(body))
	if provider == "" {
		provider = "empty_response"
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return OdorikBalanceResponse{}, fmt.Errorf("balance provider rejected request: %s (%s)", resp.Status, provider)
	}
	lowerProvider := strings.ToLower(provider)
	if strings.HasPrefix(lowerProvider, "error") || strings.HasPrefix(lowerProvider, "bad_") {
		if strings.Contains(lowerProvider, "authentication_failed") {
			return OdorikBalanceResponse{}, fmt.Errorf(
				"balance provider error: %s. Verify Odorik Account ID and API password at https://www.odorik.cz/ucet/nastaveni_uctu.html?ucet_podmenu=api_heslo",
				provider,
			)
		}
		return OdorikBalanceResponse{}, fmt.Errorf("balance provider error: %s", provider)
	}

	currency := "Kc"
	if strings.Contains(lowerProvider, "eur") {
		currency = "EUR"
	}
	return OdorikBalanceResponse{
		Status:           "ok",
		Balance:          extractBalanceValue(provider),
		Currency:         currency,
		ProviderResponse: provider,
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func postFormWithRetry(client *http.Client, endpoint string, form url.Values, attempts int) (*http.Response, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		resp, err := client.PostForm(endpoint, form)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isTimeoutError(err) || i == attempts-1 {
			return nil, err
		}
		time.Sleep(time.Duration(i+1) * 700 * time.Millisecond)
	}
	return nil, lastErr
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "tls handshake timeout") || strings.Contains(lower, "timeout")
}
