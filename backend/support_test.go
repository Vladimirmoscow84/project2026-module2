package main

import (
    "testing"
    "time"
)

func TestSessionLifecycle(t *testing.T) {
    s := NewSessions()
    token, err := s.Create("1", time.Hour)
    if err != nil { t.Fatal(err) }
    if id, ok := s.Lookup(token); !ok || id != "1" { t.Fatal("session missing") }
    s.Delete(token)
    if _, ok := s.Lookup(token); ok { t.Fatal("revoked session accepted") }
    expired, err := s.Create("1", -time.Second)
    if err != nil { t.Fatal(err) }
    if _, ok := s.Lookup(expired); ok { t.Fatal("expired session accepted") }
}
