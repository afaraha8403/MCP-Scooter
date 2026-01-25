package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mcp-scooter/scooter/tests/protocol"
	"gopkg.in/yaml.v3"
)

// Scenario represents a test scenario defined in YAML.
type Scenario struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Timeout     string         `yaml:"timeout"`
	Requires    ScenarioReqs   `yaml:"requires"`
	Steps       []ScenarioStep `yaml:"steps"`
}

type ScenarioReqs struct {
	Env      []string `yaml:"env"`
	Registry []string `yaml:"registry"`
}

type ScenarioStep struct {
	Name   string                 `yaml:"name"`
	Action string                 `yaml:"action"`
	Tool   string                 `yaml:"tool,omitempty"`
	Args   map[string]interface{} `yaml:"args,omitempty"`
	Expect map[string]interface{} `yaml:"expect"`
}

// ScenarioRunner executes test scenarios.
type ScenarioRunner struct {
	Client *protocol.MCPTestClient
}

// LoadScenario loads a scenario from a YAML file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Run executes a single scenario.
func (r *ScenarioRunner) Run(s *Scenario) error {
	fmt.Printf("Running scenario: %s\n", s.Name)

	for _, step := range s.Steps {
		fmt.Printf("  Step: %s\n", step.Name)
		var resp *protocol.JSONRPCResponse
		var err error

		switch step.Action {
		case "initialize":
			resp, err = r.Client.Initialize()
		case "list_tools":
			resp, err = r.Client.ListTools()
		case "call_tool":
			resp, err = r.Client.CallTool(step.Tool, step.Args)
		case "wait":
			seconds, _ := step.Args["seconds"].(int)
			if seconds == 0 {
				seconds = 1
			}
			fmt.Printf("  Waiting %d seconds...\n", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			continue
		default:
			return fmt.Errorf("unknown action: %s", step.Action)
		}

		if err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		if err := r.validateExpectations(step.Expect, resp); err != nil {
			return fmt.Errorf("step %s expectation failed: %w", step.Name, err)
		}
	}

	return nil
}

func (r *ScenarioRunner) validateExpectations(expect map[string]interface{}, resp *protocol.JSONRPCResponse) error {
	for key, expectedValue := range expect {
		switch key {
		case "error":
			if expectedValue == nil {
				if resp.Error != nil {
					return fmt.Errorf("expected no error, got: %s", resp.Error.Message)
				}
			} else {
				// Handle expected error check if needed
			}
		case "tools_contain":
			var result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			}
			json.Unmarshal(resp.Result, &result)
			
			// Debug: print tools found
			fmt.Printf("    Found tools: ")
			for _, t := range result.Tools {
				fmt.Printf("%s, ", t.Name)
			}
			fmt.Println()

			expectedTools := expectedValue.([]interface{})
			for _, et := range expectedTools {
				found := false
				for _, t := range result.Tools {
					if t.Name == et.(string) {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("expected tool %s not found", et)
				}
			}
		case "result.content":
			if expectedValue == "not_empty" {
				var result struct {
					Content []interface{} `json:"content"`
				}
				json.Unmarshal(resp.Result, &result)
				if len(result.Content) == 0 {
					return fmt.Errorf("expected non-empty content")
				}
			}
		case "result_contains":
			expectedStr := expectedValue.(string)
			if !strings.Contains(string(resp.Result), expectedStr) {
				return fmt.Errorf("expected result to contain '%s', but it didn't. Result: %s", expectedStr, string(resp.Result))
			}
		case "result_not_contains":
			unexpectedStr := expectedValue.(string)
			if strings.Contains(string(resp.Result), unexpectedStr) {
				return fmt.Errorf("expected result NOT to contain '%s', but it did. Result: %s", unexpectedStr, string(resp.Result))
			}
		}
	}
	return nil
}
