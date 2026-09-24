package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrTokenMalformed = errors.New("token is malformed")
	ErrTokenNotYetValid = errors.New("token is not yet valid")
	ErrTokenFuture    = errors.New("token issued in the future")
)

// clockSkewTolerance is the allowed drift between issuer and verifier clocks.
const clockSkewTolerance = 5 * time.Minute

type jwtClaims struct {
	UserID   string `json:"id"`
	FullName string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type service struct {
	privateKeyPEM   string
	publicKeyPEM    string
	expirationHours int

	privateKeyOnce sync.Once
	privateKey     *rsa.PrivateKey
	privateKeyErr  error

	publicKeyOnce sync.Once
	publicKey     *rsa.PublicKey
	publicKeyErr  error
}

// NewService constructs an auth Service. PEM keys are parsed lazily on first use
// and cached for the lifetime of the service. Parsing errors surface on first call.
func NewService(privateKeyPEM, publicKeyPEM string, expirationHours int) Service {
	return &service{
		privateKeyPEM:   privateKeyPEM,
		publicKeyPEM:    publicKeyPEM,
		expirationHours: expirationHours,
	}
}

func (s *service) loadPrivateKey() (*rsa.PrivateKey, error) {
	s.privateKeyOnce.Do(func() {
		key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(s.privateKeyPEM))
		if err != nil {
			s.privateKeyErr = fmt.Errorf("parse RSA private key: %w", err)
			return
		}
		s.privateKey = key
	})
	return s.privateKey, s.privateKeyErr
}

func (s *service) loadPublicKey() (*rsa.PublicKey, error) {
	s.publicKeyOnce.Do(func() {
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.publicKeyPEM))
		if err != nil {
			s.publicKeyErr = fmt.Errorf("parse RSA public key: %w", err)
			return
		}
		s.publicKey = key
	})
	return s.publicKey, s.publicKeyErr
}

// IssueToken signs a new RS256 JWT embedding the provided claims.
func (s *service) IssueToken(_ context.Context, claims Claims) (string, error) {
	privateKey, err := s.loadPrivateKey()
	if err != nil {
		return "", err
	}

	now := time.Now()
	c := jwtClaims{
		UserID:   claims.UserID,
		FullName: claims.FullName,
		Email:    claims.Email,
		Role:     claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expirationHours) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and fully verifies an RS256 JWT.
// Checks: signature, algorithm, exp, nbf, iat (with clock skew tolerance).
func (s *service) ValidateToken(_ context.Context, raw string) (Claims, error) {
	publicKey, err := s.loadPublicKey()
	if err != nil {
		return Claims{}, err
	}

	token, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return Claims{}, ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return Claims{}, ErrTokenNotYetValid
		case errors.Is(err, jwt.ErrTokenMalformed):
			return Claims{}, ErrTokenMalformed
		default:
			return Claims{}, ErrTokenInvalid
		}
	}

	parsed, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return Claims{}, ErrTokenInvalid
	}

	// iat check — reject tokens claiming to be issued in the future.
	// jwt library does not enforce this automatically.
	if parsed.IssuedAt != nil {
		if parsed.IssuedAt.Time.After(time.Now().Add(clockSkewTolerance)) {
			return Claims{}, ErrTokenFuture
		}
	}

	return Claims{
		UserID:    parsed.UserID,
		FullName:  parsed.FullName,
		Email:     parsed.Email,
		Role:      parsed.Role,
		ExpiresAt: parsed.ExpiresAt.Time,
	}, nil
}
