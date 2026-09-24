package admin

import "context"

// Service is the capability this module exposes to HTTP handlers.
// Business logic will be added here as admin features are defined.
type Service interface {
	// Healthcheck verifies the admin service is operational.
	Healthcheck(ctx context.Context) error
}
