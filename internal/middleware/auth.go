package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/EOEboh/ziga-kit/internal/handlers/respond"
)

// contextKey is an unexported type for context keys in this package.
// Using a typed key prevents collisions with keys from other packages.
type contextKey string

const claimsKey contextKey = "claims"

// Authenticate is a Chi-compatible middleware that validates the Authorization
// header and injects the JWT claims into the request context.
//
// Routes behind this middleware can retrieve the caller's identity with
// ClaimsFromContext(r.Context()).
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respond.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				respond.Error(w, http.StatusUnauthorized, "authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := ParseToken(parts[1], jwtSecret)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Inject claims into context so handlers don't need to re-parse
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves the authenticated user's claims from the context.
// Returns nil if the context has no claims (i.e. on a public route).
// Always call this after the Authenticate middleware — never on public routes.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}
