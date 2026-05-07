// Package profile defines audience profiles for the Engram MCP server.
//
// A profile controls three things:
//  1. ServerInstructions — the instruction string injected into the MCP client.
//  2. AllowedTypes — the observation types recommended for this audience. A nil
//     slice means no restriction (use the "all" profile for open access).
//  3. ToolDescriptions — per-tool description overrides so tool vocabulary matches
//     the audience's mental model (e.g. "memories" instead of "bugs/decisions").
//
// Built-in profiles: "dev", "mind", "all".
// Default when no --profile flag is given: "dev" (backward compatible).
package profile

import "fmt"

// Profile holds the runtime configuration for a single named audience profile.
type Profile struct {
	// Name is the canonical CLI name (e.g. "dev", "mind", "all").
	Name string

	// AllowedTypes lists the observation types recommended for this profile.
	// nil means no restriction — all types are accepted without warnings.
	AllowedTypes []string

	// ServerInstructions is the instruction string passed to the MCP client via
	// server.WithInstructions(). Each profile can present a different framing
	// to guide agent behavior.
	ServerInstructions string

	// ToolDescriptions overrides the description string for specific tools.
	// Key = tool name (e.g. "mem_save"), value = replacement description.
	// Only tools whose UX meaningfully changes between profiles need entries.
	ToolDescriptions map[string]string

	// SetupTargets lists the setup agent targets that produce a config for this
	// profile (e.g. ["claude-desktop"] for mind).
	SetupTargets []string
}

// HasType reports whether typ is in the profile's AllowedTypes.
// If AllowedTypes is nil (profile "all"), it always returns true.
func (p *Profile) HasType(typ string) bool {
	if p.AllowedTypes == nil {
		return true
	}
	for _, t := range p.AllowedTypes {
		if t == typ {
			return true
		}
	}
	return false
}

// devServerInstructions is copied verbatim from the const serverInstructions in
// internal/mcp/mcp.go. It MUST remain identical to preserve backward compatibility
// when --profile=dev (the default) is active.
const devServerInstructions = `Engram provides persistent memory that survives across sessions and compactions.

CORE TOOLS (always available — use without ToolSearch):
  mem_save — save decisions, bugs, discoveries, conventions PROACTIVELY (do not wait to be asked)
  mem_search — find past work, decisions, or context from previous sessions
  mem_context — get recent session history (call at session start or after compaction)
  mem_session_summary — save end-of-session summary (MANDATORY before saying "done")
  mem_get_observation — get full untruncated content of a search result by ID
  mem_save_prompt — save user prompt for context
  mem_current_project — detect current project from cwd (recommended first call)

DEFERRED TOOLS (use ToolSearch when needed):
  mem_update, mem_suggest_topic_key, mem_session_start, mem_session_end,
  mem_stats, mem_delete, mem_timeline, mem_capture_passive, mem_merge_projects

PROACTIVE SAVE RULE: Call mem_save immediately after ANY decision, bug fix, discovery, or convention — not just when asked.

## CONFLICT SURFACING — when mem_save returns candidates

After every mem_save call, check the response envelope for judgment_required.

IF judgment_required IS TRUE:
  Iterate candidates[] and call mem_judge once per candidate using the
  judgment_id from that candidate entry. Do NOT use the top-level judgment_id
  for multiple candidates — each candidate has its own.

  HEURISTIC — when to ask the user vs. resolve autonomously:

  ASK the user (conversationally, as part of your next reply) when:
    - confidence is below 0.7, OR
    - the relation you would choose is supersedes or conflicts_with AND the
      observation type is architecture, policy, or decision

  RESOLVE silently (call mem_judge without asking) when:
    - confidence >= 0.7 AND the relation is not supersedes/conflicts_with, OR
    - the relation is related, compatible, scoped, or not_conflict

  HOW TO ASK (conversationally — never via blocking CLI or dashboard prompt):
    Raise it naturally in your next reply to the user. Example phrasing:
    "I noticed memory #abc123 might conflict with what we just saved.
     Want me to mark the new one as superseding it, or are they about
     different scopes? I can also mark them as compatible if both still apply."

  AFTER RESOLUTION (both paths):
    Call mem_judge with the chosen relation, a reason, and if the user gave
    explicit direction, include their words as the evidence field. This persists
    the verdict and closes the pending conflict row.`

// Dev is the default profile for AI coding agents working on software projects.
// Its AllowedTypes match the standard software engineering observation taxonomy.
// Its ServerInstructions are identical to the previous hardcoded const — this is
// the backward-compatible default when no --profile flag is provided.
var Dev = Profile{
	Name: "dev",
	AllowedTypes: []string{
		"bugfix",
		"architecture",
		"decision",
		"pattern",
		"config",
		"discovery",
		"learning",
		"manual",
		"passive",
		"session_summary",
		"policy",
		"preference",
	},
	ServerInstructions: devServerInstructions,
	ToolDescriptions:   nil,
	SetupTargets:       nil,
}

// Mind is a profile for personal knowledge management.
// AllowedTypes is nil — no restriction. The type field is free-form so Claude
// can organically assign whatever type fits the memory (idea, goal, meeting,
// recipe, person, etc.). The value is in the serverInstructions, not in type
// enforcement.
var Mind = Profile{
	Name:         "mind",
	AllowedTypes: nil,
	ServerInstructions: `Engram is your personal mind — persistent memory that survives across conversations.

You remember EVERYTHING the user shares: ideas, goals, people, decisions, learnings, plans, reflections, references, meetings, habits — anything worth recalling later.

CORE TOOLS (always available):
  mem_save — capture memories PROACTIVELY (do not wait to be asked)
  mem_search — recall past memories, ideas, or context from previous conversations
  mem_context — get recent history (call at conversation start)
  mem_session_summary — save a summary before ending (MANDATORY)
  mem_get_observation — read full content of a memory by ID
  mem_save_prompt — save user prompt for context
  mem_current_project — detect current knowledge area

DEFERRED TOOLS (use ToolSearch when needed):
  mem_update, mem_suggest_topic_key, mem_session_start, mem_session_end,
  mem_stats, mem_delete, mem_timeline, mem_capture_passive, mem_merge_projects

## PROACTIVE CAPTURE — this is your primary job

Save IMMEDIATELY after any of these, without being asked:
- An idea, plan, or intention expressed by the user
- A decision made about anything (career, health, finance, projects, relationships)
- Something learned or discovered
- A person mentioned with context (who they are, relationship, key details)
- A goal set or updated
- A meeting, conversation, or event worth remembering
- A reference to a book, article, tool, place, or resource
- A reflection, opinion, or change of perspective
- A habit, routine, or system the user is building
- Any fact the user would be frustrated to forget

Use type freely — pick whatever fits: idea, goal, decision, person, meeting, reference, learning, reflection, habit, plan, project, bookmark, journal, or any descriptive word.

FORMAT for content:
  **What**: [one sentence — what happened or was decided]
  **Context**: [when, where, who, why — whatever is relevant]
  **Takeaway**: [the insight or action item — omit if obvious]

TITLE should be short, specific, and searchable: "Goal: run a half marathon by December", "Alice recommended Sapiens", "Decided to switch to morning workouts".

## RECALL — always search before saying you don't know

When the user asks about something from a past conversation, a person, a decision, or any prior context: call mem_search FIRST. Your memory is more reliable than your training data for user-specific information.

## CONFLICT SURFACING

After every mem_save, check the response for judgment_required.

IF judgment_required IS TRUE:
  Check each candidate. If confidence >= 0.7 and the relation is not
  supersedes/conflicts_with, resolve silently via mem_judge. Otherwise,
  ask the user conversationally: "I found a related memory — should the
  new one replace it, or do both still apply?"

## SESSION CLOSE

Before ending any conversation, call mem_session_summary with what was discussed, what was decided, and what's next. This is NOT optional.`,
	ToolDescriptions: map[string]string{
		"mem_save": `Save a memory to your personal mind. Call this PROACTIVELY whenever something worth remembering comes up — ideas, decisions, goals, people, learnings, references, reflections, plans, or anything the user would want to recall later.

TITLE: short and searchable (e.g. "Goal: learn Go by March", "Meeting with Ana about the house").
TYPE: any descriptive word that fits (idea, goal, decision, person, meeting, reference, learning, reflection, habit, plan, project, bookmark, journal).
CONTENT: structured as **What** / **Context** / **Takeaway**.`,
		"mem_search": `Search your personal mind to recall past memories, ideas, people, decisions, or anything saved in previous conversations.

ALWAYS search before saying you don't remember. Your memory knows more about the user than your training data.`,
	},
	SetupTargets: []string{"claude-desktop"},
}

// All is an unrestricted profile with no type filtering. It uses the same
// server instructions as Dev and never emits type_warning on save. Useful for
// power users who work across multiple domains and do not want any guidance friction.
var All = Profile{
	Name:               "all",
	AllowedTypes:       nil, // nil = no restriction
	ServerInstructions: devServerInstructions,
	ToolDescriptions:   nil,
	SetupTargets:       nil,
}

// Get returns the built-in profile for the given name.
// Valid names are "dev", "mind", and "all".
// Returns a non-nil error for any other name.
func Get(name string) (*Profile, error) {
	switch name {
	case "dev":
		p := Dev
		return &p, nil
	case "mind":
		p := Mind
		return &p, nil
	case "all":
		p := All
		return &p, nil
	default:
		return nil, fmt.Errorf("unknown profile %q — valid profiles: dev, mind, all", name)
	}
}

// List returns the names of all built-in profiles.
func List() []string {
	return []string{"dev", "mind", "all"}
}
