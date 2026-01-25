package inference

import (
	"strings"
)

func InferCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}

	first := args[0]

	// If contains dot, it's likely a tool call: server.tool
	if strings.Contains(first, ".") && !strings.HasPrefix(first, "-") {
		return "call", args
	}

	// For now, we don't have a list of known servers here.
	// In a full implementation, we might check against a local cache.

	return "", args
}
