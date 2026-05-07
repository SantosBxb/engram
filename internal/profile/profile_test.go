package profile_test

import (
	"strings"
	"testing"

	"github.com/Gentleman-Programming/engram/internal/profile"
)

// ─── Get() ───────────────────────────────────────────────────────────────────

func TestGet_KnownProfiles(t *testing.T) {
	cases := []struct {
		name           string
		wantNil        bool
		wantAllowedNil bool
	}{
		{"dev", false, false},
		{"mind", false, true}, // unrestricted — captures any type
		{"all", false, true},          // unrestricted
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := profile.Get(tc.name)
			if err != nil {
				t.Fatalf("Get(%q) returned unexpected error: %v", tc.name, err)
			}
			if p == nil {
				t.Fatalf("Get(%q) returned nil profile", tc.name)
			}
			if p.Name != tc.name {
				t.Errorf("Get(%q).Name = %q, want %q", tc.name, p.Name, tc.name)
			}
			if tc.wantAllowedNil && p.AllowedTypes != nil {
				t.Errorf("Get(%q).AllowedTypes = %v, want nil", tc.name, p.AllowedTypes)
			}
			if !tc.wantAllowedNil && p.AllowedTypes == nil {
				t.Errorf("Get(%q).AllowedTypes is nil, want non-nil slice", tc.name)
			}
		})
	}
}

func TestGet_UnknownProfile(t *testing.T) {
	p, err := profile.Get("nonexistent")
	if err == nil {
		t.Fatalf("Get(%q) expected error, got nil (profile: %+v)", "nonexistent", p)
	}
	if p != nil {
		t.Errorf("Get(%q) returned non-nil profile on error: %+v", "nonexistent", p)
	}
	// Error message should mention the invalid name
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error message %q should mention the unknown profile name", err.Error())
	}
	// Error message should list valid profiles
	for _, valid := range []string{"dev", "mind", "all"} {
		if !strings.Contains(err.Error(), valid) {
			t.Errorf("error message %q should list valid profile %q", err.Error(), valid)
		}
	}
}

// ─── HasType() ───────────────────────────────────────────────────────────────

func TestHasType_DevProfile(t *testing.T) {
	dev, err := profile.Get("dev")
	if err != nil {
		t.Fatalf("Get(dev): %v", err)
	}

	inProfile := []string{"bugfix", "architecture", "decision", "pattern", "config", "discovery",
		"learning", "manual", "passive", "session_summary", "policy", "preference"}
	for _, typ := range inProfile {
		if !dev.HasType(typ) {
			t.Errorf("dev.HasType(%q) = false, want true", typ)
		}
	}

	notInProfile := []string{"idea", "reflection", "goal", "meeting", "journal", "bookmark"}
	for _, typ := range notInProfile {
		if dev.HasType(typ) {
			t.Errorf("dev.HasType(%q) = true, want false", typ)
		}
	}
}

func TestHasType_MindProfile_Unrestricted(t *testing.T) {
	sb, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("Get(mind): %v", err)
	}

	// mind has AllowedTypes == nil — HasType must always return true
	for _, typ := range []string{"idea", "goal", "bugfix", "architecture", "anything", "custom"} {
		if !sb.HasType(typ) {
			t.Errorf("mind.HasType(%q) = false, want true (AllowedTypes is nil)", typ)
		}
	}
}

func TestHasType_AllProfile_AlwaysTrue(t *testing.T) {
	all, err := profile.Get("all")
	if err != nil {
		t.Fatalf("Get(all): %v", err)
	}

	// "all" profile has nil AllowedTypes — HasType must always return true
	for _, typ := range []string{"bugfix", "idea", "anything", "custom_type", ""} {
		if !all.HasType(typ) {
			t.Errorf("all.HasType(%q) = false, want true (AllowedTypes is nil)", typ)
		}
	}
}

// ─── List() ──────────────────────────────────────────────────────────────────

func TestList_ReturnsAllProfiles(t *testing.T) {
	names := profile.List()

	wantNames := map[string]bool{"dev": true, "mind": true, "all": true}
	if len(names) != len(wantNames) {
		t.Errorf("List() returned %d names, want %d: %v", len(names), len(wantNames), names)
	}

	for _, name := range names {
		if !wantNames[name] {
			t.Errorf("List() contains unexpected profile name %q", name)
		}
	}

	// Every name returned by List() must be retrievable via Get()
	for _, name := range names {
		if _, err := profile.Get(name); err != nil {
			t.Errorf("List() returned %q but Get(%q) failed: %v", name, name, err)
		}
	}
}

// ─── Dev backward compatibility ───────────────────────────────────────────────

func TestDev_ServerInstructionsNotEmpty(t *testing.T) {
	dev, err := profile.Get("dev")
	if err != nil {
		t.Fatalf("Get(dev): %v", err)
	}
	if strings.TrimSpace(dev.ServerInstructions) == "" {
		t.Error("dev.ServerInstructions is empty — it must contain the current server instructions")
	}
	// Spot-check that key phrases from the original const are present
	for _, phrase := range []string{"mem_save", "mem_search", "PROACTIVE SAVE RULE"} {
		if !strings.Contains(dev.ServerInstructions, phrase) {
			t.Errorf("dev.ServerInstructions missing expected phrase %q", phrase)
		}
	}
}

func TestMind_HasToolDescriptions(t *testing.T) {
	sb, err := profile.Get("mind")
	if err != nil {
		t.Fatalf("Get(mind): %v", err)
	}
	for _, tool := range []string{"mem_save", "mem_search"} {
		desc, ok := sb.ToolDescriptions[tool]
		if !ok {
			t.Errorf("mind.ToolDescriptions missing key %q", tool)
			continue
		}
		if strings.TrimSpace(desc) == "" {
			t.Errorf("mind.ToolDescriptions[%q] is empty", tool)
		}
	}
}

// ─── Get() returns independent copies ────────────────────────────────────────

func TestGet_ReturnsCopy(t *testing.T) {
	// Mutating the returned profile should not affect the package-level var.
	p1, _ := profile.Get("dev")
	p1.Name = "mutated"

	p2, _ := profile.Get("dev")
	if p2.Name != "dev" {
		t.Errorf("Get(dev) returned same pointer — mutation affected package var: Name=%q", p2.Name)
	}
}
