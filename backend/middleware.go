package main

import (
	"context"
	"net/http"
)

// contextKey - typo for context keys
type contextKey string

const userIDKey contextKey = "userID"

// authMiddleware  - check access-token from cookies
// if token valid, userID go to context and call next handler
// if token not valid - return 401 and do not call next handler
func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//get the cookie with access-token
		cookie, err := r.Cookie("edu_session")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unautorized", "Valid sessin required")
			return
		}

		//check the token in sessions db
		userID, ok := a.Sessions.Lookup(cookie.Value)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Valid session required")
			return
		}

		//put userId to context
		//context.WithValue create a new context with key userIDKey and value userID
		//r.WithContext return request-copy with new context
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		r = r.WithContext(ctx)

		//call next handler
		next.ServeHTTP(w, r)

	})
}

// userIDFromContext - func-helper for getting userID from context
// return empty string if middleware has not worked
func userIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)
	return userID
}
