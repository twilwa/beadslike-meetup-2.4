# PRD: Streamlined Beads-Compatible Task Tool (A+D Hybrid)

> **Decision**: Variant A (Beads-Lite JSONL Engine) as target engine,
> with Variant D migration mechanics (shadow-read, dual-write, rollback gates) from day one.

## 1. Problem Statement

Beads is powerful but operationally heavy for daily single-repo planning.
The full schema (120+ fields), Dolt backend, daemon process, and worktree machinery
impose cognitive and startup overhead that outweighs the value for most agent-driven workflows.
Teams want faster startup, simpler internals, and lower cognitive load while keeping
practical compatibility with existing beads tasks and workflows.

## 2. Product Goal

Build a single-binary CLI (`tl`) with append-only JSONL storage and file-lock concurrency,
optimized for agentic coding workflows. Include a beads compatibility adapter and phased
migration tooling (shadow-read → dual-write → cutover) so existing `.beads/issues.jsonl`
data and `bd`-style workflows continue working throughout the transition.

## 3. Target Users

- AI-assisted solo developers using agent harnesses (Claude Code, Cursor, etc.).
- Small teams using git branches and PR-based workflows.
- Existing beads users who want lighter local performance without data loss.

## 4. Scope

### In Scope (v1)

- **Core Engine**: `.tl/events.jsonl` append-only storage + materialized snapshot.
- **Beads Adapter**: Import/export `.beads/issues.jsonl` with content-hash deduplication.
- **CLI**: Full command set with `--json` for agent consumption + `bd` command aliases.
- **Ready Queue**: Dependency-aware, status-aware, with materialized blocked cache.
- **Atomic Claim**: File-lock serialized, race-safe, fail-fast on contention.
- **Dependency Graph**: Typed edges, cycle detection (DFS), transitive blocking.
- **Migration Tooling**: Shadow-read mode, dual-write mode, cutover with rollback.
- **Golden Corpus**: Round-trip compatibility test suite against real beads data.

### Out of Scope (v1)

- Daemon process (auto-sync in background).
- Dolt backend / SQL storage.
- Federation / peer-to-peer sync.
- Advanced message threading and inter-agent messaging.
- Web UI.

---

## 5. Behavioral Parity Matrix

### 5.1 Status Model

The tool MUST support these statuses with identical semantics to beads:

| Status | In Ready Queue | Can Claim | Can Block Others | Notes |
|--------|---------------|-----------|-----------------|-------|
| `open` | YES | YES | YES (if dep target) | Default for new tasks |
| `in_progress` | YES | NO (already claimed) | YES | Set by `claim` |
| `blocked` | NO | NO | YES (transitive) | Set when blocking dep exists |
| `deferred` | NO | NO | NO | Hidden from `ready` until reactivated |
| `closed` | NO | NO | NO | Closing clears blocking edges |
| `pinned` | NO | NO | NO | Persistent context marker, not work item |
| `hooked` | NO | NO | NO | Agent-attached work item (beads agent-as-bead) |

Soft-delete semantics: closing with `close_reason: "tombstone"` prevents resurrection on import (ergo-style).
This is NOT a separate status -- it uses `closed` status + close_reason metadata.

Unknown status values from beads MUST round-trip without loss (stored in metadata).

### 5.2 Dependency Types

| Type | Affects Ready | Transitive | Notes |
|------|--------------|-----------|-------|
| `blocks` | YES | NO | Primary blocking edge |
| `parent-child` | YES | YES | Transitive: parent blocked → children blocked |
| `conditional-blocks` | YES | NO | Blocking with metadata conditions |
| `waits-for` | YES | NO | Async gate (fanout patterns) |
| `related` | NO | NO | Informational link |
| `discovered-from` | NO | NO | Provenance tracking |

Unknown dependency types from beads MUST round-trip without loss.

### 5.3 Ready Queue Rules

A task is **ready** if ALL of the following are true:
1. Status is `open` or `in_progress`.
2. No `blocks` dependency points to an open/in_progress/blocked task.
3. No `parent-child` dependency where the parent is blocked (transitive).
4. No `conditional-blocks` or `waits-for` dependency is active.
5. `defer_until` is null or in the past.
6. Task is not `tombstone`, `pinned`, or `deferred`.

Ready queue MUST be sorted by priority (ascending: 0 = P0/critical), then `created_at`.

### 5.4 Claim Semantics

- `tl claim <id> --agent <name>` is atomic under file lock.
- Sets `assignee` + transitions status to `in_progress` in one operation.
- MUST fail fast if: already claimed by different agent, status is closed/deferred/tombstone.
- Re-claiming by same agent is idempotent (returns success).
- Lock contention: fail with structured error, caller retries.

### 5.5 CLI Command Parity

| Command | tl equivalent | bd equivalent | Notes |
|---------|--------------|--------------|-------|
| Create | `tl create` | `bd create` | Returns JSON with new ID |
| List | `tl list` | `bd list` | Filterable by status, type, assignee |
| Show | `tl show <id>` | `bd show <id>` | Full task detail |
| Ready | `tl ready` | `bd ready` | Sorted ready queue |
| Update | `tl update <id>` | `bd update <id>` | Field-level updates |
| Claim | `tl claim <id>` | `bd update <id> --claim` | Atomic claim |
| Close | `tl close <id>` | `bd close <id>` | Sets closed_at, clears blocking |
| Reopen | `tl reopen <id>` | `bd reopen <id>` | Clears closed_at, sets open |
| Dep Add | `tl dep add` | `bd dep add` | With type, cycle check |
| Dep Remove | `tl dep remove` | `bd dep remove` | Cleans both sides |
| Dep Tree | `tl dep tree` | `bd dep tree` | Visual dependency tree |
| Blocked | `tl blocked` | `bd blocked` | List blocked with blocker info |
| Import | `tl import` | N/A | From beads JSONL |
| Export | `tl export` | N/A | To beads JSONL |
| Sync | `tl sync` | `bd sync` | Export + git commit |
| Stats | `tl stats` | `bd stats` | Open/closed/blocked counts |

All commands MUST support `--json` for stable machine-readable output.

---

## 6. JSONL Schema Contract

### 6.1 On-Disk Format (`.tl/events.jsonl`)

Append-only event log. Each line is a complete JSON object representing an event:

```json
{"type":"create","id":"tl-a3f8","data":{...},"ts":"...","actor":"agent-1"}
{"type":"update","id":"tl-a3f8","data":{"status":"in_progress","assignee":"claude"},"ts":"...","actor":"claude"}
{"type":"close","id":"tl-a3f8","data":{"close_reason":"done"},"ts":"...","actor":"claude"}
```

Materialized snapshot (`.tl/snapshot.json`) rebuilt from event replay.
Snapshot used for reads; events are source of truth.

### 6.2 Beads JSONL Import/Export Format

Each issue is a complete JSON object per line (matching beads `issues.jsonl`):

```json
{
  "id": "bd-a3f8.1",
  "title": "...",
  "description": "...",
  "status": "closed",
  "priority": 1,
  "issue_type": "task",
  "assignee": "agent-1",
  "created_at": "2025-12-16T03:02:17.603608-08:00",
  "updated_at": "2025-12-16T18:07:42.94048-08:00",
  "closed_at": "2025-12-16T18:07:42.94048-08:00",
  "dependencies": [
    {
      "issue_id": "bd-a3f8.1",
      "depends_on_id": "bd-a3f8",
      "type": "parent-child",
      "created_at": "2025-12-16T03:02:17.604343-08:00",
      "created_by": "stevey"
    }
  ],
  "labels": ["backend"],
  "metadata": {}
}
```

### 6.3 Import Deduplication

Import uses content-hash comparison (SHA256 of title + description + design + acceptance_criteria):
- Same hash, same ID → skip (already imported).
- Different hash, same ID → update (newer version wins by `updated_at`).
- No match → create new task.
- Unknown fields → preserved in `metadata` for round-trip safety.

### 6.4 Hierarchical IDs

- Format: `prefix-hash` (root) or `prefix-hash.N` (child), e.g., `tl-a3f8.1`.
- Depth limit: 50 levels.
- Parent-child deps create hierarchy; missing parents import permissively.

---

## 7. Migration Mechanics (from Variant D)

### 7.1 Migration Phases

| Phase | Mode | Reads From | Writes To | Rollback |
|-------|------|-----------|----------|----------|
| 0 | Baseline | beads only | beads only | N/A |
| 1 | Shadow-read | beads (primary) | beads only | Disable shadow |
| 2 | Dual-write | beads (primary) | both | Disable tl writes |
| 3 | Cutover | tl (primary) | both | Swap primary back |
| 4 | Standalone | tl only | tl + beads export | Re-import from export |

### 7.2 Hard Cutover Gates

Each phase transition requires ALL gates to pass:

**Phase 0 → 1 (Enable shadow-read)**:
- [ ] Golden corpus round-trip: 100% field preservation on test JSONL.
- [ ] `tl ready` output matches `bd ready` on same data (ID-set parity).

**Phase 1 → 2 (Enable dual-write)**:
- [ ] Shadow-read discrepancy rate < 0.1% over 100+ operations.
- [ ] Zero status/blocking classification mismatches.
- [ ] `tl claim` race test passes (2 concurrent claimers → exactly 1 winner).

**Phase 2 → 3 (Cutover to tl-primary)**:
- [ ] Dual-write divergence rate = 0% over 500+ operations.
- [ ] Full CLI command parity verified (all commands in parity matrix).
- [ ] Performance: P95 < 50ms on 5k tasks for ready/list/show.

**Phase 3 → 4 (Standalone)**:
- [ ] 2-week soak with zero regressions.
- [ ] Rollback tested: tl → beads import succeeds with zero data loss.

### 7.3 Discrepancy Detection

Shadow-read compares tl output vs beads output on every read:
- Field-by-field comparison for show/list.
- ID-set comparison for ready/blocked.
- Discrepancies logged to `.tl/migration/discrepancies.jsonl`.
- `tl compat check` reports: field coverage %, command parity %, discrepancy rate.

### 7.4 Rollback

At any phase, rollback is one command:
- `tl migrate rollback` → reverts to previous phase.
- Beads data is always readable (export maintained in all phases).
- Rollback verified by: re-running golden corpus against beads data.

---

## 8. Concurrency Model

### 8.1 Write Serialization

All mutations acquire exclusive `flock(2)` on `.tl/lock` before appending to events.
Lock is fail-fast: if held, command returns structured error with retry guidance.

### 8.2 Blocked Cache

Materialized blocked-tasks set rebuilt on any dependency or status change:
- Only `blocks`, `parent-child`, `conditional-blocks`, `waits-for` trigger rebuild.
- Rebuild cost: O(tasks × avg_deps), target < 50ms on 5k tasks.
- Cache stored in snapshot, not events.

### 8.3 Snapshot Rebuild

On startup or after events exceed threshold (default: 100 uncommitted events):
- Replay all events to rebuild snapshot.
- Compact events: rewrite events.jsonl with only current state.
- Target: < 200ms full replay on 5k tasks.

---

## 9. Non-Functional Requirements

- P95 command latency under 50ms on 5k tasks (ready, list, show, claim).
- No daemon, no network requirement, offline-first.
- Deterministic outputs: same input → same output (for reproducible agent behavior).
- Single static binary, no runtime dependencies.
- JSONL output: stable field ordering, one entity per line, no pretty-print.
- Git-friendly: append-only events + snapshot are merge-safe.

## 10. Success Metrics

- 3x+ faster median command latency vs beads on same task corpus.
- 100% round-trip compatibility for core beads fields (Section 5 matrix).
- Zero data loss in golden corpus migration tests.
- 0 critical regressions during 2-week soak period before standalone cutover.

## 11. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Event log growth degrades replay | Slow startup on large repos | Compaction threshold + snapshot cache |
| Compatibility gaps for rare beads fields | Silent data loss on round-trip | Unknown fields → metadata passthrough + golden corpus tests |
| File lock contention under heavy multi-agent use | Claim failures, retries | Fail-fast with structured error; document retry pattern |
| Shadow-read overhead during migration | Slower reads in Phase 1-2 | Shadow-read is async/optional per command |
| Dual-write divergence | Inconsistent state | Hard gates + discrepancy logging + auto-rollback on divergence |

## 12. Technology Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, fast startup, matches beads ecosystem |
| Storage | Append-only JSONL + snapshot | Git-friendly, inspectable, no binary deps |
| Concurrency | flock(2) file lock | Simple, cross-process, no daemon needed |
| ID generation | Hash-based with prefix | Collision-resistant, hierarchical support |
| CLI framework | cobra | Standard Go CLI, matches beads |

## 13. Open Decisions (Defaults Applied)

- **Command name**: `tl` (with `bd` aliases via symlink/alias). Override if team prefers different name.
- **Single-repo only in v1**: Multi-repo deferred. Override if multi-repo is day-1 requirement.
- **No daemon in v1**: Manual sync only. Override if background sync is critical.
- **Go as implementation language**: Matches beads ecosystem. Override if team prefers different language.
