package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"OdorikCentral/internal/core"
)

type Server struct {
	backend *core.Backend
	server  *http.Server
	baseURL string
	token   string
}

func New(backend *core.Backend) *Server {
	return &Server{backend: backend}
}

func (s *Server) Start() error {
	if s.server != nil {
		return nil
	}
	apiCfg, err := s.backend.LoadHTTPAPIConfig()
	if err != nil {
		return fmt.Errorf("http api: %w", err)
	}
	if !apiCfg.Enabled {
		return nil
	}

	token := apiCfg.Token
	generated := false
	if token == "" {
		token, err = generateToken()
		if err != nil {
			return fmt.Errorf("http api: generate token: %w", err)
		}
		generated = true
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(apiCfg.Port))
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.buildHandler(token),
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.server = srv
	s.baseURL = "http://" + addr
	s.token = token

	log.Printf("HTTP API enabled on %s", s.baseURL)
	if generated {
		log.Printf("HTTP API token (generated for this run): %s", token)
	} else {
		log.Printf("HTTP API token loaded from config/env")
	}

	go func() {
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Printf("HTTP API server stopped with error: %v", serveErr)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	srv := s.server
	s.server = nil

	shutdownCtx := ctx
	if shutdownCtx == nil {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	if _, hasDeadline := shutdownCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
	}
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http api shutdown: %w", err)
	}
	return nil
}

func (s *Server) buildHandler(token string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleJSON(func(_ *http.Request) (any, error) {
		return map[string]any{
			"status":      "ok",
			"service":     "odobox-http-api",
			"config_path": s.backend.ResolveConfigPath(),
			"started_at":  time.Now().Format(time.RFC3339),
		}, nil
	}))

	mux.HandleFunc("POST /api/voicemails/list", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.ListVoicemailsRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.ListVoicemails(req)
	}))
	mux.HandleFunc("POST /api/voicemails/sync", s.handleJSON(func(r *http.Request) (any, error) {
		var req struct {
			Days int `json:"days"`
		}
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.Sync(req.Days)
	}))
	mux.HandleFunc("POST /api/voicemails/checked", s.handleJSON(func(r *http.Request) (any, error) {
		var req struct {
			ID      int  `json:"id"`
			Checked bool `json:"checked"`
		}
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.SetVoicemailChecked(req.ID, req.Checked)
	}))
	mux.HandleFunc("GET /api/voicemails/{id}/audio", s.handleJSON(func(r *http.Request) (any, error) {
		id, err := parsePathInt(r, "id")
		if err != nil {
			return nil, err
		}
		dataURL, err := s.backend.GetVoicemailAudioDataURL(id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "data_url": dataURL}, nil
	}))

	mux.HandleFunc("POST /api/sms/list", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.ListSMSMessagesRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.ListSMSMessages(req)
	}))
	mux.HandleFunc("POST /api/sms/checked", s.handleJSON(func(r *http.Request) (any, error) {
		var req struct {
			ID      int  `json:"id"`
			Checked bool `json:"checked"`
		}
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.SetSMSMessageChecked(req.ID, req.Checked)
	}))
	mux.HandleFunc("POST /api/sms/send", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.SendSMSRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.SendSMS(req)
	}))

	mux.HandleFunc("GET /api/settings", s.handleJSON(func(_ *http.Request) (any, error) {
		return s.backend.GetSettings()
	}))
	mux.HandleFunc("PATCH /api/settings", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.PatchSettingsRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.PatchSettings(req)
	}))

	mux.HandleFunc("GET /api/odorik/balance", s.handleJSON(func(_ *http.Request) (any, error) {
		return s.backend.GetOdorikBalance()
	}))

	mux.HandleFunc("GET /api/contacts", s.handleJSON(func(_ *http.Request) (any, error) {
		return s.backend.ListContacts()
	}))
	mux.HandleFunc("POST /api/contacts", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.CreateContactRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.CreateContact(req)
	}))
	mux.HandleFunc("PUT /api/contacts/{id}", s.handleJSON(func(r *http.Request) (any, error) {
		id, err := parsePathInt(r, "id")
		if err != nil {
			return nil, err
		}
		var req core.UpdateContactRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		req.ID = id
		return s.backend.UpdateContact(req)
	}))
	mux.HandleFunc("DELETE /api/contacts/{id}", s.handleJSON(func(r *http.Request) (any, error) {
		id, err := parsePathInt(r, "id")
		if err != nil {
			return nil, err
		}
		return s.backend.DeleteContact(id)
	}))

	mux.HandleFunc("GET /api/sms/templates", s.handleJSON(func(_ *http.Request) (any, error) {
		return s.backend.ListSMSTemplates()
	}))
	mux.HandleFunc("POST /api/sms/templates", s.handleJSON(func(r *http.Request) (any, error) {
		var req core.CreateSMSTemplateRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		return s.backend.CreateSMSTemplate(req)
	}))
	mux.HandleFunc("PUT /api/sms/templates/{id}", s.handleJSON(func(r *http.Request) (any, error) {
		id, err := parsePathInt(r, "id")
		if err != nil {
			return nil, err
		}
		var req core.UpdateSMSTemplateRequest
		if err := decodeJSONBody(r, &req); err != nil {
			return nil, err
		}
		req.ID = id
		return s.backend.UpdateSMSTemplate(req)
	}))
	mux.HandleFunc("DELETE /api/sms/templates/{id}", s.handleJSON(func(r *http.Request) (any, error) {
		id, err := parsePathInt(r, "id")
		if err != nil {
			return nil, err
		}
		return s.backend.DeleteSMSTemplate(id)
	}))

	mux.HandleFunc("GET /api/debug/imap", s.handleJSON(func(r *http.Request) (any, error) {
		days := parseQueryInt(r, "days", 7)
		limit := parseQueryInt(r, "limit", 40)
		return s.backend.DebugIMAP(days, limit)
	}))
	mux.HandleFunc("GET /api/debug/imap/message", s.handleJSON(func(r *http.Request) (any, error) {
		seq := parseQueryInt(r, "seq", 0)
		if seq <= 0 {
			return nil, fmt.Errorf("query param seq must be a positive integer")
		}
		return s.backend.DebugIMAPMessage(uint32(seq))
	}))

	return withCORS(withTokenAuth(token, mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withTokenAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		given := strings.TrimSpace(r.Header.Get("X-API-Token"))
		if given == "" {
			const bearerPrefix = "Bearer "
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(auth, bearerPrefix) {
				given = strings.TrimSpace(strings.TrimPrefix(auth, bearerPrefix))
			}
		}
		if given == "" || given != token {
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid API token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleJSON(fn func(*http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		if err != nil {
			writeClassifiedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func decodeJSONBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("invalid JSON body: multiple JSON values are not allowed")
	}
	return nil
}

func parsePathInt(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("missing path parameter: %s", key)
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("path parameter %s must be a positive integer", key)
	}
	return id, nil
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func writeClassifiedError(w http.ResponseWriter, err error) {
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	status := http.StatusInternalServerError
	if errors.Is(err, sql.ErrNoRows) || strings.Contains(lower, "not found") {
		status = http.StatusNotFound
	} else if strings.Contains(lower, "must") || strings.Contains(lower, "invalid") || strings.Contains(lower, "missing") || strings.Contains(lower, "required") {
		status = http.StatusBadRequest
	}
	writeJSONError(w, status, msg)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"status": "error",
		"error":  message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("http api: failed to encode response: %v", err)
	}
}

func generateToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
