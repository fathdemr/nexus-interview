package httputil

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// GinJSON writes a JSON response and attaches request/response bodies to the
// gin context so the logging middleware can capture them.
// On 4xx/5xx responses the message field is replaced with a safe reference ID
// to prevent internal error details from reaching the client.
func GinJSON(c *gin.Context, status int, response, request any) {
	c.Set("x-log-response-body", response)
	c.Set("x-log-request-body", request)
	c.Set("x-log-enable", true)

	if status >= 400 {
		response = maskErrorResponse(response)
	}

	c.JSON(status, response)
}

// maskErrorResponse replaces the message in a BaseServiceResponse with a
// reference ID, or redacts the entire payload when it is not a BaseServiceResponse.
func maskErrorResponse(response any) any {
	refID := fmt.Sprintf("# System Error # Ref ID: %d", time.Now().Unix())

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return NewErrorResponse(refID, "INTERNAL_ERROR")
	}

	var base BaseServiceResponse
	if err := json.Unmarshal(responseBytes, &base); err != nil || base.Code == "" {
		// Not a BaseServiceResponse — return a clean error envelope
		return NewErrorResponse(refID, "INTERNAL_ERROR")
	}

	base.Message = refID
	return base
}
