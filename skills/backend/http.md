# HTTP Layer — Gin Handler, GinJSON, BaseServiceResponse

## Handler Structure

Handlers live in `internal/<module>/handler.go`. The `Handler` struct holds the service interface — never the concrete service type. Route registration is a method on `Handler`, called from `main.go`.

```go
// internal/candidate/handler.go

type Handler struct {
    service Service // interface from port.go — never the concrete type
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// RegisterRoutes attaches all candidate routes to the given router group.
// Called once from main.go during server setup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.POST("", h.createCandidate)
    rg.GET("/:id", h.getCandidate)
    rg.PUT("/:id", h.updateCandidate)
    rg.DELETE("/:id", h.deleteCandidate)
}
```

### Handler Method Shape

Every handler follows the same four-step shape: parse → call service → handle error → respond.

```go
func (h *Handler) createCandidate(c *gin.Context) {
    var req CreateCandidateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        GinJSON(c, http.StatusBadRequest, NewErrorResponse("invalid request body", "VALIDATION_ERROR"), req)
        return
    }

    candidate, err := h.service.CreateCandidate(c.Request.Context(), req)
    if err != nil {
        if errors.Is(err, ErrAlreadyExists) {
            GinJSON(c, http.StatusConflict, NewErrorResponse("candidate already exists", "ALREADY_EXISTS"), req)
            return
        }
        GinJSON(c, http.StatusInternalServerError, NewErrorResponse("failed to create candidate", "INTERNAL_ERROR"), req)
        return
    }

    GinJSON(c, http.StatusCreated, NewSuccessResponse("candidate created", "CREATED", candidate), req)
}
```

### Rules
- Never call `c.JSON` directly in handlers. Always use `GinJSON`.
- Handlers contain zero business logic. If an `if` branch is doing anything beyond routing to a service call, it belongs in the service.
- Always pass the parsed `req` as the last argument to `GinJSON` so it gets logged alongside the response.
- Use `c.Request.Context()` when calling services — never `context.Background()`.

---

## GinJSON Helper

`GinJSON` is the single point for writing JSON responses. It handles response/request logging hooks and masks error details on 4xx/5xx responses to avoid leaking internal information to clients.

Lives in `pkg/middleware/gin_helper.go` (or `pkg/httputil/gin_helper.go`).

```go
// pkg/httputil/gin_helper.go

// GinJSON writes a JSON response and attaches request/response bodies to the
// gin context for the logging middleware to pick up.
// On 4xx/5xx responses the message field is replaced with a safe reference ID
// to prevent internal error details from reaching the client.
func GinJSON(c *gin.Context,
	status int,
	response,
	request any) {

	c.Set("x-log-response-body", response)
	c.Set("x-log-request-body", request)
	c.Set("x-log-enable", true)

	// Eğer response status 2xx değilse maskele
	if status >= 400 {
		var now = time.Now()
		// Gelen response BaseServiceResponse mu diye kontrol et -> öyleyse message alanını maskele. Değilse tüm response'u maskele
		var resp models.BaseServiceResponse
		responseBytes, _ := json.Marshal(response)
		err := json.Unmarshal(responseBytes, &resp)
		if err != nil {
			// Base Service Response değil, tüm response'u maskele
			responseStr := Redact(string(responseBytes), fmt.Sprintf("# Sistem Hatası # Ref ID: %d", now.Unix()))
			response = responseStr
		} else {
			// Base Service Response, sadece message alanını maskele
			resp.Message = Redact(resp.Message, fmt.Sprintf("# Sistem Hatası # Ref ID: %d", now.Unix()))
			response = resp
		}
	}
	c.JSON(status, response)
}

// Redact Uzun/şüpheli parçaları maskelemek için kullanılır.
func Redact(s, redactMessage string) string {
	var patterns = []*regexp.Regexp{
		// SELECT ... FROM ...; sorgusunu maskele
		regexp.MustCompile(`(?is)\bselect\b.+?\bfrom\b.+?(?:;|$)`),
		// Çok satırlı SELECT ... FROM ...; sorgusunu maskele
		regexp.MustCompile(`(?is)\bselect\b[\s\S]+?\bfrom\b[\s\S]+?(?:;|$)`),
		// Basit SELECT ... FROM ... sorgusunu maskele
		regexp.MustCompile(`(?is)select[\s\S]+from[\s\S]+`),
		// Insert sorgularını maskele
		regexp.MustCompile(`(?is)\binsert\s+into\s+[A-Za-z_][\w$]*(?:\.[A-Za-z_][\w$]*)*\s*\(.*?\)\s*values\s*\(.*?\)`),
		// Update sorgularını maskele
		regexp.MustCompile(`(?is)\bupdate\s+[A-Za-z_][\w$]*(?:\.[A-Za-z_][\w$]*)*\s+set\s+.*?\s*(?:where\s+.*)?`),
		// Delete sorgularını maskele
		regexp.MustCompile(`(?is)\bdelete\s+from\s+[A-Za-z_][\w$]*(?:\.[A-Za-z_][\w$]*)*\s*(?:where\s+.*)?`),
		// Şema.Tablo veya Şema.Paket.Fonksiyon(...)
		//regexp.MustCompile(`\b[A-Za-z_][\w$]*\.[A-Za-z_][\w$]*(?:\.[A-Za-z_][\w$]*)*\b`),
		// Fonksiyon çağrıları: NAME('...') veya NAME(...)
		regexp.MustCompile(`\b[A-Za-z_][\w$]*\s*$begin:math:text$[^)]*$end:math:text$`),
		// Provider / linked server isimleri
		regexp.MustCompile(`OraOLEDB\.Oracle`),
		regexp.MustCompile(`linked server\s*'[^']+'`),
		// Connection string benzeri
		regexp.MustCompile(`(?i)(User Id|Password|Data Source|Server|Host|Uid|Pwd)\s*=\s*[^;]+`),
		// Tırnak içindeki uzun parçalarda (ör: "SELECT ...") içerik maskele
		//regexp.MustCompile(`"[^"]{20,}"`),

	}
	out := s
	for _, p := range patterns {
		//out = p.ReplaceAllString(out, "Sistem Hatası")
		// if pattern matches, replace all string to "Sistem Hatası"
		if p.MatchString(out) {
			out = redactMessage
			break
		}
	}
	return out
}
```

---

## BaseServiceResponse

All API responses are wrapped in `BaseServiceResponse`. This ensures every endpoint returns a consistent envelope that clients can depend on.

Lives in `pkg/httputil/response.go`.

```go
// pkg/httputil/response.go

// BaseServiceResponse is the standard envelope for all API responses.
// Every handler must return this type (directly or embedded) via GinJSON.
type BaseServiceResponse struct {
    // Success indicates whether the operation completed without error.
    // Example: true
    Success bool `json:"success" gorm:"-"`

    // Message is a human-readable description of the outcome.
    // On error responses this field is masked with a reference ID before reaching the client.
    // Example: "candidate created successfully"
    Message string `json:"message" gorm:"-"`

    // Code is a machine-readable status identifier for the client to act on.
    // Use SCREAMING_SNAKE_CASE. Examples: "CREATED", "NOT_FOUND", "VALIDATION_ERROR", "INTERNAL_ERROR"
    Code string `json:"code" gorm:"-"`

    // Data carries the operation's result payload.
    // Omitted from the JSON output when nil.
    Data any `json:"data,omitempty" gorm:"-"`
}

// SetBaseResponse populates all fields of BaseServiceResponse in one call.
// Useful when embedding BaseServiceResponse in a larger response struct.
func (r *BaseServiceResponse) SetBaseResponse(success bool, message, code string, data any) {
    r.Success = success
    r.Message = message
    r.Code    = code
    r.Data    = data
}

// NewSuccessResponse constructs a success envelope with data payload.
func NewSuccessResponse(message, code string, data any) BaseServiceResponse {
    return BaseServiceResponse{Success: true, Message: message, Code: code, Data: data}
}

// NewErrorResponse constructs an error envelope without a data payload.
func NewErrorResponse(message, code string) BaseServiceResponse {
    return BaseServiceResponse{Success: false, Message: message, Code: code}
}
```

### Usage in Handlers

```go
// Success with data
GinJSON(c, http.StatusOK, NewSuccessResponse("interview retrieved", "OK", interview), nil)

// Success without data (e.g. delete)
GinJSON(c, http.StatusOK, NewSuccessResponse("candidate deleted", "DELETED", nil), nil)

// Error
GinJSON(c, http.StatusNotFound, NewErrorResponse("candidate not found", "NOT_FOUND"), nil)

// Embedding in a richer response struct
type InterviewDetailResponse struct {
    BaseServiceResponse
    // additional envelope-level metadata if ever needed
}
resp := InterviewDetailResponse{}
resp.SetBaseResponse(true, "interview retrieved", "OK", interviewData)
GinJSON(c, http.StatusOK, resp, nil)
```

### Response Code Conventions

| Situation | HTTP Status | Code |
|---|---|---|
| Resource created | 201 | `CREATED` |
| Successful read / update | 200 | `OK` |
| Successful delete | 200 | `DELETED` |
| Validation failure | 400 | `VALIDATION_ERROR` |
| Unauthorized (no/invalid token) | 401 | `UNAUTHORIZED` |
| Forbidden (valid token, wrong role) | 403 | `FORBIDDEN` |
| Resource not found | 404 | `NOT_FOUND` |
| Conflict (duplicate) | 409 | `ALREADY_EXISTS` |
| Unhandled internal error | 500 | `INTERNAL_ERROR` |

---

## Route Registration in main.go

Handlers register their own routes. `main.go` only wires groups and middleware.

```go
router := gin.New()
router.Use(gin.Recovery(), middleware.RequestLogger())

v1 := router.Group("/api/v1")

// Public routes — no auth
authGroup := v1.Group("/auth")
auth.NewHandler(authService).RegisterRoutes(authGroup)

// Protected routes — require valid JWT
protected := v1.Group("")
protected.Use(middleware.CheckToken(authService))

candidate.NewHandler(candidateService).RegisterRoutes(protected.Group("/candidates"))
interview.NewHandler(interviewService).RegisterRoutes(protected.Group("/interviews"))
question.NewHandler(questionService).RegisterRoutes(protected.Group("/questions"))
```

---

## Request / Response DTOs

Request and response structs are defined in the handler's file (or a `dto.go` in the same package). They are never shared across modules — each module owns its own request/response shapes.

```go
// CreateCandidateRequest carries the validated input for candidate creation.
type CreateCandidateRequest struct {
    // FullName is the candidate's display name.
    // Example: "Jane Smith"
    FullName string `json:"full_name" binding:"required"`

    // Email is the candidate's contact address and login identifier.
    // Example: "jane.smith@example.com"
    Email string `json:"email" binding:"required,email"`

    // JobPostingID is the position this candidate is being invited to interview for.
    // Example: 12
    JobPostingID uint `json:"job_posting_id" binding:"required"`
}
```

`binding` tags drive `ShouldBindJSON` validation. Use `binding:"required"` for mandatory fields and standard validators (`email`, `min`, `max`, `oneof`) where applicable.

---

## Utility Helpers

```go
// parseUintParam extracts a uint path parameter and returns 0 on parse failure.
// The handler is responsible for returning 400 if the result is 0.
func parseUintParam(c *gin.Context, key string) uint {
    val, err := strconv.ParseUint(c.Param(key), 10, 64)
    if err != nil {
        return 0
    }
    return uint(val)
}
```
