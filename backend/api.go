package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// routes - binds the path to the handlers
func routes(mux *http.ServeMux, app *App) {
	mux.HandleFunc("POST /login", app.loginHandler)
	mux.HandleFunc("GET /me", app.meHandler)
	mux.HandleFunc("POST /register", app.registerHandler)
	mux.HandleFunc("POST /refresh", app.refreshHandler)
	//
	//
	//
	//
	//

}

// loginhandler  - handler for POST method, validates JSON file with email and password
// and returns a token
func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_requesr", "Email and password are required")
		return
	}

	//check rate limit: no more one attempt on 5 seconds
	if !a.Limiter.Allow(input.Email) {
		writeError(w, http.StatusTooManyRequests, "too_many_requests", "Too many login attempts, try again later")
		return
	}

	var userID, hash string

	//do request to DB and write to userID and hash
	row := a.DB.QueryRowContext(r.Context(),
		`SELECT user_id::text, password_hash FROM users WHERE email=$1`, input.Email)
	err = row.Scan(&userID, &hash)

	if errors.Is(err, sql.ErrNoRows) || (err == nil && !CheckPassword(hash, input.Password)) {
		writeError(w, http.StatusUnauthorized, "invalid_login", "Invalid login")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "login_failed", "login not completed")
		return
	}

	//delete old seesion it is shall be
	//delete old tokien from cookie memory
	//it is a protection against accumulation of old sessions
	previos, err := r.Cookie("edu_session")
	if err == nil && previos.Value != "" {
		a.Sessions.Delete(previos.Value)
	}

	// create access-token (lives 15 minutes).
	accessToken, err := a.Sessions.Create(userID, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "Access token creation failed")
		return
	}

	// create refresh-token (lives 30 days).
	refreshToken, err := a.RefreshSessions.Create(userID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "Refresh token creation failed")
		return
	}

	// return two tikiens in cookie.
	http.SetCookie(w, sessionCookie(accessToken, false))
	http.SetCookie(w, refreshCookie(refreshToken, false))

	//response to client 200 ok with simple JSON file
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// meHandler - handler for GET /me
// return info about the current user (email)
// bot only if request has valid token in cookies
func (a *App) meHandler(w http.ResponseWriter, r *http.Request) {

	//try get cookies with any name axample: "edu_session""
	cookie, err := r.Cookie("edu_session")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Valid. session. required")
		return
	}

	//check the token in the sessions storage
	userID, ok := a.Sessions.Lookup(cookie.Value)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Valid. session. required")
		return
	}

	//go to db to get an users email
	var email string
	row := a.DB.QueryRowContext(r.Context(),
		`SELECT email FROM users WHERE user_id=$1`, userID)

	err = row.Scan(&email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "me_failed", "Could not fetch user")
		return
	}

	//all good return 200 ok and JSON. with email
	writeJSON(w, http.StatusOK, map[string]string{"email": email})
}

// registerhandler - handler for POST /register
// create new user: check email, hash password and save in DB
func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	//read request body with JSON: {"email": "...", "password": "..."}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	//decode JSON, if body not JSON or wrong syntaxis - 400
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	//worcking with email and check ifields not empty
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email and password are required")
		return
	}

	//check len password (12byte<=input.Password<=72byte)
	//bcrypt has the limmits that cut password longer than 72 bytes
	if len(input.Password) < 12 || len(input.Password) > 72 {
		writeError(w, http.StatusBadRequest, "invalid_password", "password must be 12-72 bytes")
		return
	}

	//hash the password
	//bcrypt.GenerateFromPassword - return hash ([]byte) and error
	//bcrypt.DefaultCost - standart difficult(10)
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hashing_failed", "Could not hash password")
		return
	}

	//Try to save user in DB
	//if email is using, PG returned unique error (duplicate key)
	// we are using ON CONFLICT DO NOTHING - replay not crashed program and only not insert

	result, err := a.DB.ExecContext(r.Context(),
		`INSERT INTO users (email, password_hash) VALUES($1, $2) ON CONFLICT (email) DO NOTHING`, input.Email, string(hash))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "register_failed", "Could not register user")
		return
	}

	//check inser string
	//if RowsAffected()==0 - email shall be in DB (ON CONFLICT has worked)
	rows, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "register_failed", "Could not confirm registration")
		return
	}
	if rows == 0 {
		writeError(w, http.StatusConflict, "email_taken", "Email is already registered")
		return
	}

	//all right return 201
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// refeshHandler -POST /refresh
// change refresh-token to new twins (access + refresh)
func (a *App) refreshHandler(w http.ResponseWriter, r *http.Request) {
	//get refresh-token from cookie
	cookie, err := r.Cookie("edu_refresh")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Valid refresh token required")
		return
	}

	//check refresh-token in storage
	userID, ok := a.RefreshSessions.Lookup(cookie.Value)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Valid refresh token required")
		return

	}

	//delete old refresh-token (rotation)
	a.RefreshSessions.Delete(cookie.Value)

	//create new twins token
	accessToken, err := a.Sessions.Create(userID, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "Access token creation failed")
		return
	}
	refreshToken, err := a.RefreshSessions.Create(userID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "Refresh token creation failed")
		return
	}

	// return two tokens to cookie
	http.SetCookie(w, sessionCookie(accessToken, false))
	http.SetCookie(w, refreshCookie(refreshToken, false))

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}
