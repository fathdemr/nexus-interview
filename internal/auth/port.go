package auth

import (
	"context"
	"errors"
	"time"

	"github.com/fathdemr/nexus-interview/pkg/model"
)

var (
	ErrNotFound        = errors.New("user not found")
	ErrAlreadyExists   = errors.New("user already exists")
	ErrInvalidRole     = errors.New("invalid role: must be admin or candidate")
	ErrInvalidPassword = errors.New("password does not match")
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

// Registrar handles new user sign-ups.
type Registrar interface {
	Register(ctx context.Context, req RegisterRequest) (TokenPair, error)
}

// Authenticator handles credential-based login.
type Authenticator interface {
	Login(ctx context.Context, req LoginRequest) (TokenPair, error)
}

// Service combines all auth capabilities.
// Depend on the narrower interfaces everywhere except main.go.
type Service interface {
	TokenIssuer
	TokenValidator
	TokenRefresher
	SessionTerminator
	Registrar
	Authenticator
}

// Repository is the data access contract owned by this module.
type Repository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	IsExists(ctx context.Context, email string) bool
	Create(ctx context.Context, user model.User) (model.User, error)
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

// RegisterRequest is the validated payload for the POST /auth/register endpoint.
type RegisterRequest struct {
	FirstName string         `json:"first_name" binding:"required,min=2,max=100"`
	LastName  string         `json:"last_name"  binding:"required,min=2,max=100"`
	Email     string         `json:"email"      binding:"required,email"`
	Password  string         `json:"password"   binding:"required,min=8"`
	Role      model.UserRole `json:"role"       binding:"required"`
}

// LoginRequest is the validated payload for the POST /auth/login endpoint.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
