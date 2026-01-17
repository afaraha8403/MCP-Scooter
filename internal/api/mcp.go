package api

import (
	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

// Re-export or alias if needed, but better to use registry.JSONRPCRequest directly
type JSONRPCRequest = registry.JSONRPCRequest
type JSONRPCResponse = registry.JSONRPCResponse
type JSONRPCError = registry.JSONRPCError

// Standard JSON-RPC error codes
const (
	ParseError     = registry.ParseError
	InvalidRequest = registry.InvalidRequest
	MethodNotFound = registry.MethodNotFound
	InvalidParams  = registry.InvalidParams
	InternalError  = registry.InternalError
)

// NewJSONRPCResponse creates a success response.
func NewJSONRPCResponse(id interface{}, result interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCErrorResponse creates an error response.
func NewJSONRPCErrorResponse(id interface{}, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}
