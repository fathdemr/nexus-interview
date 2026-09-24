# Package Patterns — GORM, Viper, samber/lo

## GORM

### Connection Setup

GORM connection lives in `pkg/database/database.go`. It is created once at startup and passed to every repository constructor.

```go
// pkg/database/database.go

func Connect(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Warn),
    })
    if err != nil {
        return nil, fmt.Errorf("open database connection: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("get underlying sql.DB: %w", err)
    }

    sqlDB.SetMaxOpenConns(25)
    sqlDB.SetMaxIdleConns(5)
    sqlDB.SetConnMaxLifetime(5 * time.Minute)

    return db, nil
}
```

### Repository Pattern

The repository struct is unexported. It is constructed via `New<Module>Repository` which returns the interface type — callers never hold the concrete struct.

```go
// internal/candidate/repository.go

type repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) CandidateRepository {
    return &repository{db: db}
}

func (r *repository) FindByID(ctx context.Context, id uint) (Candidate, error) {
    var c Candidate
    if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return Candidate{}, ErrNotFound
        }
        return Candidate{}, fmt.Errorf("query candidate %d: %w", id, err)
    }
    return c, nil
}

func (r *repository) Save(ctx context.Context, c *Candidate) error {
    if err := r.db.WithContext(ctx).Save(c).Error; err != nil {
        return fmt.Errorf("save candidate: %w", err)
    }
    return nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (Candidate, error) {
    var c Candidate
    if err := r.db.WithContext(ctx).Where("email = ?", email).First(&c).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return Candidate{}, ErrNotFound
        }
        return Candidate{}, fmt.Errorf("query candidate by email: %w", err)
    }
    return c, nil
}
```

### Rules
- Always pass `ctx` via `.WithContext(ctx)`. Never skip it.
- Map `gorm.ErrRecordNotFound` to the module's own `ErrNotFound` sentinel at the repository boundary. GORM errors must not leak to the service or handler layer.
- Never return a raw `*gorm.DB` from a repository method.
- Soft deletes are handled automatically by GORM via `gorm.Model`'s `DeletedAt` field. Do not implement manual deletion logic unless hard delete is explicitly required.
- Use `Save` for upserts, `Create` for inserts-only. Prefer explicit over implicit.

### Model Tags

GORM struct tags follow this order: `gorm` first, then `json`. Fields that are not persisted use `gorm:"-"`.

```go
type Candidate struct {
    gorm.Model

    // Email is the primary contact and login identifier for the candidate.
    // Must be unique across all candidates.
    // Example: "john.doe@example.com"
    Email string `gorm:"uniqueIndex;not null" json:"email"`

    // FullName is the candidate's display name shown in interview sessions and reports.
    // Example: "John Doe"
    FullName string `gorm:"not null" json:"full_name"`

    // InviteToken is a one-time token sent via email for passwordless login.
    // Nullable — nil means no active invitation.
    // Example: "a3f9c2e1-84bb-4d2a-b6f0-1234567890ab"
    InviteToken *string `gorm:"index" json:"invite_token,omitempty"`

    // InviteExpiresAt is the expiry time of the invite token.
    // Nil when no active invitation exists.
    InviteExpiresAt *time.Time `json:"invite_expires_at,omitempty"`
}
```

### Migrations

Auto-migration runs at startup for all domain models. Register models in `pkg/database/migrate.go`.

```go
// pkg/database/migrate.go

func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &candidate.Candidate{},
        &interview.Interview{},
        &question.Question{},
        &evaluation.Evaluation{},
    )
}
```

---

## Viper

### Config Struct

The entire configuration is loaded once at startup into a typed `Config` struct. Never call `viper.Get*` outside of `pkg/config`. Every other package receives only the fields it needs — pass `cfg.GroqAPIKey` to the Groq constructor, not the whole `cfg`.

```go
// pkg/config/config.go

// Config holds all runtime configuration for the application.
// Values are loaded from environment variables (12-factor) with optional .env file fallback.
// Environment variables take precedence over .env file values.
type Config struct {
    // ServerPort is the TCP port the HTTP server listens on.
    // Example: "8080"
    ServerPort string `mapstructure:"SERVER_PORT"`

    // DatabaseURL is the full PostgreSQL connection string.
    // Example: "postgres://user:pass@localhost:5432/myapp?sslmode=disable"
    DatabaseURL string `mapstructure:"DATABASE_URL"`

    // JWTSecret is the HMAC signing key for JWT tokens.
    // Must be at least 32 characters in production.
    // Example: "change-me-in-production-32chars+"
    JWTSecret string `mapstructure:"JWT_SECRET"`

    // JWTRSAPublicKey is the RSA public key (PEM) for verifying RS256 tokens issued by an external auth service.
    // Newlines are represented as \n in the environment variable and normalized at load time.
    // Example: "-----BEGIN PUBLIC KEY-----\nMIIB..."
    JWTRSAPublicKey string `mapstructure:"JWT_RSA_PUBLIC_KEY"`

    // GroqAPIKey is the API key for the Groq LLM provider.
    // Example: "gsk_xxxxxxxxxxxxxxxxxxxx"
    GroqAPIKey string `mapstructure:"GROQ_API_KEY"`

    // AWSRegion is the AWS region used for all AWS SDK clients (S3, SES, Transcribe, Polly).
    // Example: "eu-central-1"
    AWSRegion string `mapstructure:"AWS_REGION"`

    // ClockSkewSeconds is the tolerance window (in seconds) for JWT iat/nbf validation.
    // Defaults to 300 (5 minutes) if not set.
    // Example: 300
    ClockSkewSeconds int64 `mapstructure:"CLOCK_SKEW_SECONDS"`
}

func Load() (Config, error) {
    viper.AutomaticEnv()
    viper.SetConfigFile(".env")
    _ = viper.ReadInConfig() // .env is optional; env vars always take precedence

    // Normalize RSA key newlines before unmarshalling
    if raw := viper.GetString("JWT_RSA_PUBLIC_KEY"); raw != "" {
        viper.Set("JWT_RSA_PUBLIC_KEY", strings.ReplaceAll(raw, `\n`, "\n"))
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return Config{}, fmt.Errorf("unmarshal config: %w", err)
    }

    if cfg.ClockSkewSeconds == 0 {
        cfg.ClockSkewSeconds = 300
    }

    return cfg, nil
}
```

### Rules
- One `Config` struct, loaded once in `main.go`.
- Pass only the required fields to each constructor — not the full `Config`.
- Never read `viper.GetString` / `viper.GetInt` outside `pkg/config`.
- Document every field with what it is and an example value.

---

## samber/lo

Use `lo` for slice transformations and lookups instead of manual loops. Keep lambdas short — if the logic is more than one line, extract a named function.

### Filter
```go
activeCandidates := lo.Filter(candidates, func(c Candidate, _ int) bool {
    return c.Status == StatusActive
})
```

### Map (transform)
```go
// Extract IDs for a batch query
candidateIDs := lo.Map(sessions, func(s Session, _ int) uint {
    return s.CandidateID
})

// Convert domain models to response DTOs
responses := lo.Map(interviews, func(i Interview, _ int) InterviewResponse {
    return toInterviewResponse(i)
})
```

### Find
```go
// Returns zero value + false if not found — always check the bool
admin, found := lo.Find(users, func(u User) bool {
    return u.Role == RoleAdmin
})
if !found {
    return User{}, ErrNotFound
}
```

### Uniq / GroupBy
```go
uniqueTagIDs := lo.Uniq(tagIDs)

byStatus := lo.GroupBy(interviews, func(i Interview) InterviewStatus {
    return i.Status
})
// byStatus["completed"] → []Interview{...}
```

### Contains
```go
allowedRoles := []Role{RoleAdmin, RoleRecruiter}
if !lo.Contains(allowedRoles, actor.Role) {
    return ErrForbidden
}
```

### Rules
- Prefer `lo` over manual `for` loops for any collection operation expressible as filter / map / find / group.
- If the lambda grows beyond a single expression, extract it as a named function — readability over brevity.
- `lo.Find` always returns `(T, bool)`. Always check the bool; never assume the zero value is acceptable.
- Do not use `lo` for side-effecting loops (e.g., saving each item to DB). Use a plain `for` loop there — side effects in functional combinators are confusing.
