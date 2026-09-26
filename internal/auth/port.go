package auth

import (
	"context"
	"time"
)

// TokenIssuer signs and returns a new token pair for the given claims.
type TokenIssuer interface {
	IssueTokenPair(ctx context.Context, claims Claims) (TokenPair, error)
}

// TokenValidator parses and verifies a raw access token JWT string.
type TokenValidator interface {
	ValidateToken(ctx context.Context, raw string) (Claims, error)
}

// TokenRefresher rotates a refresh token and returns a new token pair.
type TokenRefresher interface {
	RefreshTokens(ctx context.Context, refreshToken string) (TokenPair, error)
}

// SessionTerminator revokes all active tokens for a user.
type SessionTerminator interface {
	Logout(ctx context.Context, userID string) error
}

// Service combines all auth capabilities.
// Depend on the narrower interfaces everywhere except main.go.
type Service interface {
	TokenIssuer
	TokenValidator
	TokenRefresher
	SessionTerminator
}

// Claims holds the payload embedded in every access token issued by this service.
type Claims struct {
	UserID    string    `json:"id"`
	FullName  string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"exp"`
}

// TokenPair bundles an access token and its paired refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	// AccessTokenExpiresAt is used to set the cookie Max-Age on the HTTP layer.
	AccessTokenExpiresAt time.Time
}
