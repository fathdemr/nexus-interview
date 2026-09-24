# Conventions — Naming, Comments, Error Handling

## Naming

### General Rules
- Use full, descriptive names. Single-letter variables only in short loops (`i`, `j`) or very tight scopes.
- No noise words: `data`, `info`, `manager`, `helper`, `util` in names signal missing design.
- Names should read like a sentence at the call site: `candidateService.InviteByEmail(ctx, id)`.

### Variables
```go
// Good
expiresAt      := time.Now().Add(tokenTTL)
activeSessions := lo.Filter(sessions, func(s Session, _ int) bool { return s.IsActive })

// Avoid
d   := time.Now().Add(ttl)
res := lo.Filter(list, func(x Session, _ int) bool { return x.IsActive })
```

### Functions and Methods
- Follow the `verb + noun` pattern: `CreateCandidate`, `FindByToken`, `ValidateJWT`, `SendInvitation`.
- Constructors are always `New<Type>`: `NewService`, `NewHandler`, `NewRepository`.
- Boolean-returning functions read as assertions: `isExpired`, `hasAccess`, `canRetry`.

### Interfaces
- Single-method interfaces: named after the method with an `-er` suffix — `Transcriber`, `Synthesizer`, `Mailer`, `Validator`.
- Multi-method interfaces: describe the role — `CandidateRepository`, `InterviewService`, `TokenValidator`.

### Errors
Package-level sentinel errors are prefixed with `Err`:
```go
var (
    ErrNotFound           = errors.New("record not found")
    ErrTokenExpired       = errors.New("token has expired")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrAlreadyExists      = errors.New("record already exists")
)
```

### Constants and Enums
Use typed string or int constants grouped by domain:
```go
type InterviewStatus string

const (
    StatusPending    InterviewStatus = "pending"
    StatusInProgress InterviewStatus = "in_progress"
    StatusCompleted  InterviewStatus = "completed"
    StatusCancelled  InterviewStatus = "cancelled"
)
```

---

## Struct Documentation

All exported structs must have field-level comments explaining:
- **what** the field represents
- **why** it exists in this struct
- a concrete **example** value

This applies to GORM models, request/response DTOs, and config structs alike.

```go
// Interview represents a single interview session between a candidate and the AI interviewer.
type Interview struct {
    // gorm.Model embeds ID (uint, primary key), CreatedAt, UpdatedAt, DeletedAt (soft delete).
    gorm.Model

    // CandidateID is the foreign key linking to the candidates table.
    // Example: 42
    CandidateID uint `gorm:"not null;index"`

    // JobPostingID ties this session to the specific job opening being interviewed for.
    // Example: 7
    JobPostingID uint `gorm:"not null"`

    // Status tracks the lifecycle of the interview session.
    // Values: "pending" | "in_progress" | "completed" | "cancelled"
    Status InterviewStatus `gorm:"not null;default:'pending'"`

    // StartedAt is set when the candidate joins the room and the session begins.
    // Nil means the candidate has not started yet.
    StartedAt *time.Time

    // CompletedAt is set when all questions are answered or the session is terminated.
    // Nil means still in progress or not started.
    CompletedAt *time.Time

    // RecordingURL is the S3 object URL of the full video recording.
    // Nil until the recording is uploaded after the session ends.
    // Example: "https://s3.amazonaws.com/nexus-recordings/interviews/42/session.webm"
    RecordingURL *string `gorm:"type:text"`
}
```

---

## Comments Policy

Comments explain **why**, not **what**. Well-named code makes the what obvious.

```go
// Good — explains a non-obvious business decision
// Soft delete is intentional: candidates must remain linked to historical
// interview records even after account removal.
if err := r.db.Delete(&candidate).Error; err != nil { ... }

// Avoid — restates what the code already says
// Delete the candidate from the database
if err := r.db.Delete(&candidate).Error; err != nil { ... }
```

**Add comments to:**
- All exported types, interfaces, and functions (a single line is enough).
- All struct fields (what, why, example — as shown above).
- Non-obvious algorithmic decisions or business rules.
- Any workaround or temporary solution, with a `// TODO:` referencing the reason.

**Do not add comments to:**
- Every line of a function body.
- Self-explanatory getters or setters.
- Code that is already clear from naming alone.

---

## Error Handling

### Wrapping
Always wrap errors with context using `fmt.Errorf("operation description: %w", err)`. This builds a navigable call stack in logs.

```go
func (s *service) FindCandidate(ctx context.Context, id uint) (Candidate, error) {
    candidate, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return Candidate{}, fmt.Errorf("find candidate %d: %w", id, err)
    }
    return candidate, nil
}
```

### Sentinel Errors
Define sentinel errors at the package level for conditions callers need to differentiate. Map infrastructure errors (e.g. `gorm.ErrRecordNotFound`) to domain sentinels at the repository boundary — they must not leak up the stack.

```go
// repository.go
func (r *repository) FindByID(ctx context.Context, id uint) (Candidate, error) {
    var c Candidate
    if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return Candidate{}, ErrNotFound // domain sentinel, not gorm error
        }
        return Candidate{}, fmt.Errorf("query candidate %d: %w", id, err)
    }
    return c, nil
}
```

### Where to Log
- **Repository / Service layer:** do not log. Just return a wrapped error.
- **Handler layer:** log at the boundary where the full request context (path, method, actor) is available.
- Never return `nil, nil` — if there is no result, return a typed error (`ErrNotFound`).

```go
// handler.go
func (h *Handler) getCandidate(c *gin.Context) {
    id := parseUintParam(c, "id")
    candidate, err := h.service.FindByID(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, candidate.ErrNotFound) {
            GinJSON(c, http.StatusNotFound, errorResponse("candidate not found"), nil)
            return
        }
        // Log here — we have request context
        log.Error("get candidate failed", "id", id, "err", err)
        GinJSON(c, http.StatusInternalServerError, errorResponse("internal error"), nil)
        return
    }
    GinJSON(c, http.StatusOK, candidate, nil)
}
```

### Error Response Consistency
Always return errors through `GinJSON` (see `http.md`) so error responses are logged and masked uniformly. Never call `c.JSON` directly in handlers.
