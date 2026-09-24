package admin

import "context"

type service struct{}

// NewService constructs the admin Service.
// Dependencies will be injected here as features are built.
func NewService() Service {
	return &service{}
}

func (s *service) Healthcheck(_ context.Context) error {
	return nil
}
