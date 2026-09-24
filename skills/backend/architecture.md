# Architecture — Modern Monolith (Microservice-Ready)

## Project Layout

```
project-root/
├── cmd/
│   └── server/
│       └── main.go          # Entry point. Wires all dependencies, starts HTTP server.
├── internal/
│   └── <module>/
│       ├── port.go          # Interfaces the module exposes and depends on.
│       ├── service.go       # Business logic. Depends only on interfaces.
│       ├── repository.go    # Database layer. Implements Repository interface.
│       ├── handler.go       # HTTP handlers. Thin layer — delegates to service.
│       └── provider/        # Adapters for external services (only where needed).
│           └── <name>.go
├── pkg/
│   ├── config/              # Viper-based config loading.
│   ├── logger/              # Structured logger setup.
│   ├── database/            # GORM connection and migration runner.
│   └── middleware/          # Shared Gin middleware (auth, cors, recovery, logging).
├── deployments/
│   └── docker/
│       └── Dockerfile
└── docs/
```

### File Responsibilities

| File | Single responsibility |
|---|---|
| `port.go` | Declares all interfaces the module exposes and all external interfaces it consumes. No concrete types leak in or out. |
| `service.go` | Pure business logic. Constructor receives interfaces only. No `http`, no `gorm`, no provider types. |
| `repository.go` | Data access via GORM. Implements the `Repository` interface from `port.go`. Returns domain types, never raw DB rows. |
| `handler.go` | HTTP boundary. Parses request → calls service → writes response. Zero business logic. |
| `provider/<name>.go` | Adapts an external API (Groq, SES, Polly…) to the internal interface. One file per provider. |

---

## SOLID in Practice

### Single Responsibility
Each file has one reason to change. Handler changes when the HTTP contract changes. Service changes when business rules change. Repository changes when the data model or query changes.

### Open / Closed
Adding a new provider means adding a new file under `provider/`. Existing code is not modified — the new file implements the existing interface and gets wired in `main.go`.

### Liskov Substitution
Every concrete type implementing an interface must be fully substitutable. If `GroqProvider` satisfies `LLMClient`, then `OpenAIProvider` must behave identically from the caller's perspective. Tests are written against the interface, not the concrete type.

### Interface Segregation
Interfaces are small and focused. Split when a consumer uses only a subset of methods.

```go
// Good — callers depend only on what they use
type Transcriber interface {
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
}

type Synthesizer interface {
    Synthesize(ctx context.Context, text string) (io.Reader, error)
}

// Avoid — forces every consumer to depend on both capabilities
type SpeechService interface {
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
    Synthesize(ctx context.Context, text string) (io.Reader, error)
}
```

### Dependency Inversion
Services and handlers never instantiate their dependencies. All concrete types are created in `main.go` and injected through constructors. The business logic layer sees only interfaces.

---

## Dependency Injection Root (main.go)

`main.go` is the only place concrete types are instantiated. Wiring order: infrastructure → repositories → providers → services → handlers.

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("connect database: %v", err)
    }

    // Repositories
    candidateRepo := candidate.NewRepository(db)
    interviewRepo := interview.NewRepository(db)

    // External providers
    groqClient := groq.NewClient(cfg.GroqAPIKey)
    sesMailer  := ses.NewMailer(cfg.AWSRegion)

    // Services
    authService      := auth.NewService(cfg.JWTSecret)
    candidateService := candidate.NewService(candidateRepo, sesMailer)
    interviewService := interview.NewService(interviewRepo, groqClient)

    // HTTP
    router := gin.New()
    router.Use(gin.Recovery(), middleware.Logger())

    v1 := router.Group("/api/v1")
    v1.Use(middleware.CheckToken(authService))

    candidate.NewHandler(candidateService).RegisterRoutes(v1.Group("/candidates"))
    interview.NewHandler(interviewService).RegisterRoutes(v1.Group("/interviews"))

    router.Run(":" + cfg.ServerPort)
}
```

---

## Adapter Pattern for External Services

Every external dependency gets a `provider/` adapter. The adapter:
1. Holds the client / config as struct fields.
2. Implements the interface declared in the module's `port.go`.
3. Translates between the external API's types and the module's domain types.

```go
// internal/ai/provider/groq.go

type GroqClient struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
}

func NewClient(apiKey string) *GroqClient {
    return &GroqClient{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        apiKey:     apiKey,
        baseURL:    "https://api.groq.com/openai/v1",
    }
}

// Complete implements the ai.LLMClient interface.
func (g *GroqClient) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
    // build HTTP request → call Groq → map response to ai.CompletionResponse
}
```

Swapping the provider = one new file + one line change in `main.go`. Service and handler are untouched.

---

## Module Port Pattern

```go
// internal/candidate/port.go

// Service is the capability this module exposes to HTTP handlers and other modules.
type Service interface {
    CreateCandidate(ctx context.Context, req CreateRequest) (Candidate, error)
    FindByID(ctx context.Context, id uint) (Candidate, error)
    InviteByEmail(ctx context.Context, candidateID uint) error
}

// Repository is the data access contract this module owns.
type Repository interface {
    Save(ctx context.Context, c *Candidate) error
    FindByID(ctx context.Context, id uint) (Candidate, error)
    FindByEmail(ctx context.Context, email string) (Candidate, error)
}

// Mailer is an external dependency this module consumes.
// The concrete implementation lives in notification/provider/ses.go.
type Mailer interface {
    Send(ctx context.Context, msg notification.Message) error
}
```
