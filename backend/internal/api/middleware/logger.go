package middleware

import (
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return chimw.Logger(next)
}

func Recoverer(next http.Handler) http.Handler {
	return chimw.Recoverer(next)
}
