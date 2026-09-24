package auth

import (
	"context"
	"time"
)

// TokenIssuer signs and returns a new JWT for the given claims.
type TokenIssuer interface {
	IssueToken(ctx context.Context, claims Claims) (string, error)
}

// TokenValidator parses and verifies a raw JWT string.
// Returns the embedded Claims on success.
type TokenValidator interface {
	ValidateToken(ctx context.Context, raw string) (Claims, error)
}

// Service combines both token capabilities.
// Depend on the narrower interfaces (TokenIssuer, TokenValidator) everywhere except main.go.
type Service interface {
	TokenIssuer
	TokenValidator
}

// Claims holds the payload embedded in every JWT issued by this service.
type Claims struct {
	UserID    string    `json:"id"`
	FullName  string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"exp"`
}
