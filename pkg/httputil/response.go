package httputil

// BaseServiceResponse is the standard envelope for all API responses.
// Every handler must return this type via GinJSON — never call c.JSON directly.
type BaseServiceResponse struct {
	// Success indicates whether the operation completed without error.
	// Example: true
	Success bool `json:"success" gorm:"-"`

	// Message is a human-readable description of the outcome.
	// On 4xx/5xx responses this field is masked with a reference ID before reaching the client.
	// Example: "candidate created successfully"
	Message string `json:"message" gorm:"-"`

	// Code is a machine-readable status identifier for the client to act on.
	// Use SCREAMING_SNAKE_CASE.
	// Example: "CREATED" | "NOT_FOUND" | "VALIDATION_ERROR" | "INTERNAL_ERROR"
	Code string `json:"code" gorm:"-"`

	// Data carries the operation's result payload.
	// Omitted from the JSON output when nil.
	Data any `json:"data,omitempty" gorm:"-"`
}

// SetBaseResponse populates all fields in a single call.
// Useful when embedding BaseServiceResponse in a larger response struct.
func (r *BaseServiceResponse) SetBaseResponse(success bool, message, code string, data any) {
	r.Success = success
	r.Message = message
	r.Code = code
	r.Data = data
}

// NewSuccessResponse constructs a success envelope with a data payload.
func NewSuccessResponse(message, code string, data any) BaseServiceResponse {
	return BaseServiceResponse{Success: true, Message: message, Code: code, Data: data}
}

// NewErrorResponse constructs an error envelope without a data payload.
func NewErrorResponse(message, code string) BaseServiceResponse {
	return BaseServiceResponse{Success: false, Message: message, Code: code}
}
