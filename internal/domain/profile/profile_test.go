package profile_test

import (
	"testing"

	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProfile_Unmarshal(t *testing.T) {
	yamlData := `
id: work-profile
remote_auth_mode: "oauth2"
remote_server_url: "https://mcp.acme-corp.com"
env:
  AWS_REGION: "us-east-1"
allow_tools:
  - "jira-mcp"
`

	var p profile.Profile
	err := yaml.Unmarshal([]byte(yamlData), &p)
	require.NoError(t, err)

	assert.Equal(t, "work-profile", p.ID)
	assert.Equal(t, "oauth2", p.RemoteAuthMode)
	assert.Equal(t, "https://mcp.acme-corp.com", p.RemoteServerURL)
	assert.Equal(t, "us-east-1", p.Env["AWS_REGION"])
	assert.Contains(t, p.AllowTools, "jira-mcp")
}

func TestProfile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		profile profile.Profile
		wantErr bool
	}{
		{
			name: "valid profile",
			profile: profile.Profile{
				ID: "work",
			},
			wantErr: false,
		},
		{
			name: "missing id",
			profile: profile.Profile{
				ID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
