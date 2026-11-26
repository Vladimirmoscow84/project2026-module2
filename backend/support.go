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

// --------------------------------------------------------------------------------------------------
// add new code from STUDENT
type RefreshSessions struct {
	mu     sync.Mutex
	values map[string]session
}

func NewRefreshSessions() *RefreshSessions {
	return &RefreshSessions{values: make(map[string]session)}
}
func (s *RefreshSessions) Create(userID string, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, value := range s.values {
		if !value.Expires.After(now) {
			delete(s.values, key)
		}
	}
	s.values[token] = session{UserID: userID, Expires: now.Add(ttl)}
	return token, nil
}
func (s *RefreshSessions) Lookup(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[token]
	if !ok || !value.Expires.After(time.Now()) {
		delete(s.values, token)
		return "", false
	}
	return value.UserID, true

}

func (s *RefreshSessions) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, token)
}

// refreshCookie  - cocie for refresh token lives 30 days
func refreshCookie(value string, clear bool) *http.Cookie {
	//30 dys from secconds
	age := 30 * 24 * 3600
	if clear {
		age = -1
	}
	return &http.Cookie{Name: "edu_refresh", Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: false, MaxAge: age}
}

// Ratelimiter - simple login attempt rate limiter
// save last time attempt for each email
type RateLimiter struct {
	mu    sync.Mutex
	last  map[string]time.Time
	limit time.Duration
}

func NewRateLimiter(limit time.Duration) *RateLimiter {
	return &RateLimiter{
		last:  make(map[string]time.Time),
		limit: limit,
	}
}

// allow - checks whether an attempt can be made
func (r *RateLimiter) Allow(email string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	last, ok := r.last[email]
	if ok && now.Sub(last) < r.limit {
		return false
	}
	r.last[email] = now

	return true
}
