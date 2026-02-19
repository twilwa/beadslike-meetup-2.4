# Work Plan: `tl` MVR — Minimum Viable Replacement for beads

## TL;DR

> **Quick Summary**: Build a Go CLI (`tl`) with append-only JSONL storage and replay-on-read architecture (no snapshot file), matching beads behavioral parity for the 8 daily-use commands, with import/export and bd aliases for seamless drop-in replacement.
>
> **Deliverables**:
> - Single-binary Go CLI with 12 commands (create, list, show, update, close, reopen, ready, claim, blocked, stats, dep add/remove)
> - Beads JSONL import/export with content-hash deduplication
> - Sync command (export + git add)
> - `bd` command aliases (dispatch wrapper with fallthrough to real bd)
> - E2E smoke test covering full workflow
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 4 waves + final verification
> **Critical Path**: T2 → T7 → T11 → T12 → T15 → T17 → T18

---

## Context

### Original Request
Build a streamlined replacement for beads (`bd`) for daily task management. User found beads "too bulky" and wants lighter overhead while maintaining compatibility with beads task data. Initially considered DB-backed storage (Dolt/HelixDB/SQLite) but after analysis agreed JSONL + in-memory replay provides sufficient query flexibility at task-tracker scale.

### Interview Summary
**Key Decisions**:
- Go as implementation language (matches beads, single binary)
- Append-only JSONL + replay-on-read (no snapshot file, no binary deps)
- flock(2) for concurrency via golang.org/x/sys/unix
- NO Store interface for MVR — package-internal functions (ergo pattern)
- MVR scope: 18 tasks to replace bd in daily workflow
- Momus review before execution

**Research Findings**:
- Ergo validates append-only JSONL + flock + replay-on-read at this scale (<5ms for <1000 tasks)
- Ergo has NO Store interface — package functions work fine
- Beads AffectsReadyWork includes conditional-blocks and waits-for beyond basic blocks
- Content-hash dedup essential for multi-clone import correctness
- Beads status model includes pinned/hooked (may have been removed from CLI in v0.36+ but status values still round-trip)
- bd alias surface is narrow: 7-8 commands in CLAUDE.md/AGENTS.md

### Metis Review
**Identified Gaps (addressed)**:
- Store interface is premature for MVR → removed, use package functions
- Snapshot file creates staleness bugs → removed, replay-on-read like ergo
- `bd prime`/hooks/doctor not in MVR → bd alias fallthrough to real bd binary
- Actor resolution unspecified → env var TL_ACTOR > git user.name > "unknown"
- Content-hash field set may not match actual beads export → executor validates, hashes whatever exists
- ID prefix on import → keep original prefix (bd-xxx stays bd-xxx)
- Orphaned dep targets → store permissively (cross-repo reference)
- Truncated events.jsonl → skip incomplete final line (crash recovery)

---

## Work Objectives

### Core Objective
Deliver a production-ready CLI tool that replaces beads for the 8 daily-use command patterns documented in CLAUDE.md and AGENTS.md, with zero data loss on import/export.

### Concrete Deliverables
- `tl` binary with 12+ commands (init, create, list, show, update, close, reopen, ready, claim, blocked, stats, dep, import, export, sync)
- `.tl/` directory structure with events.jsonl + lock file
- Beads import: read `.beads/issues.jsonl` with content-hash dedup
- Beads export: write `.beads/issues.jsonl` format
- `bd` dispatch wrapper (alias for daily commands, fallthrough for unknown)
- E2E test covering full agent workflow

### Definition of Done
- [ ] All daily-use bd commands work through tl (create, list, show, update, close, ready, claim, blocked, stats, dep add, sync)
- [ ] `tl ready` output matches beads ready semantics (status-aware, dep-aware, priority-sorted)
- [ ] `tl claim` is race-safe (concurrent test: 2 claimers → 1 winner)
- [ ] Round-trip: beads export → tl import → tl export → field-by-field comparison = zero loss
- [ ] Unknown fields/statuses/dep-types from beads round-trip without loss
- [ ] All tests pass, zero vet warnings

### Must Have
- Behavioral parity for ready/claim/dep/status per PRD Section 5
- Content-hash deduplication on import
- Dependency cycle detection (DFS)
- Blocked task computation (blocks, parent-child transitive, conditional-blocks, waits-for)
- `--json` on all commands with stable schema
- Unknown field preservation via metadata passthrough
- flock(2) serialized writes
- bd alias dispatch for CLAUDE.md command patterns

### Must NOT Have (Guardrails)
- No Store interface / abstraction layer (package functions only)
- No snapshot.json file (replay-on-read only)
- No daemon process
- No color, emoji, box-drawing, or table formatting
- No interactive prompts or wizard flows
- No logging framework (fmt.Fprintf to stderr for debug only)
- No shell completion generation
- No doctor/config/hooks/prime commands
- No dep tree visualization
- No migration phase machinery (shadow-read/dual-write/cutover)
- No `tl sync` git commit/push/pull (export + git add only)
- No `bd list` advanced flags beyond --status/--type/--assignee/--priority/--json/--limit
- No `bd show` display modes (--watch/--thread/--children)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO (new project)
- **Automated tests**: YES (TDD)
- **Framework**: Go stdlib `testing` + `testify` for assertions
- **TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task includes agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **CLI**: Use Bash — Run command, assert exit code + JSON output
- **Concurrency**: Use Bash — Parallel processes, assert exactly one winner
- **Import/Export**: Use Bash — Round-trip diff, assert zero field-level differences
- **Deps**: Use Bash — Cycle detection, transitive blocking assertions

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — 6 tasks, ALL independent, start immediately):
├── T1: Go module scaffold + cobra CLI skeleton [quick]
├── T2: Core types — Issue, Dependency, Status, DependencyType, Priority [quick]
├── T3: Event types + JSON serialization [quick]
├── T4: Content hash utility [quick]
├── T5: File lock manager (flock via golang.org/x/sys/unix) [quick]
└── T6: Beads JSONL parser [quick]

Wave 2 (Storage + CRUD — 5 tasks, T7 first, then T8-T11 parallel):
├── T7: Event store — append + replay to in-memory graph [deep] (depends: T2, T3, T5)
├── T8: Init + Create commands [unspecified-high] (depends: T1, T2, T7)
├── T9: List + Show commands [unspecified-high] (depends: T1, T2, T7)
├── T10: Update + Close + Reopen commands [unspecified-high] (depends: T1, T2, T7)
└── T11: Dependency add/remove + DFS cycle detection [deep] (depends: T2, T7)

Wave 3 (Ready/Claim + Import/Export — 4 tasks, ALL parallel):
├── T12: Blocked cache + Ready + Blocked + Stats commands [deep] (depends: T7, T11)
├── T13: Beads import with content-hash dedup [deep] (depends: T4, T6, T7)
├── T14: Beads export [unspecified-high] (depends: T2, T7)
└── T15: Atomic claim command [deep] (depends: T5, T7)

Wave 4 (Integration — 3 tasks, mostly sequential):
├── T16: Sync command (export + git add) [quick] (depends: T14)
├── T17: bd command aliases — dispatch wrapper [unspecified-high] (depends: T8-T16)
└── T18: E2E smoke test — full agent workflow [deep] (depends: all)

Wave FINAL (Independent review, 4 parallel):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real QA — full CLI workflow (unspecified-high)
└── F4: Scope fidelity check (deep)

Critical Path: T2 → T7 → T11 → T12 → T17 → T18 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 6 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| T1 | — | T8-T10 | 1 |
| T2 | — | T7-T15 | 1 |
| T3 | — | T7 | 1 |
| T4 | — | T13 | 1 |
| T5 | — | T7, T15 | 1 |
| T6 | — | T13 | 1 |
| T7 | T2, T3, T5 | T8-T15 | 2 |
| T8 | T1, T2, T7 | T17 | 2 |
| T9 | T1, T2, T7 | T17 | 2 |
| T10 | T1, T2, T7 | T17 | 2 |
| T11 | T2, T7 | T12, T17 | 2 |
| T12 | T7, T11 | T17 | 3 |
| T13 | T4, T6, T7 | T17 | 3 |
| T14 | T2, T7 | T16, T17 | 3 |
| T15 | T5, T7 | T17 | 3 |
| T16 | T14 | T17 | 4 |
| T17 | T8-T16 | T18 | 4 |
| T18 | all | F1-F4 | 4 |

### Agent Dispatch Summary

- **Wave 1**: 6 tasks — T1-T6 → `quick`
- **Wave 2**: 5 tasks — T7 → `deep`, T8-T10 → `unspecified-high`, T11 → `deep`
- **Wave 3**: 4 tasks — T12 → `deep`, T13 → `deep`, T14 → `unspecified-high`, T15 → `deep`
- **Wave 4**: 3 tasks — T16 → `quick`, T17 → `unspecified-high`, T18 → `deep`
- **FINAL**: 4 tasks — F1 → `oracle`, F2-F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [ ] 1. Go module scaffold + cobra CLI skeleton

  **What to do**:
  - Initialize Go module: `go mod init github.com/<user>/tl` (use appropriate module path)
  - Add dependencies: `github.com/spf13/cobra`, `github.com/stretchr/testify`, `golang.org/x/sys`
  - Create project structure following ergo pattern:
    - `cmd/tl/main.go` — entry point, calls `cli.Execute()`
    - `internal/tl/cli.go` — root cobra command with `--json` global flag, `--dir` override flag
    - `internal/tl/` — all package-internal logic (NO exported API)
  - Root command: `tl` with version, help text (one line per command)
  - Subcommands registered as stubs (empty RunE returning "not implemented"): init, create, list, show, update, close, reopen, ready, claim, blocked, stats, dep (with add/remove subcommands), import, export, sync
  - Global flags: `--json` (bool), `--dir` (string, override .tl/ location)
  - Write a test: `go test ./...` passes, `tl --help` shows all subcommands

  **Must NOT do**:
  - No shell completion generation
  - No cobra.GenBashCompletion
  - No verbose help text — one line per command max
  - No color, no emoji

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2, T3, T4, T5, T6)
  - **Blocks**: T8, T9, T10
  - **Blocked By**: None

  **References**:
  - `docs/references/ergo/cmd/ergo/main.go` — Entry point pattern (single main.go calling internal)
  - `docs/references/ergo/cmd/ergo/cmd_root.go` — Cobra root command setup with global options
  - `docs/references/ergo/internal/ergo/cli.go` — CLI execution flow, command registration
  - `docs/references/ergo/go.mod` — Dependency versions (cobra v1.10.2, golang.org/x/sys v0.40.0)
  - Follow ergo's pattern: `cmd/<name>/` has main.go + command wiring, `internal/<name>/` has all logic

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/tl` produces a binary
  - [ ] `go test ./... -count=1` passes
  - [ ] `go vet ./...` reports no issues
  - [ ] `./tl --help` lists all subcommands
  - [ ] `./tl create` returns "not implemented" error (stub works)
  - [ ] `./tl --json create` flag is parsed (no error about unknown flag)

  **QA Scenarios**:
  ```
  Scenario: CLI builds and runs
    Tool: Bash
    Steps:
      1. go build -o ./tl ./cmd/tl
      2. ./tl --help
      3. Assert exit code 0
      4. Assert output contains "create", "list", "show", "ready", "claim", "dep"
    Expected Result: All subcommands listed
    Evidence: .sisyphus/evidence/task-1-cli-help.txt

  Scenario: Stub command returns not-implemented
    Tool: Bash
    Steps:
      1. ./tl create 2>&1; echo "EXIT:$?"
      2. Assert output contains "not implemented"
      3. Assert exit code is non-zero
    Expected Result: Clean error, non-zero exit
    Evidence: .sisyphus/evidence/task-1-stub-error.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): scaffold Go module with cobra CLI skeleton`
  - Files: `cmd/tl/`, `internal/tl/`, `go.mod`, `go.sum`

- [ ] 2. Core types — Issue, Dependency, Status, DependencyType, Priority

  **What to do**:
  - Create `internal/tl/model.go` with core domain types matching beads behavioral parity matrix
  - `Status` type (string): open, in_progress, blocked, deferred, closed, pinned, hooked
    - `IsValid()` method for known statuses
    - Accept unknown status values without error (round-trip safety)
  - `DependencyType` type (string): blocks, parent-child, conditional-blocks, waits-for, related, discovered-from
    - `AffectsReadyWork()` method: true for blocks, parent-child, conditional-blocks, waits-for
    - Accept unknown dep types without error (round-trip safety)
  - `IssueType` type (string): bug, feature, task, epic, chore, decision (accept unknown types)
  - `Priority` as int (0 = P0/critical, no omitempty — 0 is valid)
  - `Issue` struct with beads-compatible fields:
    - Core: ID, Title, Description, Design, AcceptanceCriteria, Notes, SpecID
    - Workflow: Status, Priority, IssueType
    - Assignment: Assignee, Owner, CreatedBy
    - Timestamps: CreatedAt, UpdatedAt, ClosedAt (*time.Time), CloseReason
    - Scheduling: DeferUntil (*time.Time)
    - Labels: []string
    - Dependencies: []*Dependency
    - Metadata: map[string]json.RawMessage (unknown fields passthrough)
    - Pinned, Ephemeral bools
  - `Dependency` struct: IssueID, DependsOnID, Type (DependencyType), CreatedAt, CreatedBy, Metadata (json.RawMessage)
  - `Graph` struct: Tasks map[string]*Issue, Deps/RDeps adjacency maps
  - State transition validation: `validateTransition(from, to Status) error`
  - Write comprehensive tests for all type methods

  **Must NOT do**:
  - No agent/molecule/gate/slot/formula/HOP fields — those go into Metadata passthrough
  - No import/export logic — just type definitions
  - No Store interface

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3, T4, T5, T6)
  - **Blocks**: T7, T8, T9, T10, T11, T12, T13, T14, T15
  - **Blocked By**: None

  **References**:
  - `docs/references/beads/internal/types/types.go:15-127` — Issue struct with ALL fields (copy field names/JSON tags for core fields, remaining go to Metadata)
  - `docs/references/beads/internal/types/types.go:385-405` — Status enum: open, in_progress, blocked, deferred, closed, pinned, hooked
  - `docs/references/beads/internal/types/types.go:672-734` — DependencyType enum, AffectsReadyWork() method — MUST match this logic exactly
  - `docs/references/beads/internal/types/types.go:424-434` — IssueType enum
  - `docs/references/ergo/internal/ergo/model.go:16-96` — Ergo's simpler state machine and validation pattern — follow this code style
  - PRD Section 5.1-5.2 (`docs/PRD-streamlined-task-tool-variants.md:50-82`) — Behavioral parity matrix for statuses and dep types

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tl/ -run TestStatus -count=1` passes
  - [ ] `Status("open").IsValid()` returns true
  - [ ] `Status("custom_thing").IsValid()` returns false but doesn't error
  - [ ] `DependencyType("blocks").AffectsReadyWork()` returns true
  - [ ] `DependencyType("related").AffectsReadyWork()` returns false
  - [ ] `Issue{}` struct serializes to JSON with `priority: 0` (not omitted)
  - [ ] `validateTransition("open", "in_progress")` returns nil
  - [ ] `validateTransition("closed", "in_progress")` returns error

  **QA Scenarios**:
  ```
  Scenario: Status round-trip for unknown values
    Tool: Bash
    Steps:
      1. go test -run TestStatusRoundTrip -v ./internal/tl/
      2. Test creates Issue with Status("custom_workflow"), marshals to JSON, unmarshals back
      3. Assert Status is preserved as "custom_workflow"
    Expected Result: Unknown status round-trips without error
    Evidence: .sisyphus/evidence/task-2-status-roundtrip.txt

  Scenario: Priority zero is not omitted
    Tool: Bash
    Steps:
      1. go test -run TestPriorityZero -v ./internal/tl/
      2. Marshal Issue{Priority: 0} to JSON
      3. Assert JSON contains "priority":0
    Expected Result: Priority 0 present in JSON output
    Evidence: .sisyphus/evidence/task-2-priority-zero.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): core types — Issue, Dependency, Status, DependencyType`
  - Files: `internal/tl/model.go`, `internal/tl/model_test.go`

- [ ] 3. Event types + JSON serialization

  **What to do**:
  - Create `internal/tl/events.go` with event type definitions
  - Event struct: Type (string), ID (string), Timestamp (time.Time), Actor (string), Data (json.RawMessage)
  - Event types: "create", "update", "close", "reopen", "dep_add", "dep_remove", "claim"
  - Each event type has a typed data struct (CreateEventData, UpdateEventData, etc.)
  - Actor resolution: env var `TL_ACTOR` → `git config user.name` → "unknown"
    - `resolveActor() string` function
  - JSON serialization: one JSON object per line, no pretty-printing
  - `newEvent(eventType string, data interface{}) (Event, error)` constructor that sets timestamp + actor
  - Write tests for serialization round-trip (marshal → unmarshal → compare)

  **Must NOT do**:
  - No event replay logic (that's T7)
  - No file I/O (that's T7)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T4, T5, T6)
  - **Blocks**: T7
  - **Blocked By**: None

  **References**:
  - `docs/references/ergo/internal/ergo/graph.go:16-61` — Event replay switch statement shows what event types to support and their data shapes
  - `docs/references/ergo/internal/ergo/model.go` — Event struct definition, newEvent constructor pattern
  - PRD Section 6.1 (`docs/PRD-streamlined-task-tool-variants.md:131-142`) — Event format specification

  **Acceptance Criteria**:
  - [ ] `go test -run TestEventSerialization -count=1 ./internal/tl/` passes
  - [ ] Event marshals to single-line JSON (no newlines within)
  - [ ] Event round-trips: marshal → unmarshal → deep equal
  - [ ] Actor resolution reads TL_ACTOR env var when set

  **QA Scenarios**:
  ```
  Scenario: Event serialization produces valid single-line JSON
    Tool: Bash
    Steps:
      1. go test -run TestEventJSON -v ./internal/tl/
      2. Create event, marshal to JSON
      3. Assert result is single line (no embedded newlines)
      4. Assert result parses as valid JSON
    Expected Result: Single-line JSON, parseable
    Evidence: .sisyphus/evidence/task-3-event-json.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): event types and JSON serialization`
  - Files: `internal/tl/events.go`, `internal/tl/events_test.go`

- [ ] 4. Content hash utility

  **What to do**:
  - Create `internal/tl/hash.go` with content hash computation
  - `ComputeContentHash(issue *Issue) string` — SHA256 hex digest
  - Hash fields in stable order matching beads: title, description, design, acceptance_criteria, notes, spec_id, status, priority, issue_type, assignee, owner, created_by
  - Use null-byte separator between fields (same as beads `hashFieldWriter` pattern)
  - For fields that don't exist in a given issue, hash empty string + separator (stable output)
  - Include metadata in hash (serialize to JSON, then hash)
  - Write tests comparing hash output against known beads hash values if possible

  **Must NOT do**:
  - No import logic — just the hash function
  - No beads-specific field handling beyond what's in Issue struct

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T5, T6)
  - **Blocks**: T13
  - **Blocked By**: None

  **References**:
  - `docs/references/beads/internal/types/types.go:129-210` — ComputeContentHash implementation (MUST match field order and null-separator pattern for import dedup to work)
  - `docs/references/beads/internal/types/types.go:212-261` — hashFieldWriter helper (null-byte separator pattern)

  **Acceptance Criteria**:
  - [ ] `go test -run TestContentHash -count=1 ./internal/tl/` passes
  - [ ] Same issue content produces same hash every time (deterministic)
  - [ ] Different content produces different hash
  - [ ] Hash includes null-byte separators between fields

  **QA Scenarios**:
  ```
  Scenario: Deterministic hash output
    Tool: Bash
    Steps:
      1. go test -run TestHashDeterministic -v ./internal/tl/
      2. Compute hash of same issue twice
      3. Assert hashes are identical
    Expected Result: Identical hashes for identical content
    Evidence: .sisyphus/evidence/task-4-hash-deterministic.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): content hash utility for import dedup`
  - Files: `internal/tl/hash.go`, `internal/tl/hash_test.go`

- [ ] 5. File lock manager (flock via golang.org/x/sys/unix)

  **What to do**:
  - Create `internal/tl/lock.go` with file-lock serialization
  - `withLock(lockPath string, fn func() error) error` — acquire exclusive flock, run fn, release
  - Use `golang.org/x/sys/unix.Flock()` with `unix.LOCK_EX | unix.LOCK_NB` (non-blocking)
  - On lock contention: return structured error with message "lock busy, retry" (fail-fast)
  - Lock file: `.tl/lock` (created if not exists)
  - Write concurrency test: two goroutines race for lock, exactly one gets it immediately, other gets error

  **Must NOT do**:
  - No retry logic — fail fast, let caller retry
  - No timeout — non-blocking only
  - No daemon or long-held locks

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4, T6)
  - **Blocks**: T7, T15
  - **Blocked By**: None

  **References**:
  - `docs/references/ergo/internal/ergo/storage.go:253-298` — `writeLinkEvent` shows the `withLock(lockPath, syscall.LOCK_EX, func() error {...})` pattern: lock → loadGraph → validate → append
  - `docs/references/ergo/internal/ergo/storage.go:192-209` — `appendEvents` under lock
  - Ergo uses `syscall.Flock` (deprecated) — use `golang.org/x/sys/unix.Flock` instead (same semantics, modern package)

  **Acceptance Criteria**:
  - [ ] `go test -run TestLock -count=1 ./internal/tl/` passes
  - [ ] `withLock` acquires and releases lock correctly
  - [ ] Concurrent lock attempt returns error (not deadlock)
  - [ ] Lock file is created if it doesn't exist

  **QA Scenarios**:
  ```
  Scenario: Lock contention returns error
    Tool: Bash
    Steps:
      1. go test -run TestLockContention -v ./internal/tl/
      2. Two goroutines call withLock simultaneously
      3. Assert exactly one succeeds, one returns ErrLockBusy
    Expected Result: One winner, one error, no deadlock
    Evidence: .sisyphus/evidence/task-5-lock-contention.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): file lock manager via flock(2)`
  - Files: `internal/tl/lock.go`, `internal/tl/lock_test.go`

- [ ] 6. Beads JSONL parser

  **What to do**:
  - Create `internal/tl/beads_parser.go` — read beads `.beads/issues.jsonl` format
  - Parse one JSON object per line (beads format: complete issue object with nested deps)
  - Map beads fields to tl Issue struct:
    - Known fields: id, title, description, design, acceptance_criteria, notes, spec_id, status, priority, issue_type, assignee, owner, created_by, created_at, updated_at, closed_at, close_reason, defer_until, labels, dependencies, pinned, ephemeral
    - Unknown fields: capture into Issue.Metadata as json.RawMessage (agent fields, molecule fields, gate fields, etc.)
  - Handle `dependencies` as nested array within each issue (not separate lines)
  - Tolerate truncated final line (crash recovery — follow ergo pattern)
  - `ParseBeadsJSONL(reader io.Reader) ([]*Issue, error)` — returns slice of issues
  - Write tests with sample JSONL input containing known + unknown fields
  - **Create test fixture**: `internal/tl/testdata/beads_sample.jsonl` — a representative beads JSONL file containing:
    - 3-5 issues with varying statuses, priorities, types
    - Nested dependencies (blocks, parent-child, related)
    - Unknown beads-specific fields (hook_bead, mol_type, agent_state, etc.) to verify metadata passthrough
    - This fixture is used by T13 (import), T14 (export round-trip), T18 (E2E)

  **Must NOT do**:
  - No import dedup logic (that's T13)
  - No writing/export (that's T14)
  - No event generation — just parsing into Issue structs

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4, T5)
  - **Blocks**: T13
  - **Blocked By**: None

  **References**:
  - `docs/references/beads/internal/types/types.go:15-127` — Full Issue struct with ALL fields and JSON tags. Known fields map directly; everything else → Metadata
  - `docs/references/beads/internal/types/types.go:672-708` — Dependency struct and all DependencyType values
  - `docs/references/ergo/internal/ergo/storage.go:109-178` — `readEvents` with truncated-line tolerance. Follow this crash-recovery pattern.
  - PRD Section 6.2 (`docs/PRD-streamlined-task-tool-variants.md:144-172`) — Beads JSONL import format with nested deps example
  - `internal/tl/testdata/beads_sample.jsonl` — Test fixture created in T6 with representative beads data (including unknown fields for metadata passthrough testing)

  **Acceptance Criteria**:
  - [ ] `go test -run TestBeadsParser -count=1 ./internal/tl/` passes
  - [ ] Parses beads JSONL with all known fields correctly
  - [ ] Unknown fields preserved in Metadata (not silently dropped)
  - [ ] Nested dependencies parsed correctly
  - [ ] Truncated final line tolerated (returns partial results, no error)

  **QA Scenarios**:
  ```
  Scenario: Parse beads JSONL with unknown fields
    Tool: Bash
    Steps:
      1. go test -run TestBeadsParserUnknownFields -v ./internal/tl/
      2. Parse JSONL containing {"hook_bead":"x","mol_type":"swarm",...}
      3. Assert hook_bead and mol_type appear in Metadata
    Expected Result: Unknown fields in Metadata, not lost
    Evidence: .sisyphus/evidence/task-6-unknown-fields.txt

  Scenario: Truncated line tolerance
    Tool: Bash
    Steps:
      1. go test -run TestBeadsParserTruncated -v ./internal/tl/
      2. Parse JSONL where last line is incomplete (no closing brace)
      3. Assert all complete lines parsed, no error
    Expected Result: Partial results returned, incomplete line skipped
    Evidence: .sisyphus/evidence/task-6-truncated.txt
  ```

  **Commit**: YES (group with Wave 1)
  - Message: `feat(tl): beads JSONL parser with unknown field preservation`
  - Files: `internal/tl/beads_parser.go`, `internal/tl/beads_parser_test.go`

- [ ] 7. Event store — append events + replay to in-memory graph

  **What to do**:
  - Create `internal/tl/store.go` — the core storage layer (package-internal, NO interface)
  - `.tl/` directory structure: `events.jsonl` (append-only event log), `lock` (flock file)
  - `initDir(path string) error` — create `.tl/` with empty events.jsonl and lock file
  - `resolveTLDir(start string) (string, error)` — walk up directory tree looking for `.tl/` (follow ergo pattern)
  - `loadGraph(dir string) (*Graph, error)` — replay all events to build in-memory Graph
    - Parse events.jsonl line by line
    - Switch on event type: create, update, close, reopen, dep_add, dep_remove, claim
    - Build Graph.Tasks map and Graph.Deps/RDeps adjacency maps
  - `appendEvents(path string, events []Event) error` — append JSON lines to events.jsonl
  - `withLock` integration: all mutations must call `withLock(lockPath, func() error { loadGraph → validate → appendEvents })`
  - ID generation: `tl-<4hex>` format (hash-based, 4-char hex). For child tasks: `tl-<4hex>.N`
    - `generateID() string` — crypto/rand based
  - No snapshot file — replay on every read (ergo pattern, fast at <1000 tasks)
  - Tolerate truncated final line in events.jsonl

  **Must NOT do**:
  - No snapshot.json file
  - No compaction logic
  - No Store interface — all functions are package-internal

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (foundational — T8-T11 depend on this)
  - **Parallel Group**: Wave 2 (runs first, then T8-T11 parallel)
  - **Blocks**: T8, T9, T10, T11, T12, T13, T14, T15
  - **Blocked By**: T2, T3, T5

  **References**:
  - `docs/references/ergo/internal/ergo/storage.go:100-107` — `loadGraph()` pattern: get events path → readEvents → replayEvents
  - `docs/references/ergo/internal/ergo/storage.go:109-178` — `readEvents()` with line-by-line parsing, truncation tolerance
  - `docs/references/ergo/internal/ergo/storage.go:192-209` — `appendEvents()` — O_APPEND | O_CREATE | O_WRONLY, marshal + newline
  - `docs/references/ergo/internal/ergo/storage.go:253-298` — `writeLinkEvent()` — canonical lock → load → validate → append pattern
  - `docs/references/ergo/internal/ergo/graph.go:16-61` — `replayEvents()` switch statement — build tasks map from events
  - `docs/references/ergo/internal/ergo/storage.go:28-62` — `resolveErgoDir()` — walk up to find `.ergo/`

  **Acceptance Criteria**:
  - [ ] `go test -run TestStore -count=1 ./internal/tl/` passes
  - [ ] initDir creates `.tl/events.jsonl` and `.tl/lock`
  - [ ] appendEvents writes valid JSONL lines
  - [ ] loadGraph replays events correctly (create → task in graph)
  - [ ] Mutations happen under flock (withLock wrapping)
  - [ ] resolveTLDir walks up directory tree

  **QA Scenarios**:
  ```
  Scenario: Create and replay
    Tool: Bash
    Steps:
      1. go test -run TestStoreCreateReplay -v ./internal/tl/
      2. Init dir, append create event, loadGraph
      3. Assert task exists in graph with correct fields
    Expected Result: Event replay reconstructs graph state
    Evidence: .sisyphus/evidence/task-7-create-replay.txt

  Scenario: Multiple events replay correctly
    Tool: Bash
    Steps:
      1. go test -run TestStoreMultiEvent -v ./internal/tl/
      2. Append create → update → close events for same ID
      3. loadGraph, assert final state is closed
    Expected Result: Last event wins for each field
    Evidence: .sisyphus/evidence/task-7-multi-event.txt
  ```

  **Commit**: YES (group with Wave 2)
  - Message: `feat(tl): event store — append-only JSONL with replay`
  - Files: `internal/tl/store.go`, `internal/tl/store_test.go`

- [ ] 8. Init + Create commands

  **What to do**:
  - Implement `tl init` command in `internal/tl/cmd_init.go`:
    - Creates `.tl/` directory with empty events.jsonl + lock file
    - If `.tl/` already exists, return error "already initialized"
    - Output: `{"initialized": true, "path": ".tl/"}` with --json, or "Initialized .tl/" text
  - Implement `tl create` command in `internal/tl/cmd_create.go`:
    - Flags: `--title` (required), `--type` (default "task"), `--priority` (default 2), `--description`, `--json`
    - Under lock: generate ID → create event → append → output
    - Output: full Issue JSON with --json, or "Created <id>: <title>" text
    - Positional arg support: `tl create "My task title"` (title as first positional arg, like bd)
  - Wire both commands into cobra root command (replace stubs from T1)
  - Write tests for both commands using t.TempDir()

  **Must NOT do**:
  - No --dry-run, --deps, --spec-id, --estimate, --ephemeral flags
  - No interactive prompts
  - No wizard flow

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after T7)
  - **Parallel Group**: Wave 2 (with T9, T10, T11)
  - **Blocks**: T17
  - **Blocked By**: T1, T2, T7

  **References**:
  - `docs/references/ergo/internal/ergo/storage.go:301-318` — `createTask()` — lock → loadGraph → validate epic → generate ID → append event pattern
  - `docs/references/ergo/cmd/ergo/cmd_actions.go` — Command wiring pattern for cobra RunE functions
  - `docs/references/beads/.github/copilot-instructions.md` — bd create usage: `bd create "Title" --description="..." -t task -p 2 --json`

  **Acceptance Criteria**:
  - [ ] `tl init` creates `.tl/` directory
  - [ ] `tl init` in already-initialized dir returns error
  - [ ] `tl create --title "Test" --json` returns valid JSON with id, title, status=open, priority
  - [ ] `tl create "Test"` works (positional arg for title)
  - [ ] Created task has `tl-` prefix ID
  - [ ] events.jsonl contains the create event after running

  **QA Scenarios**:
  ```
  Scenario: Init and create flow
    Tool: Bash
    Preconditions: Empty temp directory
    Steps:
      1. cd to temp dir
      2. ./tl init
      3. ./tl create --title "First task" --type task --priority 1 --json
      4. Assert exit 0
      5. Assert JSON output has "id" starting with "tl-"
      6. Assert JSON output has "status":"open"
      7. cat .tl/events.jsonl | wc -l → 1 line
    Expected Result: Init + create works, event persisted
    Evidence: .sisyphus/evidence/task-8-init-create.txt

  Scenario: Double init fails
    Tool: Bash
    Steps:
      1. ./tl init (first time — success)
      2. ./tl init (second time)
      3. Assert exit code non-zero
      4. Assert error message contains "already initialized"
    Expected Result: Second init rejected
    Evidence: .sisyphus/evidence/task-8-double-init.txt
  ```

  **Commit**: YES (group with Wave 2)
  - Message: `feat(tl): init and create commands`
  - Files: `internal/tl/cmd_init.go`, `internal/tl/cmd_create.go`, tests

- [ ] 9. List + Show commands

  **What to do**:
  - Implement `tl list` in `internal/tl/cmd_list.go`:
    - Load graph, filter tasks, output
    - Filter flags: `--status` (string), `--type` (string), `--assignee` (string), `--priority` (int), `--limit` (int, default 0=all)
    - Sort: by priority (ascending, 0=P0 first), then created_at
    - `--json`: output JSON array of issues
    - Text output: one line per issue — `<id> [<status>] P<priority> <title>`
  - Implement `tl show <id>` in `internal/tl/cmd_show.go`:
    - Load graph, find task by ID, output full detail
    - `--json`: output single Issue JSON
    - Text output: field-per-line (id, title, status, priority, description, etc.)
    - Include dependencies in output (both blocking and blocked-by)
    - Error if ID not found

  **Must NOT do**:
  - No --watch, --thread, --children, --short, --full flags
  - No color or emoji in output
  - No table formatting

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after T7)
  - **Parallel Group**: Wave 2 (with T8, T10, T11)
  - **Blocks**: T17
  - **Blocked By**: T1, T2, T7

  **References**:
  - `docs/references/ergo/internal/ergo/output.go` — Output formatting patterns (text vs JSON mode)
  - `docs/references/ergo/internal/ergo/text.go` — Text output rendering for list/show
  - `docs/references/beads/.github/copilot-instructions.md` — `bd list --status open --priority 1 --json`, `bd show <id> --json`

  **Acceptance Criteria**:
  - [ ] `tl list --json` returns JSON array (empty array when no tasks)
  - [ ] `tl list --status open --json` filters correctly
  - [ ] `tl list --priority 0 --json` shows only P0 tasks
  - [ ] `tl show <id> --json` returns full issue JSON
  - [ ] `tl show nonexistent` returns error with non-zero exit

  **QA Scenarios**:
  ```
  Scenario: List with status filter
    Tool: Bash
    Steps:
      1. tl init && tl create --title "Open" --json && tl create --title "Closed" --json
      2. Close the second task
      3. tl list --status open --json
      4. Assert result contains "Open", not "Closed"
    Expected Result: Filter works correctly
    Evidence: .sisyphus/evidence/task-9-list-filter.txt

  Scenario: Show nonexistent returns error
    Tool: Bash
    Steps:
      1. tl show tl-0000 2>&1
      2. Assert exit code non-zero
      3. Assert output contains "not found"
    Expected Result: Clean error for missing ID
    Evidence: .sisyphus/evidence/task-9-show-notfound.txt
  ```

  **Commit**: YES (group with Wave 2)
  - Message: `feat(tl): list and show commands with filters`
  - Files: `internal/tl/cmd_list.go`, `internal/tl/cmd_show.go`, tests

- [ ] 10. Update + Close + Reopen commands

  **What to do**:
  - Implement `tl update <id>` in `internal/tl/cmd_update.go`:
    - Flags: `--status`, `--title`, `--description`, `--priority`, `--assignee`, `--type`, `--json`
    - Under lock: loadGraph → find task → validate transition (if --status) → append update event
    - Only changed fields go into the update event data
  - Implement `tl close <id>` in `internal/tl/cmd_close.go`:
    - Flags: `--reason` (string), `--json`
    - Sets status=closed, closed_at=now, close_reason=<reason>
    - Validate transition from current status to closed
  - Implement `tl reopen <id>` in same file or `cmd_reopen.go`:
    - Sets status=open, clears closed_at
    - Validate transition from closed/deferred → open

  **Must NOT do**:
  - No --claim flag on update (claim is a separate command T15)
  - No interactive editor ($EDITOR)
  - No --design, --notes, --acceptance flags in MVR (store in metadata if imported)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after T7)
  - **Parallel Group**: Wave 2 (with T8, T9, T11)
  - **Blocks**: T17
  - **Blocked By**: T1, T2, T7

  **References**:
  - `docs/references/ergo/internal/ergo/model.go:73-96` — `validateTransition()` state machine — follow this pattern for status validation
  - `docs/references/ergo/internal/ergo/graph.go:62-80` — State event replay (state change + claim clearing logic)
  - `docs/references/beads/.github/copilot-instructions.md` — `bd update <id> --status in_progress --json`, `bd close <id> --reason "Done" --json`

  **Acceptance Criteria**:
  - [ ] `tl update <id> --status in_progress --json` transitions open → in_progress
  - [ ] `tl update <id> --status blocked` from closed returns validation error
  - [ ] `tl close <id> --reason "done" --json` sets status=closed, close_reason="done"
  - [ ] `tl reopen <id> --json` sets status=open, clears closed_at
  - [ ] events.jsonl contains update/close/reopen events

  **QA Scenarios**:
  ```
  Scenario: Update-close-reopen lifecycle
    Tool: Bash
    Steps:
      1. tl create --title "Test" --json → capture id
      2. tl update $id --status in_progress --json → assert status=in_progress
      3. tl close $id --reason "finished" --json → assert status=closed
      4. tl reopen $id --json → assert status=open
      5. tl show $id --json → verify final state
    Expected Result: Full lifecycle works
    Evidence: .sisyphus/evidence/task-10-lifecycle.txt

  Scenario: Invalid transition rejected
    Tool: Bash
    Steps:
      1. tl create --title "Test" --json → capture id
      2. tl close $id
      3. tl update $id --status blocked 2>&1
      4. Assert exit code non-zero
      5. Assert error mentions "invalid transition"
    Expected Result: Closed → blocked rejected
    Evidence: .sisyphus/evidence/task-10-invalid-transition.txt
  ```

  **Commit**: YES (group with Wave 2)
  - Message: `feat(tl): update, close, and reopen commands`
  - Files: `internal/tl/cmd_update.go`, `internal/tl/cmd_close.go`, tests

- [ ] 11. Dependency add/remove + DFS cycle detection

  **What to do**:
  - Implement `tl dep add <issue-id> <depends-on-id>` in `internal/tl/cmd_dep.go`:
    - Flags: `--type` (default "blocks"), `--json`
    - Under lock: loadGraph → validate both IDs exist → cycle check → append dep_add event
    - Cycle detection: DFS from depends-on-id following all blocking dep types — if it reaches issue-id, reject
  - Implement `tl dep remove <issue-id> <depends-on-id>`:
    - Under lock: loadGraph → validate dep exists → append dep_remove event
  - `hasCycle(graph *Graph, from, to string) bool` — DFS cycle detection
    - Walk forward deps from `to`; if `from` is reachable, there's a cycle
  - Dep types accepted: any string (unknown types stored, only known types affect readiness)
  - Graph adjacency: Deps[issueID] = set of {dependsOnID}, RDeps[dependsOnID] = set of {issueID}

  **Must NOT do**:
  - No dep tree visualization
  - No cross-repo dep resolution

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after T7)
  - **Parallel Group**: Wave 2 (with T8, T9, T10)
  - **Blocks**: T12, T17
  - **Blocked By**: T2, T7

  **References**:
  - `docs/references/ergo/internal/ergo/storage.go:253-298` — `writeLinkEvent()` — lock → loadGraph → validateDepSelf → validateDepKinds → hasCycle → append. EXACT pattern to follow.
  - `docs/references/ergo/internal/ergo/graph.go` — `hasCycle()` DFS implementation
  - `docs/references/beads/internal/types/types.go:672-734` — DependencyType enum and AffectsReadyWork() — accept all types, only known ones affect readiness

  **Acceptance Criteria**:
  - [ ] `tl dep add A B --type blocks --json` creates dependency
  - [ ] `tl dep add A B` then `tl dep add B A` → cycle detected error
  - [ ] `tl dep remove A B --json` removes dependency
  - [ ] Unknown dep types accepted without error (e.g., `--type custom-link`)
  - [ ] Self-dependency rejected (A depends on A)

  **QA Scenarios**:
  ```
  Scenario: Cycle detection
    Tool: Bash
    Steps:
      1. tl create --title "A" --json → $A
      2. tl create --title "B" --json → $B
      3. tl dep add $B $A --type blocks → success
      4. tl dep add $A $B --type blocks 2>&1 → assert error
      5. Assert error contains "cycle"
    Expected Result: Cycle detected and rejected
    Evidence: .sisyphus/evidence/task-11-cycle.txt

  Scenario: Unknown dep type accepted
    Tool: Bash
    Steps:
      1. tl create --title "X" --json → $X
      2. tl create --title "Y" --json → $Y
      3. tl dep add $Y $X --type custom-workflow --json
      4. Assert exit 0
      5. tl show $Y --json → dep type is "custom-workflow"
    Expected Result: Unknown types stored without error
    Evidence: .sisyphus/evidence/task-11-unknown-dep.txt
  ```

  **Commit**: YES (group with Wave 2)
  - Message: `feat(tl): dependency CRUD with DFS cycle detection`
  - Files: `internal/tl/cmd_dep.go`, `internal/tl/cycle.go`, tests

- [ ] 12. Blocked cache + Ready + Blocked + Stats commands

  **What to do**:
  - **Blocked cache** in `internal/tl/ready.go`:
    - `computeBlockedSet(graph *Graph) map[string]bool` — compute which tasks are blocked
    - A task is blocked if ANY of these are true:
      1. Has `blocks` dep pointing to an open/in_progress/blocked task
      2. Has `parent-child` dep where parent is blocked (TRANSITIVE — recurse up parent chain)
      3. Has `conditional-blocks` dep pointing to an open/in_progress/blocked task
      4. Has `waits-for` dep pointing to an open/in_progress/blocked task
    - Only dep types where `AffectsReadyWork()` returns true participate
    - Cache recomputed on every loadGraph (replay-on-read, so this is fine)
  - **Ready command** in `internal/tl/cmd_ready.go`:
    - Task is ready if ALL conditions met (per PRD Section 5.3):
      1. Status is `open` OR `in_progress` (both are workable states)
      2. Not in blocked set
      3. DeferUntil is nil or in the past
      4. Not pinned, not deferred
    - Sort by priority ascending (0 first), then created_at ascending
    - `--json`: JSON array of ready issues
    - Text: one line per issue — `<id> P<priority> <title>`
  - **Blocked command** in `internal/tl/cmd_blocked.go`:
    - List tasks where status != closed AND in blocked set
    - For each, show what's blocking it (the dep target IDs)
    - `--json`: JSON array with `blockers` field per issue
  - **Stats command** in `internal/tl/cmd_stats.go`:
    - Count by status: open, in_progress, blocked, closed, deferred
    - `--json`: `{"open": N, "in_progress": N, "blocked": N, "closed": N, "total": N}`
    - Text: `Open: N | In Progress: N | Blocked: N | Closed: N | Total: N`

  **Must NOT do**:
  - No `in_progress` tasks in ready queue (they're already claimed)
  - No performance optimization beyond simple computation (MVP scale)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T13, T14, T15)
  - **Blocks**: T17
  - **Blocked By**: T7, T11

  **References**:
  - `docs/references/beads/internal/types/types.go:730-734` — `AffectsReadyWork()` — MUST match: blocks, parent-child, conditional-blocks, waits-for
  - PRD Section 5.3 (`docs/PRD-streamlined-task-tool-variants.md:86-94`) — Ready queue rules (all 6 conditions)
  - PRD Section 5.1 (`docs/PRD-streamlined-task-tool-variants.md:56-64`) — Status model: which statuses appear in ready queue
  - `docs/references/ergo/internal/ergo/graph.go` — Ergo's readiness check (simpler: just checks deps resolved) — tl needs MORE checks (transitive parent-child, waits-for, defer_until)

  **Acceptance Criteria**:
  - [ ] `tl ready --json` returns only tasks that are open + unblocked + not deferred
  - [ ] Task with blocking dep on open task does NOT appear in ready
  - [ ] Closing the blocker makes blocked task appear in ready
  - [ ] Parent-child transitive blocking works (parent blocked → child blocked)
  - [ ] `tl blocked --json` shows blocked tasks with their blocker IDs
  - [ ] `tl stats --json` returns correct counts

  **QA Scenarios**:
  ```
  Scenario: Ready queue respects blocking deps
    Tool: Bash
    Steps:
      1. tl init
      2. tl create --title "Blocker" --json → $BLOCKER
      3. tl create --title "Blocked" --json → $BLOCKED
      4. tl dep add $BLOCKED $BLOCKER --type blocks
      5. tl ready --json → assert contains $BLOCKER, NOT $BLOCKED
      6. tl close $BLOCKER
      7. tl ready --json → assert contains $BLOCKED
    Expected Result: Blocking deps control readiness
    Evidence: .sisyphus/evidence/task-12-ready-blocking.txt

  Scenario: Stats reflect actual counts
    Tool: Bash
    Steps:
      1. tl create 3 tasks, close 1
      2. tl stats --json
      3. Assert {"open": 2, "closed": 1, "total": 3, ...}
    Expected Result: Accurate counts
    Evidence: .sisyphus/evidence/task-12-stats.txt
  ```

  **Commit**: YES (group with Wave 3)
  - Message: `feat(tl): ready queue, blocked cache, blocked and stats commands`
  - Files: `internal/tl/ready.go`, `internal/tl/cmd_ready.go`, `internal/tl/cmd_blocked.go`, `internal/tl/cmd_stats.go`, tests

- [ ] 13. Beads import with content-hash dedup

  **What to do**:
  - Implement `tl import` in `internal/tl/cmd_import.go`:
    - Flag: `--from` (path to beads JSONL file, default `.beads/issues.jsonl`)
    - Under lock: load tl graph → parse beads JSONL (using T6 parser) → dedup → generate events
    - Dedup logic using content hash (T4):
      1. For each beads issue, compute content hash
      2. Same hash + same ID in tl graph → skip (already imported)
      3. Different hash + same ID in tl graph → update (newer by updated_at wins)
      4. No matching ID → create new task
    - ID handling: keep beads IDs as-is (`bd-xxxx` stays `bd-xxxx`)
    - Dependencies: import as dep_add events
    - Unknown fields: preserved in Issue.Metadata
    - Output: `{"imported": N, "updated": N, "skipped": N}` with --json
    - Text: `Imported N, Updated N, Skipped N from <path>`

  **Must NOT do**:
  - No ID remapping (keep original beads IDs)
  - No interactive confirmation

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T14, T15)
  - **Blocks**: T17
  - **Blocked By**: T4, T6, T7

  **References**:
  - `docs/references/beads/internal/types/types.go:129-210` — `ComputeContentHash()` — MUST compute compatible hash for dedup to work correctly across beads↔tl
  - PRD Section 6.3 (`docs/PRD-streamlined-task-tool-variants.md:176-181`) — Import dedup rules: same hash=skip, different hash=update, no match=create
  - PRD Section 6.4 (`docs/PRD-streamlined-task-tool-variants.md:182-187`) — Hierarchical ID format and permissive parent handling
  - `.beads/issues.jsonl` — Actual beads data from THIS repo for real-world testing

  **Acceptance Criteria**:
  - [ ] `tl import --from internal/tl/testdata/beads_sample.jsonl` imports all tasks from fixture
  - [ ] Running import twice = zero new imports (dedup works)
  - [ ] Beads IDs preserved (`bd-xxxx` not remapped)
  - [ ] Dependencies from beads JSONL imported correctly
  - [ ] Unknown beads fields (hook_bead, mol_type, etc.) in Metadata

  **QA Scenarios**:
  ```
  Scenario: Import dedup — second run skips
    Tool: Bash
    Steps:
      1. tl init
      2. tl import --from internal/tl/testdata/beads_sample.jsonl --json → {"imported": N, ...}
      3. tl import --from internal/tl/testdata/beads_sample.jsonl --json → {"imported": 0, "skipped": N}
    Expected Result: Second import skips all (content hash match)
    Evidence: .sisyphus/evidence/task-13-dedup.txt

  Scenario: Import preserves beads IDs
    Tool: Bash
    Steps:
      1. tl import --from internal/tl/testdata/beads_sample.jsonl
      2. tl list --json | jq '.[].id'
      3. Assert IDs start with "bd-" (not remapped to "tl-")
    Expected Result: Original IDs preserved
    Evidence: .sisyphus/evidence/task-13-ids.txt
  ```

  **Commit**: YES (group with Wave 3)
  - Message: `feat(tl): beads import with content-hash deduplication`
  - Files: `internal/tl/cmd_import.go`, `internal/tl/cmd_import_test.go`

- [ ] 14. Beads export

  **What to do**:
  - Implement `tl export` in `internal/tl/cmd_export.go`:
    - Flag: `--to` (path to write, default `.beads/issues.jsonl`)
    - Load graph → serialize all tasks to beads JSONL format (one complete issue per line)
    - Each line is a complete JSON object with nested dependencies (matching beads format)
    - Metadata fields are PROMOTED back to top-level in the export (not nested under "metadata")
      - This is critical for round-trip: beads writes `hook_bead` at top level → tl imports to metadata → tl exports back to top level
    - Write to temp file first, then atomic rename (prevent partial writes)
    - Output: `{"exported": N, "path": "<path>"}` with --json
    - Text: `Exported N issues to <path>`

  **Must NOT do**:
  - No pretty-printing (one compact JSON object per line)
  - No filtering on export (export everything)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T13, T15)
  - **Blocks**: T16, T17
  - **Blocked By**: T2, T7

  **References**:
  - PRD Section 6.2 (`docs/PRD-streamlined-task-tool-variants.md:144-172`) — Beads JSONL export format with nested deps
  - `docs/references/beads/internal/types/types.go:15-127` — Full Issue struct — export must include all known fields at top level + promote metadata back
  - `docs/references/ergo/internal/ergo/storage.go:211-231` — `writeEventsFile` — atomic write pattern (write to file, flush, sync)

  **Acceptance Criteria**:
  - [ ] `tl export --to /tmp/out.jsonl` writes valid JSONL
  - [ ] Each line is a complete issue JSON with nested dependencies
  - [ ] Metadata fields promoted to top level (not nested under "metadata" key)
  - [ ] Atomic write: no partial file on crash (temp file + rename)
  - [ ] Export after import = field-level equivalent to original (round-trip)

  **QA Scenarios**:
  ```
  Scenario: Round-trip field preservation
    Tool: Bash
    Steps:
      1. tl import --from internal/tl/testdata/beads_sample.jsonl
      2. tl export --to /tmp/roundtrip.jsonl
      3. For each issue: compare fields (jq sort-by-key comparison)
      4. Assert no fields lost (metadata promoted back to top level)
    Expected Result: Zero field loss on round-trip
    Evidence: .sisyphus/evidence/task-14-roundtrip.txt
  ```

  **Commit**: YES (group with Wave 3)
  - Message: `feat(tl): beads export with metadata promotion`
  - Files: `internal/tl/cmd_export.go`, `internal/tl/cmd_export_test.go`

- [ ] 15. Atomic claim command

  **What to do**:
  - Implement `tl claim <id>` in `internal/tl/cmd_claim.go`:
    - Flags: `--agent` (string, required — who is claiming), `--json`
    - Under flock: loadGraph → validate → transition → append events
    - Validation rules:
      1. Task must exist
      2. Status must be `open` (claimable)
      3. If already claimed by SAME agent → idempotent success (return current state)
      4. If already claimed by DIFFERENT agent → error: "already claimed by <agent>"
      5. If status is closed/deferred → error: "cannot claim <status> task"
    - On success: set status=in_progress + assignee=<agent> in single update event
    - `--agent` can also be set via `TL_ACTOR` env var
    - Output: full Issue JSON with --json, or "Claimed <id> by <agent>" text

  **Must NOT do**:
  - No retry logic (fail fast)
  - No --force flag
  - No checking if task is "ready" — claiming any open task is valid regardless of blocking state (matches beads behavior)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T13, T14)
  - **Blocks**: T17
  - **Blocked By**: T5, T7

  **References**:
  - PRD Section 5.4 (`docs/PRD-streamlined-task-tool-variants.md:98-103`) — Claim semantics: atomic, fail-fast, idempotent re-claim
  - `docs/references/ergo/internal/ergo/model.go:98-100` — Claim invariant validation (state requires claim, claim requires state)
  - `docs/references/ergo/internal/ergo/commands_work.go` — Ergo's claim implementation (find oldest ready → claim under lock)
  - `docs/references/ergo/cmd/ergo/race_enabled_test.go` — Concurrent claim race test pattern

  **Acceptance Criteria**:
  - [ ] `tl claim <id> --agent claude --json` sets status=in_progress, assignee=claude
  - [ ] Re-claiming by same agent = idempotent success
  - [ ] Claiming task already claimed by different agent = error
  - [ ] Claiming closed task = error
  - [ ] Concurrent claims: exactly one winner (race test)

  **QA Scenarios**:
  ```
  Scenario: Concurrent claim race
    Tool: Bash
    Steps:
      1. tl create --title "Race test" --json → $ID
      2. tl claim $ID --agent agent-1 --json & PID1=$!
      3. tl claim $ID --agent agent-2 --json & PID2=$!
      4. wait $PID1; R1=$?; wait $PID2; R2=$?
      5. Assert exactly one exit 0, one exit non-zero
    Expected Result: One winner, one loser, no corruption
    Evidence: .sisyphus/evidence/task-15-race.txt

  Scenario: Idempotent re-claim
    Tool: Bash
    Steps:
      1. tl create --title "Test" --json → $ID
      2. tl claim $ID --agent claude --json → success
      3. tl claim $ID --agent claude --json → success (same agent)
      4. Assert both return exit 0
    Expected Result: Same-agent re-claim succeeds
    Evidence: .sisyphus/evidence/task-15-idempotent.txt
  ```

  **Commit**: YES (group with Wave 3)
  - Message: `feat(tl): atomic claim with race safety`
  - Files: `internal/tl/cmd_claim.go`, `internal/tl/cmd_claim_test.go`

- [ ] 16. Sync command (export + git add)

  **What to do**:
  - Implement `tl sync` in `internal/tl/cmd_sync.go`:
    - Export all tasks to `.beads/issues.jsonl` (reuse export logic from T14)
    - Run `git add .beads/issues.jsonl` (shell exec)
    - Output: `{"exported": N, "git_added": true}` with --json
    - Text: `Exported N issues to .beads/issues.jsonl and staged for git`
    - If `.beads/` directory doesn't exist, create it
  - `tl sync --from-main` variant:
    - Read `.beads/issues.jsonl` from git's main branch: `git show main:.beads/issues.jsonl`
    - Import those tasks (reuse import logic from T13)
    - This replaces `bd sync --from-main` for ephemeral branches
    - Output: `{"imported": N, "updated": N, "skipped": N}` with --json

  **Must NOT do**:
  - No `git commit` or `git push` — only `git add`
  - No complex merge/rebase logic
  - No worktree mechanics

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential: T16 → T17 → T18)
  - **Blocks**: T17
  - **Blocked By**: T14 (export), T13 (import for --from-main)

  **References**:
  - CLAUDE.md — `bd sync --from-main` usage: "Pull beads updates from main (for ephemeral branches)"
  - `docs/references/beads/docs/WORKTREES.md` — Complex sync mechanics (we're implementing the SIMPLE version only)
  - T14 output — reuse export function
  - T13 output — reuse import function for --from-main

  **Acceptance Criteria**:
  - [ ] `tl sync --json` exports to .beads/issues.jsonl and runs git add
  - [ ] `tl sync --from-main --json` imports from main branch's .beads/issues.jsonl
  - [ ] Creates .beads/ directory if it doesn't exist

  **QA Scenarios**:
  ```
  Scenario: Sync exports and stages
    Tool: Bash
    Steps:
      1. tl create --title "Sync test"
      2. tl sync --json
      3. Assert .beads/issues.jsonl exists
      4. git status → assert .beads/issues.jsonl is staged
    Expected Result: Export + git add works
    Evidence: .sisyphus/evidence/task-16-sync.txt
  ```

  **Commit**: YES (group with Wave 4)
  - Message: `feat(tl): sync command — export + git add`
  - Files: `internal/tl/cmd_sync.go`, tests

- [ ] 17. bd command aliases — dispatch wrapper

  **What to do**:
  - Create `cmd/bd/main.go` — a dispatch wrapper binary that routes bd commands to tl
  - Command mapping (from CLAUDE.md/AGENTS.md usage patterns):
    - `bd ready` → `tl ready`
    - `bd create "title"` → `tl create "title"`
    - `bd create "title" --description="..." -t task -p 2 --json` → `tl create --title "title" --description "..." --type task --priority 2 --json`
    - `bd show <id>` → `tl show <id>`
    - `bd update <id> --status in_progress` → `tl update <id> --status in_progress`
    - `bd update <id> --claim` → `tl claim <id> --agent <resolved-actor>`
    - `bd close <id> --reason "done"` → `tl close <id> --reason "done"`
    - `bd list --status=open` → `tl list --status open`
    - `bd blocked` → `tl blocked`
    - `bd stats` → `tl stats`
    - `bd dep add <a> <b>` → `tl dep add <a> <b>`
    - `bd sync` → `tl sync`
    - `bd sync --from-main` → `tl sync --from-main`
  - **Fallthrough**: Unknown bd commands → try to exec real `bd` binary (look in PATH excluding self). If not found, structured error: "unknown command '<cmd>', try 'tl <cmd>'"
  - This handles `bd prime`, `bd doctor`, `bd hooks` by falling through to real bd
  - Build produces both `tl` and `bd` binaries from same module

  **Must NOT do**:
  - No reimplementation of bd commands not in MVR scope
  - No symlink approach — use a real dispatch binary for flag translation
  - No attempt to match ALL bd flags — only the patterns in CLAUDE.md

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on all commands being implemented)
  - **Parallel Group**: Wave 4 (after T16)
  - **Blocks**: T18
  - **Blocked By**: T8, T9, T10, T11, T12, T13, T14, T15, T16

  **References**:
  - CLAUDE.md — Exact bd command patterns used in daily workflow: `bd ready`, `bd create "..."`, `bd show <id>`, `bd update <id> --status X`, `bd close <id>`, `bd sync`, `bd dep add`
  - AGENTS.md — Session completion protocol: `bd sync`, `bd close <id>`
  - `docs/references/beads/.github/copilot-instructions.md` — Full bd command reference for flag mapping

  **Acceptance Criteria**:
  - [ ] `bd ready --json` returns same output as `tl ready --json`
  - [ ] `bd create "Test" -t task -p 1 --json` creates task via tl
  - [ ] `bd update <id> --claim` translates to `tl claim <id>`
  - [ ] `bd sync --from-main` works
  - [ ] Unknown commands (bd prime, bd doctor) attempt real bd binary
  - [ ] Both `tl` and `bd` binaries build from `go build ./cmd/...`

  **QA Scenarios**:
  ```
  Scenario: bd alias full workflow
    Tool: Bash
    Steps:
      1. bd ready --json → (empty array or existing tasks)
      2. bd create "Alias test" --json → capture $ID
      3. bd show $ID --json → assert task exists
      4. bd update $ID --status in_progress --json → assert status changed
      5. bd close $ID --reason "done" --json → assert closed
      6. bd stats --json → assert closed count > 0
    Expected Result: Full bd workflow via tl dispatch
    Evidence: .sisyphus/evidence/task-17-bd-alias.txt

  Scenario: Unknown command fallthrough
    Tool: Bash
    Steps:
      1. bd nonexistent-command 2>&1
      2. Assert error message suggests tl equivalent or mentions unknown command
    Expected Result: Clean error for unknown commands
    Evidence: .sisyphus/evidence/task-17-fallthrough.txt
  ```

  **Commit**: YES (group with Wave 4)
  - Message: `feat(tl): bd command aliases with dispatch wrapper`
  - Files: `cmd/bd/main.go`, tests

- [ ] 18. E2E smoke test — full agent workflow

  **What to do**:
  - Create `internal/tl/e2e_test.go` — comprehensive integration test
  - Test exercises the COMPLETE agent workflow as described in CLAUDE.md:
    1. `tl init` — initialize project
    2. `tl create` — create multiple tasks with different types/priorities
    3. `tl dep add` — add dependencies between tasks
    4. `tl ready --json` — verify only unblocked tasks appear
    5. `tl claim` — claim a ready task
    6. `tl update` — update task fields
    7. `tl close` — close tasks, verify blocking cleared
    8. `tl blocked --json` — verify blocked list
    9. `tl stats --json` — verify counts
    10. `tl reopen` — reopen a closed task
    11. `tl import` — import from beads JSONL fixture
    12. `tl export` — export back to beads format
    13. Verify round-trip: field-level comparison (not byte-level — JSON key ordering may differ)
    14. `tl sync` — export + git add
  - Also test error cases:
    - Create without init → error
    - Show nonexistent ID → error
    - Cycle detection → error
    - Invalid status transition → error
    - Concurrent claims → exactly one winner
  - Use `t.TempDir()` for all tests, build binary with `go build` in TestMain
  - Run via bd alias binary too (verify dispatch works end-to-end)

  **Must NOT do**:
  - No mocking — real filesystem, real binary, real JSONL
  - No human-required verification steps

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on everything)
  - **Parallel Group**: Wave 4 (after T17)
  - **Blocks**: F1, F2, F3, F4
  - **Blocked By**: All T1-T17

  **References**:
  - `docs/references/ergo/cmd/ergo/integration_test.go` — Ergo's integration test pattern (build binary, run in temp dir, assert output)
  - `docs/references/ergo/cmd/ergo/race_enabled_test.go` — Concurrent race test pattern
  - `docs/references/jari/tests/test_cli_e2e.py` — Jari's comprehensive E2E test as reference for coverage
  - CLAUDE.md — The exact workflow the E2E test must replicate

  **Acceptance Criteria**:
  - [ ] `go test -run TestE2E -count=1 -v ./internal/tl/` passes
  - [ ] All happy-path scenarios pass
  - [ ] All error-case scenarios pass
  - [ ] Concurrent claim test passes (race detector clean)
  - [ ] Import/export round-trip verified
  - [ ] bd alias dispatch verified

  **QA Scenarios**:
  ```
  Scenario: Full E2E
    Tool: Bash
    Steps:
      1. go test -run TestE2E -count=1 -race -v ./internal/tl/
      2. Assert all subtests pass
      3. Assert exit 0
    Expected Result: All E2E tests pass including race detector
    Evidence: .sisyphus/evidence/task-18-e2e.txt

  Scenario: E2E via bd alias
    Tool: Bash
    Steps:
      1. go test -run TestE2EBdAlias -count=1 -v ./internal/tl/
      2. Assert bd alias routes to tl correctly
    Expected Result: bd dispatch works for all daily commands
    Evidence: .sisyphus/evidence/task-18-bd-e2e.txt
  ```

  **Commit**: YES
  - Message: `feat(tl): E2E smoke test — full agent workflow`
  - Files: `internal/tl/e2e_test.go`

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan and PRD end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./...`. Review all files for: type assertion panics, empty error handling, fmt.Println in prod code, commented-out code, unused imports. Check naming conventions match AGENTS.md guidelines (no implementation-detail names, no temporal names).
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real QA** — `unspecified-high`
  Start from clean state (`rm -rf .tl/`). Execute full workflow: init → create tasks → add deps → check ready → claim → close → reopen → import from beads → export → verify round-trip. Test edge cases: empty state, invalid input, concurrent claims, cycle detection, unknown status round-trip. Run all via bd aliases to verify dispatch works. Save evidence to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual code. Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT Have" compliance: search for Store interface, snapshot.json, color codes, interactive prompts, logging frameworks, shell completion. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- After Wave 1: `feat(tl): foundation — types, events, lock, parser`
- After Wave 2: `feat(tl): core CRUD — init, create, list, show, update, close, reopen, deps`
- After Wave 3: `feat(tl): ready queue, claim, import/export`
- After Wave 4: `feat(tl): sync, bd aliases, E2E smoke test`
- Final: `feat(tl): complete MVR — beads replacement ready`

---

## Success Criteria

### Verification Commands
```bash
go test ./... -count=1                  # Expected: all pass
go vet ./...                            # Expected: no issues
tl init && tl create --title "Test" --type task --priority 1 --json  # Expected: valid JSON with id
tl ready --json                         # Expected: valid JSON array containing the task
tl claim <id> --agent test --json       # Expected: status=in_progress, assignee=test
tl close <id> --reason "done" --json    # Expected: status=closed
tl import --from internal/tl/testdata/beads_sample.jsonl  # Expected: exit 0, tasks imported
tl export --to /tmp/out.jsonl           # Expected: exit 0, JSONL written
bd ready --json                         # Expected: same as tl ready (alias works)
```

### Final Checklist
- [ ] All 12+ CLI commands functional with --json
- [ ] bd alias dispatch works for all CLAUDE.md patterns
- [ ] Import/export round-trip: zero field-level data loss
- [ ] Unknown fields preserved in metadata and re-promoted on export
- [ ] Concurrent claim test: exactly 1 winner
- [ ] Dependency cycle detection works
- [ ] Ready queue respects all blocking dep types
- [ ] All tests pass, zero vet warnings
