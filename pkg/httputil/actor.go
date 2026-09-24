package httputil

import "github.com/gin-gonic/gin"

// Actor represents the authenticated identity extracted from a verified JWT token.
// Set on the gin context by CheckToken middleware; read via GetActor.
type Actor struct {
	// Id is the unique identifier of the authenticated user.
	// Sourced from the "id" JWT claim.
	// Example: "usr_01HXQ3Z7W9FKBJ2M4N6P8R5T"
	Id string `json:"id"`

	// FullName is the display name of the user.
	// Sourced from the "name" JWT claim.
	// Example: "Jane Smith"
	FullName string `json:"full_name"`

	// Email is the user's email address.
	// Sourced from the "email" JWT claim.
	// Example: "jane.smith@company.com"
	Email string `json:"email"`

	// Role determines what the user is authorised to do.
	// Sourced from the "role" JWT claim.
	// Example: "admin" | "recruiter" | "candidate"
	Role string `json:"role"`

	// ClientIP is the request's remote IP, populated by the middleware — not from the token.
	// Used for audit logging.
	// Example: "203.0.113.42"
	ClientIP string `json:"client_ip"`
}

const actorContextKey = "actor"

// GetActor retrieves the Actor set by the auth middleware from the gin context.
// Returns a pointer to a zero-value Actor when no authenticated actor exists.
func GetActor(c *gin.Context) *Actor {
	val, exists := c.Get(actorContextKey)
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

// SetActor stores an Actor on the gin context. Called by auth middleware.
func SetActor(c *gin.Context, actor Actor) {
	c.Set(actorContextKey, actor)
}
