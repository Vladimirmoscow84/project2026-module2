// Supplied bootstrap only. No application handlers are implemented here.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type App struct {
	DB              *sql.DB
	Sessions        *Sessions
	RefreshSessions *RefreshSessions
}

func CheckPassword(hash string, value string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value)) == nil
}
func openDB() (*sql.DB, error) {
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("database configuration invalid")
	}
	config.Host = "db"
	config.Port = 5432
	config.User = os.Getenv("POSTGRES_USER")
	config.Password = os.Getenv("POSTGRES_PASSWORD")
	config.Database = os.Getenv("POSTGRES_DB")
	config.TLSConfig = nil
	config.Fallbacks = nil
	if config.User == "" || config.Password == "" || config.Database == "" {
		return nil, fmt.Errorf("required database configuration is missing")
	}
	db := stdlib.OpenDB(*config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database unavailable")
	}
	return db, nil
}
func seed(db *sql.DB) error {
	for _, email := range []string{"learner-a@example.invalid", "learner-b@example.invalid"} {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("interactive terminal required")
		}
		fmt.Printf("Set local training password for %s (12–72 bytes, hidden): ", email)
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("input unavailable")
		}
		if len(value) < 12 || len(value) > 72 {
			return fmt.Errorf("training password length invalid")
		}

		hash, err := bcrypt.GenerateFromPassword(value, bcrypt.DefaultCost)
		clear(value)
		if err != nil {
			return fmt.Errorf("hashing failed")
		}
		_, err = db.Exec(`INSERT INTO users(email, password_hash) VALUES($1,$2)
            ON CONFLICT(email) DO UPDATE SET password_hash=EXCLUDED.password_hash`, email, string(hash))
		if err != nil {
			return fmt.Errorf("seed failed")
		}
	}
	return nil
}
func main() {
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := seed(db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Training users prepared; no values printed.")
		return
	}
	app := &App{DB: db, Sessions: NewSessions(), RefreshSessions: NewRefreshSessions()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if app.DB.PingContext(ctx) != nil {
			writeError(w, 503, "not_ready", "Database unavailable")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	// STUDENT TASK: add six route registrations and implement handlers in new files.
	// A health-only bootstrap is NOT a complete API or a passing lesson artifact.
	routes(mux, app)
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second}
	log.Print("bootstrap listening; application API must be implemented by learner")
	log.Fatal(server.ListenAndServe())
}
