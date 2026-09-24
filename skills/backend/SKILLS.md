# Go Backend Skill — Index

This skill defines how Go backend code is written across all projects using this stack.
Each topic lives in its own focused file. Load only the file relevant to the task at hand.

---

## Stack

| Package | Purpose |
|---|---|
| `github.com/gin-gonic/gin` | HTTP router and middleware |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM for PostgreSQL |
| `github.com/golang-jwt/jwt/v5` | JWT creation and validation |
| `github.com/spf13/viper` | Configuration loading (env, file) |
| `github.com/samber/lo` | Generic collection utilities |

---

## Skill Files

### [`architecture.md`](./architecture.md)
Project layout, file responsibilities (`port.go`, `service.go`, `repository.go`, `handler.go`, `provider/`), SOLID principles with concrete examples, dependency injection root pattern, adapter pattern for external services, module port pattern.

> Load when: designing a new module, adding a provider, wiring dependencies in `main.go`.

---

### [`conventions.md`](./conventions.md)
Naming rules (variables, functions, interfaces, errors, enums), struct field documentation format (what / why / example), comments policy (when to write, when not to), error handling (wrapping, sentinel errors, where to log).

> Load when: writing any new code, reviewing naming or comment style, handling errors.

---

### [`packages.md`](./packages.md)
GORM (connection setup, repository pattern, model tags, migrations), Viper (typed `Config` struct, `Load()`, rules), samber/lo (filter, map, find, groupby, contains with rules and examples).

> Load when: writing a repository, adding a config field, transforming slices.

---

### [`http.md`](./http.md)
Gin handler structure (`Handler` struct, `RegisterRoutes`, 4-step handler shape), `GinJSON` helper (logging hooks + error masking), `BaseServiceResponse` envelope (`NewSuccessResponse`, `NewErrorResponse`, `SetBaseResponse`), HTTP status + code conventions table, request/response DTO patterns, route registration in `main.go`.

> Load when: writing a handler, adding an endpoint, defining a response shape.

---

### [`auth.md`](./auth.md)
`Actor` struct (fields + JWT claim sources), `ParseToken` and `GetActor` helpers, `Middlewares` struct with `ParamsReader` interface, `loadPublicKey` (double-checked locking), `CheckToken` (hard abort on failure), `CheckTokenWithoutAbort` (soft — never aborts), shared helpers (`extractBearerToken`, `validateTimeClaims`, `setActorOnContext`), JWT claim conventions table, Gin context key table.

> Load when: writing or modifying middleware, reading the actor in a handler, wiring auth in `main.go`.

---

## Quick Reference

| Question | Answer |
|---|---|
| Where do interfaces live? | `internal/<module>/port.go` |
| Where does business logic live? | `internal/<module>/service.go` |
| Where does DB access live? | `internal/<module>/repository.go` |
| Where do HTTP handlers live? | `internal/<module>/handler.go` |
| Where are external adapters? | `internal/<module>/provider/<name>.go` |
| Where is config loaded? | `pkg/config/config.go` — once at startup |
| Where are concrete types created? | `cmd/server/main.go` only |
| How to send a JSON response? | Always via `GinJSON` — never `c.JSON` directly |
| How to read the caller's identity? | `tokenhelper.GetActor(c)` |
| When to use `lo`? | Slice filter / map / find instead of manual loops |
| When to add a comment? | Explain *why*, not *what*. Always on exported symbols and struct fields. |
| Where to log errors? | Handler layer only — not in service or repository |
