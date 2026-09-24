# Auth — JWT Middleware, Actor, Token Helpers

## Overview

Authentication is split across two layers:

- **`internal/auth/`** — token issuance and validation logic (service + port)
- **`pkg/middleware/`** — Gin middleware that enforces auth on route groups

The middleware depends on the `auth.TokenValidator` interface, never on the concrete auth service. This keeps the middleware decoupled and testable.

---

## Actor Model

`Actor` represents the authenticated identity extracted from a valid JWT. It is set on the Gin context by the middleware and read by handlers and services that need the caller's identity.

Lives in `pkg/httputil/actor.go` (or `internal/auth/actor.go` if kept domain-side).

```go
// Actor represents the authenticated user extracted from a verified JWT token.
type Actor struct {
    // Id is the unique identifier of the authenticated user.
    // Sourced from the "id" JWT claim.
    // Example: "usr_01HXQ3Z7W9FKBJ2M4N6P8R5T"
    Id string `json:"id"`

    // FullName is the display name of the user.
    // Sourced from the "Name" JWT claim.
    // Example: "Jane Smith"
    FullName string `json:"full_name"`

    // Email is the user's email address.
    // Sourced from the "email" JWT claim.
    // Example: "jane.smith@company.com"
    Email string `json:"email"`

    // Role determines what the user is authorized to do.
    // Sourced from the "Role" JWT claim.
    // Example: "admin" | "recruiter" | "candidate"
    Role string `json:"role"`

    // ClientIP is the request's remote IP, populated by the middleware — not from the token.
    // Used for audit logging.
    // Example: "203.0.113.42"
    ClientIP string `json:"client_ip"`
}
```

---

## Token Helpers

Lives in `pkg/tokenhelper/tokenhelper.go`.

```go
// ParseToken safely extracts a named claim from a validated JWT token.
// Returns an empty string if the token is invalid or the claim does not exist.
func ParseToken(token *jwt.Token, claimName string) interface{} {
    if token.Valid && token.Claims.(jwt.MapClaims)[claimName] != nil {
        return token.Claims.(jwt.MapClaims)[claimName]
    }
    return ""
}

// GetActor retrieves the Actor set by the auth middleware from the Gin context.
// Returns a zero-value Actor pointer if none was set (unauthenticated route).
func GetActor(c *gin.Context) *Actor {
    val, exists := c.Get("actor")
    if !exists {
        return &Actor{}
    }
    actor, ok := val.(Actor)
    if !ok {
        return &Actor{}
    }
    actor.ClientIP = c.ClientIP()
    return &actor
}
```

### Usage in Handlers

```go
func (h *Handler) createInterview(c *gin.Context) {
    actor := tokenhelper.GetActor(c)

    interview, err := h.service.CreateInterview(c.Request.Context(), actor.Id, req)
    // ...
}
```

---

## Middleware

Lives in `pkg/middleware/auth.go`. The `Middlewares` struct holds configuration dependencies (accessed via the `Params` abstraction over Viper).

```go
// pkg/middleware/auth.go

var (
    publicKeyCache     *rsa.PublicKey
    publicKeyCacheLock sync.RWMutex
)

// Middlewares holds shared dependencies for all middleware functions.
type Middlewares struct {
    // Params provides runtime config access (wraps Viper).
    Params ParamsReader
}

// ParamsReader is the interface Middlewares uses to read config values.
// Keeps the middleware decoupled from Viper directly.
type ParamsReader interface {
    GetString(key string) string
    GetInt64(key string) int64
}

func NewMiddlewares(params ParamsReader) *Middlewares {
    return &Middlewares{Params: params}
}
```

### loadPublicKey

RSA public key is loaded once and cached. Uses a double-checked locking pattern to avoid redundant parsing on concurrent startup requests.

```go
// loadPublicKey loads and caches the RSA public key from config.
// Thread-safe via double-checked locking.
func (m *Middlewares) loadPublicKey() (*rsa.PublicKey, error) {
    publicKeyCacheLock.RLock()
    if publicKeyCache != nil {
        publicKeyCacheLock.RUnlock()
        return publicKeyCache, nil
    }
    publicKeyCacheLock.RUnlock()

    publicKeyCacheLock.Lock()
    defer publicKeyCacheLock.Unlock()

    // Re-check after acquiring write lock — another goroutine may have populated it
    if publicKeyCache != nil {
        return publicKeyCache, nil
    }

    raw := m.Params.GetString("JWT_RSA_PUBLIC_KEY")
    if raw == "" {
        return nil, fmt.Errorf("JWT_RSA_PUBLIC_KEY is not configured")
    }

    raw = strings.ReplaceAll(raw, `\n`, "\n")

    key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(raw))
    if err != nil {
        return nil, fmt.Errorf("parse RSA public key: %w", err)
    }

    publicKeyCache = key
    return publicKeyCache, nil
}
```

### CheckToken

Hard enforcement middleware. Aborts with 401 on any token problem. Attach to route groups that must always be authenticated.

```go
// CheckToken verifies a Bearer JWT (RS256) and populates the Actor on the context.
// Aborts the request with 401 if the token is missing, invalid, or fails claim checks.
func (m *Middlewares) CheckToken(c *gin.Context) {
    c.Header("Access-Control-Allow-Origin", "*")
    if c.Request.Method == http.MethodOptions {
        if headers := c.Request.Header["Access-Control-Request-Headers"]; len(headers) > 0 {
            c.Header("Access-Control-Allow-Headers", headers[0])
        }
    }

    tokenStr := extractBearerToken(c)

    publicKey, err := m.loadPublicKey()
    if err != nil {
        c.AbortWithStatusJSON(http.StatusInternalServerError, NewErrorResponse("public key unavailable", "INTERNAL_ERROR"))
        return
    }

    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return publicKey, nil
    })

    if err != nil || !token.Valid {
        c.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "UNAUTHORIZED"))
        return
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        c.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse("invalid token claims", "UNAUTHORIZED"))
        return
    }

    if err := m.validateTimeClaims(claims); err != nil {
        c.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(err.Error(), "UNAUTHORIZED"))
        return
    }

    m.setActorOnContext(c, token)
    c.Next()
}
```

### CheckTokenWithoutAbort

Soft variant. Attempts to parse the token and populate the Actor, but always calls `c.Next()` regardless of outcome. Use on routes that serve both authenticated and anonymous users with different behaviour.

```go
// CheckTokenWithoutAbort attempts to verify the token and populate the Actor,
// but never aborts. Routes must check whether an Actor is present themselves.
func (m *Middlewares) CheckTokenWithoutAbort(c *gin.Context) {
    c.Header("Access-Control-Allow-Origin", "*")

    tokenStr := extractBearerToken(c)

    publicKey, err := m.loadPublicKey()
    if err != nil {
        c.Next()
        return
    }

    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return publicKey, nil
    })

    if err != nil || !token.Valid {
        c.Next()
        return
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || m.validateTimeClaims(claims) != nil {
        c.Next()
        return
    }

    m.setActorOnContext(c, token)
    c.Next()
}
```

### Shared Helpers

```go
// extractBearerToken strips the "Bearer " prefix from the Authorization header.
func extractBearerToken(c *gin.Context) string {
    raw := c.GetHeader("Authorization")
    raw = strings.TrimPrefix(raw, "Bearer ")
    raw = strings.TrimPrefix(raw, "bearer ")
    return strings.TrimSpace(raw)
}

// validateTimeClaims checks iat and nbf against the current time with clock skew tolerance.
func (m *Middlewares) validateTimeClaims(claims jwt.MapClaims) error {
    now := time.Now().Unix()
    skew := m.Params.GetInt64("CLOCK_SKEW_SECONDS")
    if skew == 0 {
        skew = 300
    }

    if iat, ok := claims["iat"].(float64); ok {
        if int64(iat) > now+skew {
            return fmt.Errorf("token issued in the future")
        }
    }

    if nbf, ok := claims["nbf"].(float64); ok {
        if int64(nbf) > now+skew {
            return fmt.Errorf("token not yet valid")
        }
    }

    return nil
}

// setActorOnContext extracts claims from a valid token and sets them on the context.
func (m *Middlewares) setActorOnContext(c *gin.Context, token *jwt.Token) {
    userID   := tokenhelper.ParseToken(token, "id").(string)
    fullName := tokenhelper.ParseToken(token, "Name").(string)
    role     := tokenhelper.ParseToken(token, "Role").(string)
    exp      := tokenhelper.ParseToken(token, "exp").(float64)

    actor := Actor{
        Id:       userID,
        FullName: fullName,
        Role:     role,
        ClientIP: c.ClientIP(),
    }

    c.Set("actor",   actor)
    c.Set("user_id", userID)
    c.Set("user_name", fullName)
    c.Set("role",    role)
    c.Set("exp",     time.Unix(int64(exp), 0))
}
```

---

## Wiring in main.go

```go
mw := middleware.NewMiddlewares(viperParamsAdapter)

// Public routes
public := router.Group("/api/v1")
auth.NewHandler(authService).RegisterRoutes(public.Group("/auth"))

// Hard-enforced auth
protected := router.Group("/api/v1")
protected.Use(mw.CheckToken)
candidate.NewHandler(candidateService).RegisterRoutes(protected.Group("/candidates"))

// Optional auth (serves authenticated and anonymous differently)
hybrid := router.Group("/api/v1")
hybrid.Use(mw.CheckTokenWithoutAbort)
interview.NewHandler(interviewService).RegisterRoutes(hybrid.Group("/interviews/public"))
```

---

## JWT Claim Conventions

| Claim | Type | Description |
|---|---|---|
| `id` | string | Unique user identifier |
| `Name` | string | User's full name |
| `email` | string | User's email address |
| `Role` | string | User's role (`admin`, `recruiter`, `candidate`) |
| `iat` | int64 | Issued at (Unix timestamp) |
| `nbf` | int64 | Not before (Unix timestamp) |
| `exp` | int64 | Expiry (Unix timestamp) |

> Claim names (`Name`, `Role`) match the external auth service's token schema. Do not rename them without coordinating with the issuer.

---

## Context Keys

| Key | Type | Set by | Description |
|---|---|---|---|
| `actor` | `Actor` | `CheckToken` / `CheckTokenWithoutAbort` | Full actor struct for the request |
| `user_id` | `string` | `CheckToken` / `CheckTokenWithoutAbort` | Shorthand for actor.Id |
| `user_name` | `string` | `CheckToken` / `CheckTokenWithoutAbort` | Shorthand for actor.FullName |
| `role` | `string` | `CheckToken` / `CheckTokenWithoutAbort` | Shorthand for actor.Role |
| `exp` | `time.Time` | `CheckToken` / `CheckTokenWithoutAbort` | Token expiry time |

Always retrieve the actor via `tokenhelper.GetActor(c)`. Do not read raw context keys directly in handlers.
