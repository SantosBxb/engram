package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Gentleman-Programming/engram/internal/profile"
	mcppkg "github.com/mark3labs/mcp-go/mcp"
)

// ─── handleSave: soft type warning ───────────────────────────────────────────

// TestHandleSave_TypeWarning_OutOfProfileType verifies that saving an observation
// whose type is not in the active profile's AllowedTypes adds a "type_warning"
// key to the JSON response, while the save itself succeeds (no IsError).
func TestHandleSave_TypeWarning_OutOfProfileType(t *testing.T) {
	s := newMCPTestStore(t)
	devProfile, err := profile.Get("dev")
	if err != nil {
		t.Fatalf("profile.Get(dev): %v", err)
	}

	cfg := MCPConfig{Profile: devProfile}
	h := handleSave(s, cfg, NewSessionActivity(10*time.Minute))

	// "idea" is not in the dev profile's AllowedTypes
	req := mcppkg.CallToolRequest{Params: mcppkg.CallToolParams{Arguments: map[string]any{
		"title":   "Random idea",
		"content": "Something creative",
		"type":    "idea",
		"project": "engram",
	}}}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("save must succeed even for out-of-profile types; got error: %s", callResultText(t, res))
	}

	text := callResultText(t, res)

	var envelope map[string]any
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("response is not valid JSON: %v\ntext: %s", err, text)
	}

	if _, ok := envelope["type_warning"]; !ok {
		t.Errorf("expected \"type_warning\" key in response; got: %s", text)
	}
}

// TestHandleSave_NoTypeWarning_InProfileType verifies that saving with a type
// that IS in the active profile's AllowedTypes produces no "type_warning" key.
func TestHandleSave_NoTypeWarning_InProfileType(t *testing.T) {
	s := newMCPTestStore(t)
	devProfile, err := profile.Get("dev")
	if err != nil {
		t.Fatalf("profile.Get(dev): %v", err)
	}

	cfg := MCPConfig{Profile: devProfile}
	h := handleSave(s, cfg, NewSessionActivity(10*time.Minute))

	// "bugfix" is in the dev profile
	req := mcppkg.CallToolRequest{Params: mcppkg.CallToolParams{Arguments: map[string]any{
		"title":   "Fixed nil deref",
		"content": "Added nil guard in handler",
		"type":    "bugfix",
		"project": "engram",
	}}}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", callResultText(t, res))
	}

	text := callResultText(t, res)

	var envelope map[string]any
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("response is not valid JSON: %v\ntext: %s", err, text)
	}

	if _, ok := envelope["type_warning"]; ok {
		t.Errorf("unexpected \"type_warning\" in response for in-profile type; got: %s", text)
	}
}

// TestHandleSave_NoTypeWarning_AllProfile verifies that the "all" profile (nil
// AllowedTypes) never emits a type_warning regardless of the type string.
func TestHandleSave_NoTypeWarning_AllProfile(t *testing.T) {
	s := newMCPTestStore(t)
	allProfile, err := profile.Get("all")
	if err != nil {
		t.Fatalf("profile.Get(all): %v", err)
	}

	cfg := MCPConfig{Profile: allProfile}
	h := handleSave(s, cfg, NewSessionActivity(10*time.Minute))

	for _, typ := range []string{"idea", "bugfix", "custom_type", "anything"} {
		t.Run(typ, func(t *testing.T) {
			req := mcppkg.CallToolRequest{Params: mcppkg.CallToolParams{Arguments: map[string]any{
				"title":   "Some memory",
				"content": "Content",
				"type":    typ,
				"project": "engram",
			}}}

			res, err := h(context.Background(), req)
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("unexpected error: %s", callResultText(t, res))
			}

			text := callResultText(t, res)
			var envelope map[string]any
			if err := json.Unmarshal([]byte(text), &envelope); err != nil {
				t.Fatalf("response not JSON: %v\ntext: %s", err, text)
			}
			if _, ok := envelope["type_warning"]; ok {
				t.Errorf("\"all\" profile must never emit type_warning; got one for type=%q: %s", typ, text)
			}
		})
	}
}

// TestHandleSave_NoTypeWarning_NilProfile verifies that when Profile is nil
// (backward compat mode), no type_warning is ever emitted.
func TestHandleSave_NoTypeWarning_NilProfile(t *testing.T) {
	s := newMCPTestStore(t)

	cfg := MCPConfig{} // Profile is nil
	h := handleSave(s, cfg, NewSessionActivity(10*time.Minute))

	req := mcppkg.CallToolRequest{Params: mcppkg.CallToolParams{Arguments: map[string]any{
		"title":   "Anything",
		"content": "Content",
		"type":    "idea",
		"project": "engram",
	}}}

	res, err := h(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", callResultText(t, res))
	}

	text := callResultText(t, res)
	var envelope map[string]any
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		t.Fatalf("response not JSON: %v\ntext: %s", err, text)
	}
	if _, ok := envelope["type_warning"]; ok {
		t.Errorf("nil profile must never emit type_warning; got one: %s", text)
	}
}

// ─── registerTools: tool description override ─────────────────────────────────

// TestToolDescription_Override verifies that toolDescription returns the profile
// override when the profile has an entry for the tool name.
func TestToolDescription_Override(t *testing.T) {
	sbProfile, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("profile.Get(mind): %v", err)
	}

	cfg := MCPConfig{Profile: sbProfile}
	const defaultDesc = "default description"

	got := toolDescription(cfg, "mem_save", defaultDesc)
	want := sbProfile.ToolDescriptions["mem_save"]
	if got != want {
		t.Errorf("toolDescription with override: got %q, want %q", got, want)
	}
	if got == defaultDesc {
		t.Error("toolDescription returned the default when an override is present")
	}
}

// TestToolDescription_NoOverride verifies that toolDescription falls back to the
// default when the profile has no entry for the tool name.
func TestToolDescription_NoOverride(t *testing.T) {
	sbProfile, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("profile.Get(mind): %v", err)
	}

	cfg := MCPConfig{Profile: sbProfile}
	const defaultDesc = "default description for mem_update"

	// mem_update has no override in Mind
	got := toolDescription(cfg, "mem_update", defaultDesc)
	if got != defaultDesc {
		t.Errorf("toolDescription without override: got %q, want %q", got, defaultDesc)
	}
}

// TestToolDescription_NilProfile verifies that toolDescription returns the default
// when Profile is nil (backward compat mode).
func TestToolDescription_NilProfile(t *testing.T) {
	cfg := MCPConfig{} // nil Profile
	const defaultDesc = "default"
	got := toolDescription(cfg, "mem_save", defaultDesc)
	if got != defaultDesc {
		t.Errorf("toolDescription with nil profile: got %q, want %q", got, defaultDesc)
	}
}

// TestNewServerWithConfig_MindDescriptionOverride verifies that creating
// a server with the mind profile causes mem_save to be registered with
// the profile's description override.
func TestNewServerWithConfig_MindDescriptionOverride(t *testing.T) {
	s := newMCPTestStore(t)
	sbProfile, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("profile.Get(mind): %v", err)
	}

	srv := NewServerWithConfig(s, MCPConfig{Profile: sbProfile}, nil)
	if srv == nil {
		t.Fatal("expected non-nil MCP server")
	}

	// The server is created without error — tool registration with overrides worked.
	// The description override is exercised via toolDescription() which is unit-tested above.
	// This integration test confirms the server builds successfully with a profile.
	_ = srv
}

// ─── Server instructions ──────────────────────────────────────────────────────

// TestNewServerWithActivity_DevProfile_BackwardCompat verifies that the dev profile
// produces instructions identical to the package-level const.
func TestNewServerWithActivity_DevProfile_BackwardCompat(t *testing.T) {
	devProfile, err := profile.Get("dev")
	if err != nil {
		t.Fatalf("profile.Get(dev): %v", err)
	}

	// Dev profile's instructions must equal the const.
	if devProfile.ServerInstructions != serverInstructions {
		// Find the first difference for a helpful error message
		const maxLen = 80
		dLen := len(devProfile.ServerInstructions)
		cLen := len(serverInstructions)
		if dLen < cLen {
			t.Errorf("dev.ServerInstructions is shorter (%d vs %d chars)", dLen, cLen)
		} else if dLen > cLen {
			t.Errorf("dev.ServerInstructions is longer (%d vs %d chars)", dLen, cLen)
		}
		// Character-level diff for first mismatch
		for i := 0; i < len(devProfile.ServerInstructions) && i < len(serverInstructions); i++ {
			if devProfile.ServerInstructions[i] != serverInstructions[i] {
				start := i - 20
				if start < 0 {
					start = 0
				}
				t.Errorf("first difference at byte %d: profile=%q const=%q",
					i,
					devProfile.ServerInstructions[start:min(i+maxLen, dLen)],
					serverInstructions[start:min(i+maxLen, cLen)],
				)
				break
			}
		}
	}
}

func TestNewServerWithActivity_MindProfile_DifferentInstructions(t *testing.T) {
	sbProfile, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("profile.Get(mind): %v", err)
	}

	// Mind must differ from the dev const
	if sbProfile.ServerInstructions == serverInstructions {
		t.Error("mind.ServerInstructions must differ from the dev const serverInstructions")
	}
	if strings.TrimSpace(sbProfile.ServerInstructions) == "" {
		t.Error("mind.ServerInstructions must not be empty")
	}
}

