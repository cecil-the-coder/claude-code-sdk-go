package shared

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Test control request types and discriminated union pattern through SDKControlRequest
func TestControlRequestTypes(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantType string
		validate func(*testing.T, ControlRequest)
	}{
		{
			name:     "initialize request",
			jsonData: `{"type":"sdk_control_request","request_id":"req_1_test","request":{"type":"initialize","hooks":["can_use_tool"],"callbacks":[{"id":"cb1","name":"test_callback"}]}}`,
			wantType: "initialize",
			validate: func(t *testing.T, req ControlRequest) {
				t.Helper()
				initReq, ok := req.(*InitializeRequest)
				if !ok {
					t.Fatalf("expected *InitializeRequest, got %T", req)
				}
				if len(initReq.Hooks) != 1 || initReq.Hooks[0] != "can_use_tool" {
					t.Errorf("hooks = %v, want [can_use_tool]", initReq.Hooks)
				}
				if len(initReq.Callbacks) != 1 || initReq.Callbacks[0].ID != "cb1" {
					t.Errorf("callbacks incorrect")
				}
			},
		},
		{
			name:     "can_use_tool request",
			jsonData: `{"type":"sdk_control_request","request_id":"req_2_test","request":{"type":"can_use_tool","tool_name":"Read","input":{"file_path":"/test.txt"},"suggestions":["allow"]}}`,
			wantType: "can_use_tool",
			validate: func(t *testing.T, req ControlRequest) {
				t.Helper()
				canUseReq, ok := req.(*CanUseToolRequest)
				if !ok {
					t.Fatalf("expected *CanUseToolRequest, got %T", req)
				}
				if canUseReq.ToolName != "Read" {
					t.Errorf("tool_name = %s, want Read", canUseReq.ToolName)
				}
				if canUseReq.Input["file_path"] != "/test.txt" {
					t.Errorf("input incorrect")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sdkReq SDKControlRequest
			err := json.Unmarshal([]byte(tt.jsonData), &sdkReq)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if sdkReq.Request.RequestType() != tt.wantType {
				t.Errorf("RequestType() = %s, want %s", sdkReq.Request.RequestType(), tt.wantType)
			}

			tt.validate(t, sdkReq.Request)
		})
	}
}

// Test request ID generation
func TestGenerateRequestID(t *testing.T) {
	t.Run("format validation", func(t *testing.T) {
		id := GenerateRequestID()

		// Format: req_{counter}_{8-char-hex}
		if !strings.HasPrefix(id, "req_") {
			t.Errorf("ID does not start with 'req_': %s", id)
		}

		parts := strings.Split(id, "_")
		if len(parts) != 3 {
			t.Errorf("ID should have 3 parts separated by '_', got %d: %s", len(parts), id)
		}

		// Check hex part is 8 characters
		hexPart := parts[2]
		if len(hexPart) != 8 {
			t.Errorf("hex part should be 8 characters, got %d: %s", len(hexPart), hexPart)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		ids := make(map[string]bool)
		count := 1000

		for i := 0; i < count; i++ {
			id := GenerateRequestID()
			if ids[id] {
				t.Errorf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}

		if len(ids) != count {
			t.Errorf("expected %d unique IDs, got %d", count, len(ids))
		}
	})
}

// Test PendingControlResponses manager
func TestPendingControlResponses(t *testing.T) {
	t.Run("register and wait success", func(t *testing.T) {
		mgr := NewPendingControlResponses()
		requestID := "req_test_1"

		// Register request
		respChan := mgr.Register(requestID, 1*time.Second)

		// Simulate response in goroutine
		go func() {
			time.Sleep(10 * time.Millisecond)
			response := &ControlResponseData{
				Success: true,
				Result:  map[string]interface{}{"data": "test"},
			}
			mgr.Resolve(requestID, response)
		}()

		// Wait for response
		response, err := mgr.Wait(requestID, respChan)
		if err != nil {
			t.Fatalf("Wait error: %v", err)
		}

		respData, ok := response.(*ControlResponseData)
		if !ok {
			t.Fatalf("response is not *ControlResponseData")
		}

		if !respData.Success {
			t.Error("expected success response")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		mgr := NewPendingControlResponses()
		requestID := "req_test_timeout"

		// Register with short timeout
		respChan := mgr.Register(requestID, 50*time.Millisecond)

		// Wait for timeout
		_, err := mgr.Wait(requestID, respChan)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}

		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("expected timeout error, got: %v", err)
		}
	})
}
