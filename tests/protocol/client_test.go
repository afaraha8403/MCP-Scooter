package protocol

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocol_Initialize(t *testing.T) {
	scooterURL := os.Getenv("SCOOTER_URL")
	if scooterURL == "" {
		t.Skip("SCOOTER_URL not set")
	}

	client := NewClient(scooterURL, "work", "")
	resp, err := client.Initialize()
	require.NoError(t, err)
	assert.Nil(t, resp.Error)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
}

func TestProtocol_ListTools(t *testing.T) {
	scooterURL := os.Getenv("SCOOTER_URL")
	if scooterURL == "" {
		t.Skip("SCOOTER_URL not set")
	}

	client := NewClient(scooterURL, "work", "")
	_, err := client.Initialize()
	require.NoError(t, err)

	resp, err := client.ListTools()
	require.NoError(t, err)
	assert.Nil(t, resp.Error)

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)

	// Should at least have builtin tools
	foundFind := false
	for _, tool := range result.Tools {
		if tool.Name == "scooter_find" {
			foundFind = true
			break
		}
	}
	assert.True(t, foundFind, "scooter_find tool not found")
}

func TestProtocol_CallBuiltinTool(t *testing.T) {
	scooterURL := os.Getenv("SCOOTER_URL")
	if scooterURL == "" {
		t.Skip("SCOOTER_URL not set")
	}

	client := NewClient(scooterURL, "work", "")
	_, err := client.Initialize()
	require.NoError(t, err)

	resp, err := client.CallTool("scooter_find", map[string]interface{}{
		"query": "search",
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Error)

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)
}
