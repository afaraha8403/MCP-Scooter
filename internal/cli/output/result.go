package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mcp-scooter/scooter/internal/cli/client"
)

type CallResult struct {
	Raw     *client.CallResult
	Content []client.ContentBlock
}

func NewCallResult(raw *client.CallResult) *CallResult {
	return &CallResult{
		Raw:     raw,
		Content: raw.Content,
	}
}

func (r *CallResult) Text(joiner string) string {
	var parts []string
	for _, c := range r.Content {
		if c.Type == "text" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, joiner)
}

func (r *CallResult) JSON() (string, error) {
	data, err := json.MarshalIndent(r.Raw, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *CallResult) Markdown() string {
	var sb strings.Builder
	for _, c := range r.Content {
		switch c.Type {
		case "text":
			sb.WriteString(c.Text)
			sb.WriteString("\n\n")
		case "image":
			sb.WriteString("![Image](data:image/png;base64,")
			// Assuming Data contains base64 string for images
			if str, ok := c.Data.(string); ok {
				sb.WriteString(str)
			}
			sb.WriteString(")\n\n")
		case "resource":
			sb.WriteString(fmt.Sprintf("### Resource: %v\n\n", c.Data))
		}
	}
	return strings.TrimSpace(sb.String())
}

func (r *CallResult) IsError() bool {
	return r.Raw.IsError
}
