package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type matter struct {
	Phone          string `json:"phone"`
	MatterID       string `json:"matter_id"`
	SignedDocument string `json:"signed_document"`
	Deadline       string `json:"deadline"`
}

type confirmation struct {
	MatterID string `json:"matter_id"`
	Code     string `json:"code"`
}

type intake struct {
	mu      sync.Mutex
	pending map[string]matter
	client  phoneClient
}

func validateMatter(m matter, now time.Time) error {
	if m.MatterID == "" || !strings.HasPrefix(m.Phone, "+") || len(m.Phone) < 8 || m.SignedDocument == "" {
		return errors.New("matter_id, E.164 phone and signed_document are required")
	}
	deadline, err := time.Parse("2006-01-02", m.Deadline)
	if err != nil || deadline.Before(now.UTC().Truncate(24*time.Hour)) {
		return errors.New("deadline must be today or a future YYYY-MM-DD date")
	}
	return nil
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func reject(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var api *apiError
	if errors.As(err, &api) {
		status = http.StatusBadRequest
	}
	reply(w, status, map[string]string{"error": err.Error()})
}

func (s *intake) start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var m matter
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&m) != nil || validateMatter(m, time.Now()) != nil {
		reply(w, http.StatusBadRequest, map[string]string{"error": "invalid matter intake"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pending[m.MatterID]; exists {
		reply(w, http.StatusConflict, map[string]string{"error": "matter_id already pending"})
		return
	}
	if err := s.client.post(r.Context(), "/v1/auth/phone/send_code", map[string]any{"phone": m.Phone, "purpose": "login"}); err != nil {
		reject(w, err)
		return
	}
	s.pending[m.MatterID] = m
	reply(w, http.StatusAccepted, map[string]string{"matter_id": m.MatterID, "status": "awaiting_phone_code"})
}

func (s *intake) confirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var input confirmation
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&input) != nil || input.Code == "" {
		reply(w, http.StatusBadRequest, map[string]string{"error": "matter_id and code are required"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, found := s.pending[input.MatterID]
	if !found {
		reply(w, http.StatusNotFound, map[string]string{"error": "matter not pending"})
		return
	}
	if err := s.client.post(r.Context(), "/v1/auth/phone/verify", map[string]any{"phone": m.Phone, "code": input.Code, "login": true}); err != nil {
		reject(w, err)
		return
	}
	// The same credential sends the deadline follow-up directly after identity verification.
	if err := s.client.post(r.Context(), "/v1/sms/otp", map[string]any{"to": m.Phone}); err != nil {
		reject(w, err)
		return
	}
	delete(s.pending, input.MatterID)
	reply(w, http.StatusOK, map[string]string{"matter_id": m.MatterID, "signed_document": m.SignedDocument, "deadline": m.Deadline, "status": "phone_verified_followup_sent"})
}
