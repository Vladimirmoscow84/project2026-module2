// Supplied session primitives; learners write login/me/logout and access checks.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type session struct {
	UserID  string
	Expires time.Time
}
type Sessions struct {
	mu     sync.Mutex
	values map[string]session
}

func NewSessions() *Sessions { return &Sessions{values: make(map[string]session)} }
func (s *Sessions) Create(userID string, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, entry := range s.values {
		if !entry.Expires.After(now) {
			delete(s.values, key)
		}
	}
	s.values[token] = session{UserID: userID, Expires: now.Add(ttl)}
	return token, nil
}
func (s *Sessions) Lookup(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[token]
	if !ok || !value.Expires.After(time.Now()) {
		delete(s.values, token)
		return "", false
	}
	return value.UserID, true
}
func (s *Sessions) Delete(token string) { s.mu.Lock(); defer s.mu.Unlock(); delete(s.values, token) }
func sessionCookie(value string, clear bool) *http.Cookie {
	age := 3600
	if clear {
		age = -1
	}
	return &http.Cookie{Name: "edu_session", Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: false, MaxAge: age}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "serialization failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
