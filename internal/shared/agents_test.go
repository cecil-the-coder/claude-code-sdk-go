package shared

import (
	"encoding/json"
	"testing"
)

// TestAgentDefinitionBasicValidation tests basic AgentDefinition struct creation and validation
func TestAgentDefinitionBasicValidation(t *testing.T) {
	tests := []struct {
		name        string
		description string
		prompt      string
		tools       []string
		model       *string
		wantValid   bool
	}{
		{
			name:        "valid_minimal_agent",
			description: "A test agent",
			prompt:      "You are a test agent",
			tools:       nil,
			model:       nil,
			wantValid:   true,
		},
		{
			name:        "valid_with_tools",
			description: "Code reviewer",
			prompt:      "Review code for best practices",
			tools:       []string{"Read", "Grep"},
			model:       nil,
			wantValid:   true,
		},
		{
			name:        "valid_with_model_sonnet",
			description: "Documentation writer",
			prompt:      "Write comprehensive documentation",
			tools:       []string{"Read", "Write", "Edit"},
			model:       agentStringPtr("sonnet"),
			wantValid:   true,
		},
		{
			name:        "valid_with_model_opus",
			description: "Advanced analyzer",
			prompt:      "Analyze complex code patterns",
			tools:       []string{"Read", "Grep", "Glob"},
			model:       agentStringPtr("opus"),
			wantValid:   true,
		},
		{
			name:        "valid_with_model_haiku",
			description: "Quick responder",
			prompt:      "Provide quick answers",
			tools:       []string{"Read"},
			model:       agentStringPtr("haiku"),
			wantValid:   true,
		},
		{
			name:        "valid_with_model_inherit",
			description: "Inheriting agent",
			prompt:      "Use parent model",
			tools:       []string{"Read"},
			model:       agentStringPtr("inherit"),
			wantValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentDefinition{
				Description: tt.description,
				Prompt:      tt.prompt,
				Tools:       tt.tools,
				Model:       tt.model,
			}

			// Basic validation - all agents should be valid in this test
			if !tt.wantValid {
				t.Error("Expected agent to be invalid, but structure was created")
			}

			// Verify fields are set correctly
			if agent.Description != tt.description {
				t.Errorf("Description = %q, want %q", agent.Description, tt.description)
			}
			if agent.Prompt != tt.prompt {
				t.Errorf("Prompt = %q, want %q", agent.Prompt, tt.prompt)
			}
		})
	}
}

// TestAgentDefinitionJSONMarshaling tests JSON marshaling with nil field filtering
func TestAgentDefinitionJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		agent    *AgentDefinition
		wantJSON string
	}{
		{
			name: "minimal_agent_filters_nil_fields",
			agent: &AgentDefinition{
				Description: "Test agent",
				Prompt:      "You are a test agent",
				Tools:       nil,
				Model:       nil,
			},
			wantJSON: `{"description":"Test agent","prompt":"You are a test agent"}`,
		},
		{
			name: "agent_with_tools",
			agent: &AgentDefinition{
				Description: "Code reviewer",
				Prompt:      "Review code",
				Tools:       []string{"Read", "Grep"},
				Model:       nil,
			},
			wantJSON: `{"description":"Code reviewer","prompt":"Review code","tools":["Read","Grep"]}`,
		},
		{
			name: "agent_with_model",
			agent: &AgentDefinition{
				Description: "Doc writer",
				Prompt:      "Write docs",
				Tools:       []string{"Write"},
				Model:       agentStringPtr("sonnet"),
			},
			wantJSON: `{"description":"Doc writer","prompt":"Write docs","tools":["Write"],"model":"sonnet"}`,
		},
		{
			name: "agent_with_empty_tools_array",
			agent: &AgentDefinition{
				Description: "Empty tools",
				Prompt:      "Test empty tools",
				Tools:       []string{},
				Model:       nil,
			},
			wantJSON: `{"description":"Empty tools","prompt":"Test empty tools","tools":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJSON, err := json.Marshal(tt.agent)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Parse both to compare structure, not string
			var got, want map[string]interface{}
			if err := json.Unmarshal(gotJSON, &got); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &want); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			// Compare field by field (order-independent)
			if !mapsEqual(got, want) {
				t.Errorf("json.Marshal() = %s, want %s", gotJSON, tt.wantJSON)
			}

			// Test roundtrip
			var roundtrip AgentDefinition
			if err := json.Unmarshal(gotJSON, &roundtrip); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if roundtrip.Description != tt.agent.Description {
				t.Errorf("Roundtrip Description = %q, want %q", roundtrip.Description, tt.agent.Description)
			}
			if roundtrip.Prompt != tt.agent.Prompt {
				t.Errorf("Roundtrip Prompt = %q, want %q", roundtrip.Prompt, tt.agent.Prompt)
			}
		})
	}
}

// TestAgentsMapMarshaling tests marshaling map[string]AgentDefinition with multiple agents
func TestAgentsMapMarshaling(t *testing.T) {
	agents := map[string]AgentDefinition{
		"code-reviewer": {
			Description: "Reviews code for best practices",
			Prompt:      "You are a code reviewer",
			Tools:       []string{"Read", "Grep"},
			Model:       agentStringPtr("sonnet"),
		},
		"doc-writer": {
			Description: "Writes documentation",
			Prompt:      "You are a documentation expert",
			Tools:       []string{"Read", "Write", "Edit"},
			Model:       nil,
		},
	}

	jsonData, err := json.Marshal(agents)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal to verify structure
	var roundtrip map[string]AgentDefinition
	if err := json.Unmarshal(jsonData, &roundtrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(roundtrip) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(roundtrip))
	}

	// Verify code-reviewer
	codeReviewer, ok := roundtrip["code-reviewer"]
	if !ok {
		t.Fatal("code-reviewer agent not found")
	}
	if codeReviewer.Description != "Reviews code for best practices" {
		t.Errorf("code-reviewer Description = %q", codeReviewer.Description)
	}
	if len(codeReviewer.Tools) != 2 {
		t.Errorf("code-reviewer Tools length = %d, want 2", len(codeReviewer.Tools))
	}

	// Verify doc-writer
	docWriter, ok := roundtrip["doc-writer"]
	if !ok {
		t.Fatal("doc-writer agent not found")
	}
	if docWriter.Model != nil {
		t.Errorf("doc-writer Model = %v, want nil", docWriter.Model)
	}
}

// TestOptionsWithAgents tests Options struct with Agents field
func TestOptionsWithAgents(t *testing.T) {
	opts := &Options{
		Agents: map[string]AgentDefinition{
			"test-agent": {
				Description: "Test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"Read"},
				Model:       agentStringPtr("sonnet"),
			},
		},
	}

	// Test JSON marshaling of Options with agents
	jsonData, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify agents field is present in JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := result["agents"]; !ok {
		t.Error("Expected 'agents' field in JSON output")
	}
}

// TestAgentDefinitionNilFieldFiltering tests that nil fields are properly filtered
func TestAgentDefinitionNilFieldFiltering(t *testing.T) {
	agent := &AgentDefinition{
		Description: "Test",
		Prompt:      "Test prompt",
		Tools:       nil,
		Model:       nil,
	}

	jsonData, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Parse back to check what fields are present
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Nil fields should not be present in JSON
	if _, ok := parsed["tools"]; ok {
		t.Error("Expected nil 'tools' field to be omitted from JSON")
	}
	if _, ok := parsed["model"]; ok {
		t.Error("Expected nil 'model' field to be omitted from JSON")
	}

	// Required fields should be present
	if _, ok := parsed["description"]; !ok {
		t.Error("Expected 'description' field to be present in JSON")
	}
	if _, ok := parsed["prompt"]; !ok {
		t.Error("Expected 'prompt' field to be present in JSON")
	}
}

// Helper function for agent tests
func agentStringPtr(s string) *string {
	return &s
}

// Helper to compare maps (order-independent)
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		// Handle nested structures if needed
		switch av := v.(type) {
		case []interface{}:
			bvSlice, ok := bv.([]interface{})
			if !ok || len(av) != len(bvSlice) {
				return false
			}
			for i, item := range av {
				if item != bvSlice[i] {
					return false
				}
			}
		default:
			if v != bv {
				return false
			}
		}
	}
	return true
}
