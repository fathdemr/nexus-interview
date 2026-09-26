package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fathdemr/nexus-interview/pkg/cache"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenInvalid     = errors.New("token is invalid")
	ErrTokenMalformed   = errors.New("token is malformed")
	ErrTokenNotYetValid = errors.New("token is not yet valid")
	ErrTokenFuture      = errors.New("token issued in the future")
	ErrRefreshNotFound  = errors.New("refresh token not found or already used")
)

const clockSkewTolerance = 5 * time.Minute

// refreshKeyPrefix defines the Redis key namespace for refresh tokens.
// Full key: "auth:refresh:<userID>:<tokenID>"
// Logout deletes all keys under "auth:refresh:<userID>:" in one sweep.
const refreshKeyPrefix = "auth:refresh"

type jwtClaims struct {
	UserID   string `json:"id"`
	FullName string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// refreshPayload is the value stored in Redis for each refresh token.
// Storing claims here avoids a second DB lookup during token rotation.
type refreshPayload struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type service struct {
	privateKeyPEM                string
	publicKeyPEM                 string
	accessTokenExpirationMinutes int
	refreshTokenExpirationDays   int
	cache                        *cache.Service

	privateKeyOnce sync.Once
	privateKey     *rsa.PrivateKey
	privateKeyErr  error

	publicKeyOnce sync.Once
	publicKey     *rsa.PublicKey
	publicKeyErr  error
}

// NewService constructs an auth Service. PEM keys are parsed lazily on first use.
func NewService(
	privateKeyPEM, publicKeyPEM string,
	accessTokenExpirationMinutes, refreshTokenExpirationDays int,
	cache *cache.Service,
) Service {
	return &service{
		privateKeyPEM:                privateKeyPEM,
		publicKeyPEM:                 publicKeyPEM,
		accessTokenExpirationMinutes: accessTokenExpirationMinutes,
		refreshTokenExpirationDays:   refreshTokenExpirationDays,
		cache:                        cache,
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

// IssueTokenPair creates a short-lived RS256 access token and a long-lived refresh token.
// The refresh token is stored in Redis as: "auth:refresh:<userID>:<tokenID>" → JSON(refreshPayload).
func (s *service) IssueTokenPair(_ context.Context, claims Claims) (TokenPair, error) {
	accessToken, expiresAt, err := s.signAccessToken(claims)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := s.storeRefreshToken(claims)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: expiresAt,
	}, nil
}

// RefreshTokens atomically rotates a refresh token and issues a fresh pair.
// GetDel is used so the old token is consumed in a single atomic operation —
// preventing replay attacks even under concurrent requests.
func (s *service) RefreshTokens(ctx context.Context, refreshToken string) (TokenPair, error) {
	userID, tokenID, err := splitRefreshToken(refreshToken)
	if err != nil {
		return TokenPair{}, ErrTokenInvalid
	}

	redisKey := buildRefreshKey(userID, tokenID)

	rawPayload, err := s.cache.GetDel(redisKey)
	if err != nil || rawPayload == "" {
		return TokenPair{}, ErrRefreshNotFound
	}

	var payload refreshPayload
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		return TokenPair{}, ErrTokenInvalid
	}

	claims := Claims{
		UserID:   payload.UserID,
		FullName: payload.FullName,
		Email:    payload.Email,
		Role:     payload.Role,
	}

	return s.IssueTokenPair(ctx, claims)
}

// Logout revokes all active refresh tokens for a user by deleting every key
// under the "auth:refresh:<userID>:" prefix.
func (s *service) Logout(_ context.Context, userID string) error {
	s.cache.DeleteStartWithKeys(buildRefreshPrefix(userID))
	return nil
}

// ValidateToken parses and fully verifies an RS256 access token.
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

	if parsed.IssuedAt != nil && parsed.IssuedAt.Time.After(time.Now().Add(clockSkewTolerance)) {
		return Claims{}, ErrTokenFuture
	}

	return Claims{
		UserID:    parsed.UserID,
		FullName:  parsed.FullName,
		Email:     parsed.Email,
		Role:      parsed.Role,
		ExpiresAt: parsed.ExpiresAt.Time,
	}, nil
}

// --- internal helpers ---

func (s *service) signAccessToken(claims Claims) (string, time.Time, error) {
	privateKey, err := s.loadPrivateKey()
	if err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.accessTokenExpirationMinutes) * time.Minute)

	c := jwtClaims{
		UserID:   claims.UserID,
		FullName: claims.FullName,
		Email:    claims.Email,
		Role:     claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *service) storeRefreshToken(claims Claims) (string, error) {
	tokenID := uuid.Must(uuid.NewV4()).String()
	ttl := time.Duration(s.refreshTokenExpirationDays) * 24 * time.Hour

	payload := refreshPayload{
		UserID:   claims.UserID,
		FullName: claims.FullName,
		Email:    claims.Email,
		Role:     claims.Role,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal refresh payload: %w", err)
	}

	redisKey := buildRefreshKey(claims.UserID, tokenID)
	if err := s.cache.SetWithExpireDuration(redisKey, string(payloadJSON), ttl); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	// Opaque token handed to the client: "<userID>.<tokenID>"
	return claims.UserID + "." + tokenID, nil
}

func buildRefreshKey(userID, tokenID string) string {
	return fmt.Sprintf("%s:%s:%s", refreshKeyPrefix, userID, tokenID)
}

func buildRefreshPrefix(userID string) string {
	return fmt.Sprintf("%s:%s:", refreshKeyPrefix, userID)
}

// splitRefreshToken splits "<userID>.<tokenID>" at the last dot.
func splitRefreshToken(raw string) (userID, tokenID string, err error) {
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '.' {
			return raw[:i], raw[i+1:], nil
		}
	}
	return "", "", errors.New("malformed refresh token")
}
