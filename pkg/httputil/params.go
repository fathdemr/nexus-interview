package httputil

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseUintParam extracts a uint path parameter by name.
// Returns 0 if the parameter is missing or not a valid positive integer.
// The handler is responsible for returning 400 when 0 is returned.
func ParseUintParam(c *gin.Context, key string) uint {
	val, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0
	}
	return uint(val)
}
