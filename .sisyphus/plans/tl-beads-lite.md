# Work Plan: `tl` — Streamlined Beads-Compatible Task Tool

## TL;DR

> **Quick Summary**: Build a Go CLI (`tl`) with append-only JSONL storage, beads behavioral parity (ready/claim/dep/status), and phased migration mechanics (shadow-read → dual-write → cutover).
>
> **Deliverables**:
> - Single-binary Go CLI with full command set
> - Beads JSONL import/export with content-hash deduplication
> - Migration tooling with hard phase gates
> - Golden corpus test suite for round-trip validation
> - `bd` command aliases for seamless transition
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 5 waves + final verification
> **Critical Path**: Types → Storage → Blocked Cache → Ready Queue → Migration Tooling → Golden Corpus E2E

---

## Context

### Original Request
Build a better, more streamlined version of the beads task planning tool while maintaining compatibility with beads tasks. Based on review of 5 reference projects (beads, ergo, jari, niwa, Backlog.md), the A+D hybrid approach was selected: Variant A's JSONL engine as target, with Variant D's migration mechanics from day one.

### Interview Summary
**Key Decisions**:
- Go as implementation language (matches beads ecosystem, single binary)
- Append-only JSONL + snapshot (git-friendly, inspectable, no binary deps)
- flock(2) for concurrency (simple, cross-process, no daemon)
- Full behavioral parity matrix specified (statuses, dep types, ready rules, claim semantics)
- Phased migration with hard cutover gates

**Research Findings**:
- Ergo validates append-only JSONL + flock is viable and fast for this scale
- Beads `AffectsReadyWork` includes `conditional-blocks` and `waits-for` beyond basic `blocks`
- Beads status set includes `pinned` and `hooked` beyond common workflow statuses
- Materialized blocked cache provides 25x speedup vs recursive CTE at scale
- Content-hash dedup is essential for multi-clone import correctness

### Metis Review
**Identified Gaps (addressed)**:
- Status model: `tombstone` was ergo concept, not beads status — corrected to `closed` + `close_reason`
- Content hash scope: SHA256 of title + description + design + acceptance_criteria
- Export format: beads JSONL (one complete issue per line with nested deps), not ergo events
- `conditional-blocks` semantics: same blocking behavior as `blocks`, with metadata conditions

---

## Work Objectives

### Core Objective
Deliver a production-ready CLI tool that replaces beads for daily task management with lower overhead, while maintaining data and behavioral compatibility.

### Concrete Deliverables
- `tl` binary with 16+ commands (see PRD Section 5.5)
- `.tl/` directory structure with events.jsonl, snapshot.json, lock
- Beads adapter: import from / export to `.beads/issues.jsonl`
- Migration tooling: shadow-read, dual-write, cutover, rollback
- `bd` command aliases (symlink or wrapper)
- Golden corpus test suite + benchmark suite
- PRD at `docs/PRD-streamlined-task-tool-variants.md` (already complete)

### Definition of Done
- [ ] All 16 CLI commands work with `--json` output
- [ ] `tl ready` matches `bd ready` output on golden corpus (ID-set parity)
- [ ] `tl claim` is race-safe (concurrent test: 2 claimers → 1 winner)
- [ ] Round-trip: beads export → tl import → tl export → diff = zero
- [ ] P95 < 50ms on 5k tasks for ready/list/show/claim
- [ ] Migration phases 0-4 all pass gate checks
- [ ] All tests pass, zero lint errors

### Must Have
- Behavioral parity for ready/claim/dep/status per PRD Section 5
- Content-hash deduplication on import
- Materialized blocked cache for ready performance
- Hierarchical IDs with parent-child semantics
- `--json` on all commands with stable schema
- Migration phase gates with hard thresholds

### Must NOT Have (Guardrails)
- No daemon process in v1
- No Dolt/SQL backend
- No web UI
- No inter-agent messaging / federation
- No markdown-as-source-of-truth (JSONL is truth)
- No pretty-printed JSON in storage files (one object per line)
- No mutable event log (append-only until explicit compaction)

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
- **Import/Export**: Use Bash — Round-trip diff, assert zero differences
- **Performance**: Use Bash — `time` command, assert P95 < threshold

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — all independent, START IMMEDIATELY):
├── Task 1: Go module scaffold + cobra CLI skeleton [quick]
├── Task 2: Core types — Issue, Dependency, Status, enums [quick]
├── Task 3: Event schema + JSON serialization [quick]
├── Task 4: Content hash utility [quick]
├── Task 5: Golden corpus test fixtures from beads reference [quick]
├── Task 6: flock(2) lock manager [quick]
└── Task 7: Beads JSONL parser (read .beads/issues.jsonl) [quick]

Wave 2 (Storage + Core Commands — depend on Wave 1):
├── Task 8: Event store (append + replay + snapshot) [deep] (depends: 3, 6)
├── Task 9: Create command [unspecified-high] (depends: 1, 2, 8)
├── Task 10: List + Show commands [unspecified-high] (depends: 1, 2, 8)
├── Task 11: Update command [unspecified-high] (depends: 1, 2, 8)
├── Task 12: Close + Reopen commands [unspecified-high] (depends: 1, 2, 8)
├── Task 13: Dependency CRUD (add/remove) with cycle detection [deep] (depends: 2, 8)
└── Task 14: Materialized blocked cache [deep] (depends: 2, 8, 13)

Wave 3 (Ready/Claim/Graph — depend on blocked cache):
├── Task 15: Ready queue command [deep] (depends: 14)
├── Task 16: Atomic claim command [deep] (depends: 6, 14, 15)
├── Task 17: Blocked command [unspecified-high] (depends: 14)
├── Task 18: Dep tree command [unspecified-high] (depends: 13)
├── Task 19: Stats command [quick] (depends: 8)
├── Task 20: Sync command (export + git add/commit) [unspecified-high] (depends: 8)
└── Task 21: Snapshot compaction [unspecified-high] (depends: 8)

Wave 4 (Migration + Compatibility — depend on core commands):
├── Task 22: Beads export (to .beads/issues.jsonl) [deep] (depends: 8, 2)
├── Task 23: Beads import with content-hash dedup [deep] (depends: 7, 4, 8)
├── Task 24: bd command aliases [quick] (depends: 1)
├── Task 25: Shadow-read mode [unspecified-high] (depends: 10, 15, 22)
├── Task 26: Dual-write mode [unspecified-high] (depends: 25)
├── Task 27: Compat check command [unspecified-high] (depends: 25)
└── Task 28: Migrate rollback command [unspecified-high] (depends: 26)

Wave 5 (Integration + Verification — depend on all above):
├── Task 29: Golden corpus round-trip E2E tests [deep] (depends: 22, 23, 5)
├── Task 30: Concurrent claim race test [deep] (depends: 16)
├── Task 31: Performance benchmarks (P95 < 50ms on 5k) [unspecified-high] (depends: 15, 16)
├── Task 32: Migration phase gate E2E tests [deep] (depends: 25, 26, 27, 28)
└── Task 33: CLI help text + quickstart doc [writing] (depends: all commands)

Wave FINAL (Independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real QA — full CLI workflow (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: T2 → T8 → T13 → T14 → T15 → T16 → T29 → F1-F4
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 7 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | 9-12, 24 | 1 |
| 2 | — | 8-14, 22 | 1 |
| 3 | — | 8 | 1 |
| 4 | — | 23 | 1 |
| 5 | — | 29 | 1 |
| 6 | — | 8, 16 | 1 |
| 7 | — | 23 | 1 |
| 8 | 3, 6 | 9-14, 19-23 | 2 |
| 9 | 1, 2, 8 | — | 2 |
| 10 | 1, 2, 8 | 25 | 2 |
| 11 | 1, 2, 8 | — | 2 |
| 12 | 1, 2, 8 | — | 2 |
| 13 | 2, 8 | 14, 18 | 2 |
| 14 | 2, 8, 13 | 15-17 | 2 |
| 15 | 14 | 16, 25, 31 | 3 |
| 16 | 6, 14, 15 | 30, 31 | 3 |
| 17 | 14 | — | 3 |
| 18 | 13 | — | 3 |
| 19 | 8 | — | 3 |
| 20 | 8 | — | 3 |
| 21 | 8 | — | 3 |
| 22 | 8, 2 | 25, 29 | 4 |
| 23 | 7, 4, 8 | 29 | 4 |
| 24 | 1 | — | 4 |
| 25 | 10, 15, 22 | 26, 27 | 4 |
| 26 | 25 | 28, 32 | 4 |
| 27 | 25 | 32 | 4 |
| 28 | 26 | 32 | 4 |
| 29 | 22, 23, 5 | F1-F4 | 5 |
| 30 | 16 | F1-F4 | 5 |
| 31 | 15, 16 | F1-F4 | 5 |
| 32 | 25-28 | F1-F4 | 5 |
| 33 | all cmds | F1-F4 | 5 |

### Agent Dispatch Summary

- **Wave 1**: 7 tasks — T1-T5 → `quick`, T6 → `quick`, T7 → `quick`
- **Wave 2**: 7 tasks — T8 → `deep`, T9-T12 → `unspecified-high`, T13-T14 → `deep`
- **Wave 3**: 7 tasks — T15-T16 → `deep`, T17-T18 → `unspecified-high`, T19 → `quick`, T20-T21 → `unspecified-high`
- **Wave 4**: 7 tasks — T22-T23 → `deep`, T24 → `quick`, T25-T28 → `unspecified-high`
- **Wave 5**: 5 tasks — T29-T30 → `deep`, T31 → `unspecified-high`, T32 → `deep`, T33 → `writing`
- **FINAL**: 4 tasks — F1 → `oracle`, F2-F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan and PRD end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./...`. Review all files for: type assertion panics, empty error handling, fmt.Println in prod, commented-out code, unused imports. Check naming conventions match AGENTS.md guidelines.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real QA** — `unspecified-high`
  Start from clean state. Execute full workflow: create tasks → add deps → check ready → claim → close → reopen → import from beads → export to beads → verify round-trip. Test edge cases: empty state, invalid input, concurrent claims. Save evidence to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual code. Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- After each Wave: `feat(tl): <wave description>`
- Final: `feat(tl): complete v1 with migration tooling`

---

## Success Criteria

### Verification Commands
```bash
go test ./... -count=1           # Expected: all pass
go vet ./...                     # Expected: no issues
tl ready --json                  # Expected: valid JSON array
tl import --from beads && tl export --to beads && diff .beads/issues.jsonl exported.jsonl  # Expected: zero diff
time tl ready --json             # Expected: < 50ms
```

### Final Checklist
- [ ] All 16 CLI commands functional with --json
- [ ] Golden corpus round-trip: zero data loss
- [ ] Concurrent claim test: exactly 1 winner
- [ ] P95 < 50ms on 5k tasks
- [ ] Migration phase gates all pass
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
