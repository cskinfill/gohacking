package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type AuthenticationMiddleware struct {
	db *sql.DB
}

func NewAuthMiddleware(db *sql.DB) (*AuthenticationMiddleware, error) {
	return &AuthenticationMiddleware{
		db: db,
	}, nil
}

func (amw *AuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		_, span := tracer.Start(r.Context(), "Service")
		defer span.End()

		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		if path == "/metrics" {
			next.ServeHTTP(w, r)
		}

		account, token, _ := r.BasicAuth()
		var found bool
		if err := amw.db.QueryRowContext(r.Context(), "SELECT (count(1)==1) FROM auth WHERE account = ? AND token = ?", account, token).Scan(&found); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}
		}
		if found {
			log.Printf("Authenticated account %s\n", account)
			// Pass down the request to the next middleware (or final handler)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}
