# Task Organization & Agentic Coding Workflow Reference Inventory

## Executive Summary

This inventory maps 5 reference projects in `/docs/references/` that implement task organization, decomposition, and agentic coding workflows. Each project represents a different architectural philosophy, from heavyweight (Beads) to lightweight (Ergo), with specialized variants for document editing (Niwa) and conflict-aware task tracking (Jari).

---

## Project Inventory

### 1. **Beads (bd)** — Distributed Graph Issue Tracker
**Location:** `/docs/references/beads/`  
**Language:** Go  
**Status:** Production (steveyegge/beads)

#### Purpose
Persistent, structured memory for coding agents. Replaces markdown plans with a dependency-aware graph for long-horizon tasks.

#### Architecture Style
- **Three-layer model**: Git (historical truth) → JSONL (operational truth) → SQLite (fast queries)
- **Storage**: Dolt (version-controlled SQL) + JSONL for git portability
- **Concurrency**: Hash-based IDs (`bd-a1b2`) prevent merge collisions
- **Hierarchy**: Supports epics with dot-notation (`bd-a3f8.1.1`)

#### Task Model
- **Entities**: Issues (with types: task, bug, feature, message)
- **State Machine**: `todo`, `in_progress`, `done`, `blocked`, `error`, `canceled`
- **Dependencies**: `blocks`, `blocked_by`, `relates_to`, `duplicates`, `supersedes`, `replies_to`
- **Metadata**: Priority (0-4), assignee, labels, description, design notes, acceptance criteria
- **Hierarchy**: Parent-child relationships via dot notation

#### Key Features
- **Compaction**: Semantic "memory decay" summarizes old closed tasks
- **Messaging**: Message type with threading and ephemeral lifecycle
- **Graph Links**: Knowledge graph support (relates_to, duplicates, supersedes)
- **Multi-agent**: Stealth mode, contributor vs maintainer roles
- **MCP Integration**: Python MCP server for Claude/Codex integration
- **Audit Trail**: Full version history with `bd show <id>`

#### Notable Strengths
✅ Mature, battle-tested in production  
✅ Rich dependency model (graph-based, not just linear)  
✅ Compaction for context window management  
✅ Multi-role support (contributor/maintainer)  
✅ Comprehensive MCP integration  
✅ Git-native with full history recovery  

#### Notable Weaknesses
❌ Complex architecture (3 layers, Dolt, SQLite)  
❌ Slower than alternatives (5-15x slower than Ergo)  
❌ Steep learning curve for agents  
❌ Daemon required for background operations  
❌ Heavy dependencies (Dolt, Go runtime)  

#### Beads-Compatible Hooks
- **ID Format**: Hash-based (`bd-a1b2`), hierarchical with dots
- **Storage**: JSONL + Git (portable)
- **Sync Model**: `bd sync` for bidirectional git sync
- **Ready Queue**: `bd ready` returns unblocked tasks
- **Claim Model**: Atomic claim via `bd update <id> --claim`

---

### 2. **Ergo** — Fast, Minimal Planning CLI
**Location:** `/docs/references/ergo/`  
**Language:** Go  
**Status:** Production (sandover/ergo)

#### Purpose
Fast, minimal planning tool for Claude Code and Codex. Stores epics & tasks in repo as compact JSONL.

#### Architecture Style
- **Single-layer model**: Append-only JSONL event log (`.ergo/plans.jsonl`)
- **Replay-based**: State reconstructed from events on each command
- **Concurrency**: File lock (`flock`) for write serialization
- **Simplicity**: No daemons, no git hooks, few opinions

#### Task Model
- **Entities**: Epic (grouping only), Task (unit of work)
- **State Machine**: `todo`, `doing`, `done`, `blocked`, `canceled`, `error`
- **Dependencies**: Task-to-task, Epic-to-epic (no cross-kind)
- **Metadata**: Title, body, claim (agent ID), state
- **Hierarchy**: Epic-to-task parent-child only

#### Key Features
- **Speed**: 5-15x faster than Beads (especially large projects)
- **Simplicity**: No daemons, no git hooks
- **Concurrency Safety**: File lock serializes writes, race-safe claiming
- **Append-only**: Corruption tolerance (truncated final line OK)
- **Compaction**: `ergo compact` collapses history
- **Unixy**: Text or JSON on stdin/stdout

#### Notable Strengths
✅ Extremely fast (replay is instant for 100-1000 tasks)  
✅ Minimal dependencies (just Go)  
✅ Simple, auditable code (append-only JSONL)  
✅ No daemons or background processes  
✅ Excellent for large projects  
✅ Clear state machine invariants  

#### Notable Weaknesses
❌ Simpler dependency model (no graph links)  
❌ No built-in compaction UI  
❌ Limited metadata (no design notes, acceptance criteria)  
❌ No messaging/threading  
❌ No multi-role support  
❌ Minimal MCP integration  

#### Ergo-Compatible Hooks
- **ID Format**: Base32 (e.g., `GQUJPG`)
- **Storage**: JSONL only (no git integration)
- **Concurrency**: File lock (`flock`) for writes
- **Ready Queue**: `ergo --json list --ready`
- **Claim Model**: `ergo claim <id> --agent <name>`
- **State Machine**: Strict invariants (doing/error require claim)

---

### 3. **Jari (砂利)** — Task Tracker for LLM Workflows
**Location:** `/docs/references/jari/`  
**Language:** Python  
**Status:** Production (secemp9/jari)

#### Purpose
Task/issue tracker built for multi-agent AI workflows. Handles priorities, dependencies, atomic claims, field-level conflict detection.

#### Architecture Style
- **LMDB-based**: Lightning Memory-Mapped Database for concurrent access
- **Field-level conflicts**: Tracks which fields changed, auto-merges non-overlapping edits
- **Niwa integration**: Links todos to markdown document sections
- **Claude Code hooks**: SessionStart, PreCompact, Stop for context injection

#### Task Model
- **Entities**: Todo (with type: feature, task, bug, chore)
- **State Machine**: `open`, `in_progress`, `done`, `closed`, `deferred`
- **Dependencies**: Blocked-by relationships with cycle detection
- **Metadata**: Priority (critical, high, medium, low, backlog), labels, assignee, description
- **Hierarchy**: Epic/parent-child relationships
- **Conflict Detection**: Field-level (same field = conflict, different fields = auto-merge)

#### Key Features
- **Ready Queue**: Automatically surfaces unblocked todos sorted by priority
- **Atomic Claim**: Race-safe task assignment (first agent wins)
- **Conflict Detection**: Field-level tracking detects overlapping edits
- **Auto-Merge**: Non-overlapping field changes merge automatically
- **Niwa Integration**: Link todos to markdown document nodes
- **Claude Code Hooks**: SessionStart, PreCompact, Stop for context awareness
- **Version History**: Full audit trail with rollback

#### Notable Strengths
✅ Field-level conflict detection (sophisticated)  
✅ LMDB for high-performance concurrent access  
✅ Niwa integration for document-task linking  
✅ Claude Code hooks for context injection  
✅ Atomic claim with race-safe semantics  
✅ Auto-merge for non-conflicting changes  

#### Notable Weaknesses
❌ Python-only (slower than Go alternatives)  
❌ LMDB dependency (less portable than JSONL)  
❌ No git integration (local database only)  
❌ Limited hierarchy (epic/parent-child only)  
❌ No graph links (relates_to, duplicates, etc.)  
❌ Smaller ecosystem than Beads  

#### Jari-Compatible Hooks
- **ID Format**: Sequential (`todo_1`, `todo_2`, ...)
- **Storage**: LMDB (local database)
- **Concurrency**: LMDB transactions for atomic operations
- **Ready Queue**: `jari ready` (sorted by priority)
- **Claim Model**: `jari claim <id> --agent <name>` (atomic)
- **Conflict Detection**: Field-level with auto-merge
- **Claude Hooks**: SessionStart, PreCompact, Stop

---

### 4. **Niwa (庭)** — Collaborative Markdown Editing
**Location:** `/docs/references/niwa/`  
**Language:** Python  
**Status:** Production (secemp9/niwa)

#### Purpose
CLI tool for multiple LLM agents to collaboratively edit markdown documents with automatic conflict detection and resolution.

#### Architecture Style
- **LMDB-based**: Lightning Memory-Mapped Database for concurrent access
- **AST-based parsing**: markdown-it-py token line mapping preserves exact formatting
- **Version vectors**: Each node tracks version independently
- **Agent isolation**: Each agent's pending reads/conflicts tracked separately

#### Task Model
- **Entities**: Nodes (headings → content hierarchy)
- **Hierarchy**: H1, H2, H3... with node IDs like `h2_3`
- **Metadata**: Title, content, version, agent, summary
- **Conflict Detection**: Version-based (same node edited by multiple agents)
- **Resolution**: ACCEPT_YOURS, ACCEPT_THEIRS, MANUAL_MERGE

#### Key Features
- **Concurrent Editing**: Multiple agents read/edit simultaneously
- **Conflict Detection**: Automatic version tracking
- **Smart Merging**: Auto-merge suggestions for compatible changes
- **Sub-Agent Support**: Stored conflicts survive context switches
- **Full Markdown Support**: GFM tables, task lists, footnotes, frontmatter
- **Claude Code Hooks**: SessionStart, PreCompact, PreToolUse, PostToolUse, Stop
- **Search**: Find content by keyword

#### Notable Strengths
✅ Full GFM markdown support (tables, task lists, footnotes)  
✅ AST-based parsing (preserves formatting)  
✅ Version-based conflict detection  
✅ Claude Code hooks for context injection  
✅ Sub-agent support (conflicts survive context switches)  
✅ Search by keyword  

#### Notable Weaknesses
❌ Document-focused (not task-focused)  
❌ No dependency tracking  
❌ No priority system  
❌ No git integration  
❌ LMDB dependency  
❌ Python-only  

#### Niwa-Compatible Hooks
- **ID Format**: Node-based (`h1_0`, `h2_3`, `h3_5`)
- **Storage**: LMDB (local database)
- **Concurrency**: LMDB transactions
- **Conflict Detection**: Version-based per node
- **Resolution**: ACCEPT_YOURS, ACCEPT_THEIRS, MANUAL_MERGE
- **Claude Hooks**: SessionStart, PreCompact, PreToolUse, PostToolUse, Stop

---

### 5. **Backlog.md** — Markdown-Native Task Manager
**Location:** `/docs/references/Backlog.md/`  
**Language:** TypeScript (Bun)  
**Status:** Production (MrLesk/Backlog.md)

#### Purpose
Markdown-native task manager & Kanban visualizer for any Git repository. Built for spec-driven AI development.

#### Architecture Style
- **Markdown-first**: Tasks stored as `.md` files in `backlog/` directory
- **Zero-config**: Works with any Git repo
- **Multi-interface**: CLI, TUI (terminal), Web UI
- **MCP-native**: Designed for Claude Code, Codex, Gemini CLI, Kiro
- **Git-integrated**: All data in repo, no external service

#### Task Model
- **Entities**: Task, Draft, Document, Decision, Milestone
- **State Machine**: `To Do`, `In Progress`, `Done`, `Archived`
- **Metadata**: ID, title, description, acceptance criteria, priority, assignee, labels, notes
- **Hierarchy**: Parent-child via task ID references
- **Kanban**: Status-based columns (configurable)

#### Key Features
- **Markdown Files**: Each task is a `.md` file (human-readable)
- **Instant Kanban**: `backlog board` paints live board in shell
- **Web UI**: `backlog browser` launches modern web interface
- **Search**: Fuzzy search across tasks, docs, decisions
- **MCP Integration**: Auto-configures Claude Code, Codex, Gemini CLI, Kiro
- **Definition of Done**: Reusable checklist defaults
- **Board Export**: `backlog board export` creates shareable markdown
- **Offline**: 100% private, no external service

#### Notable Strengths
✅ Markdown-native (human-readable, git-friendly)  
✅ Zero-config (works with any repo)  
✅ Multiple interfaces (CLI, TUI, Web)  
✅ MCP-native (designed for AI agents)  
✅ Rich metadata (acceptance criteria, notes, etc.)  
✅ Instant Kanban visualization  
✅ Web UI with drag-and-drop  

#### Notable Weaknesses
❌ No dependency tracking (parent-child only)  
❌ No graph links  
❌ No conflict detection  
❌ No atomic claims  
❌ File-based (slower for large projects)  
❌ TypeScript/Bun dependency  

#### Backlog.md-Compatible Hooks
- **ID Format**: Sequential with prefix (`BACK-1`, `BACK-10`, ...)
- **Storage**: Markdown files in `backlog/` directory
- **Hierarchy**: Parent-child via task ID references
- **Kanban**: Status-based columns (configurable)
- **MCP Integration**: Auto-configures for Claude Code, Codex, Gemini CLI, Kiro
- **Definition of Done**: Reusable checklist defaults
- **Web UI**: Drag-and-drop Kanban board

---

## Comparative Analysis

### Storage Models

| Project | Storage | Format | Concurrency | Git-Native | Speed |
|---------|---------|--------|-------------|-----------|-------|
| **Beads** | SQLite + JSONL | Binary + JSON | Dolt merge | ✅ Yes | Slow |
| **Ergo** | JSONL only | JSON | File lock | ❌ No | Fast |
| **Jari** | LMDB | Binary | LMDB txn | ❌ No | Fast |
| **Niwa** | LMDB | Binary | LMDB txn | ❌ No | Fast |
| **Backlog.md** | Markdown files | Text | File-based | ✅ Yes | Slow |

### Task Models

| Project | Hierarchy | Dependencies | Conflicts | Metadata | Messaging |
|---------|-----------|--------------|-----------|----------|-----------|
| **Beads** | Dots (epic.1.1) | Graph (5 types) | None | Rich | ✅ Yes |
| **Ergo** | Epic→Task | Linear | None | Minimal | ❌ No |
| **Jari** | Epic→Task | Linear | Field-level | Medium | ❌ No |
| **Niwa** | H1→H2→H3 | None | Version-based | Medium | ❌ No |
| **Backlog.md** | Parent→Child | None | None | Rich | ❌ No |

### Agent Integration

| Project | MCP | Claude Hooks | CLI-First | JSON Output | Ready Queue |
|---------|-----|--------------|-----------|-------------|-------------|
| **Beads** | ✅ Python | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes |
| **Ergo** | ❌ No | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes |
| **Jari** | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **Niwa** | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ❌ No |
| **Backlog.md** | ✅ Native | ❌ No | ✅ Yes | ✅ Yes | ❌ No |

---

## Patterns for a Slimmer Alternative

### Core Insights

1. **Append-Only JSONL is Optimal**
   - Ergo proves append-only JSONL is fast enough (replay is instant)
   - Simpler than Beads' 3-layer model
   - More portable than LMDB
   - Git-mergeable (reduces conflicts)

2. **File Lock > Dolt for Concurrency**
   - Ergo's `flock` is simpler and faster than Dolt
   - Sufficient for agent workflows (agents don't race as much as humans)
   - No daemon required

3. **Field-Level Conflict Detection is Valuable**
   - Jari's approach (track which fields changed) is more sophisticated than version-based
   - Auto-merge non-overlapping changes
   - Reduces manual conflict resolution

4. **Hierarchy Matters More Than Graph Links**
   - All projects support parent-child
   - Graph links (relates_to, duplicates) are rarely used
   - Epic→Task hierarchy is sufficient for most workflows

5. **Ready Queue is Essential**
   - All projects with agent focus have it
   - Surfaces unblocked work automatically
   - Atomic claim prevents race conditions

6. **Markdown-First is Underrated**
   - Backlog.md proves markdown files work well
   - Human-readable, git-friendly
   - Slower for large projects, but acceptable for most

### Recommended Architecture for Lightweight Alternative

```
┌─────────────────────────────────────────────────────────────┐
│                    Lightweight Task Tool                     │
├─────────────────────────────────────────────────────────────┤
│  Storage Layer                                               │
│  ├── Append-only JSONL (.tasks/events.jsonl)                │
│  ├── File lock for write serialization                       │
│  └── Optional: Git sync (like Beads)                         │
├─────────────────────────────────────────────────────────────┤
│  Task Model                                                  │
│  ├── Entities: Task, Epic                                    │
│  ├── State: todo, in_progress, done, blocked, error         │
│  ├── Dependencies: Linear (blocked_by)                       │
│  ├── Metadata: title, description, priority, assignee       │
│  └── Hierarchy: Epic→Task (dots like Beads)                 │
├─────────────────────────────────────────────────────────────┤
│  Features                                                    │
│  ├── Ready queue (unblocked tasks)                           │
│  ├── Atomic claim (race-safe assignment)                     │
│  ├── Field-level conflict detection (like Jari)             │
│  ├── Auto-merge for non-overlapping changes                 │
│  ├── Compaction (like Ergo)                                 │
│  └── Claude Code hooks (like Jari/Niwa)                     │
├─────────────────────────────────────────────────────────────┤
│  Interfaces                                                  │
│  ├── CLI (primary)                                           │
│  ├── JSON output (for agents)                                │
│  └── Optional: Web UI (like Backlog.md)                     │
└─────────────────────────────────────────────────────────────┘
```

### Compatibility Hooks with Beads

To remain compatible with Beads while being lighter:

1. **ID Format**: Support both hash-based (`bd-a1b2`) and sequential (`task-1`)
2. **Storage**: JSONL + optional Git sync (like Beads)
3. **Sync Model**: `sync` command for bidirectional git sync
4. **Ready Queue**: `ready` command returns unblocked tasks
5. **Claim Model**: Atomic claim via `update <id> --claim`
6. **Hierarchy**: Dot notation for epics (`epic-1.1`, `epic-1.2`)
7. **Dependencies**: Linear (blocked_by) with cycle detection
8. **Metadata**: title, description, priority, assignee, labels
9. **MCP Integration**: Support Claude Code hooks
10. **Compaction**: Summarize old closed tasks

---

## Key Takeaways

### What Works Well
- ✅ Append-only JSONL (fast, simple, git-friendly)
- ✅ File lock for concurrency (simpler than Dolt)
- ✅ Field-level conflict detection (sophisticated)
- ✅ Ready queue (essential for agents)
- ✅ Atomic claims (prevents race conditions)
- ✅ Claude Code hooks (context injection)
- ✅ Markdown-first (human-readable)

### What to Avoid
- ❌ Complex 3-layer models (Beads is overkill for most)
- ❌ Dolt (adds complexity, slower)
- ❌ Graph links (rarely used, adds complexity)
- ❌ Daemons (unnecessary for agent workflows)
- ❌ LMDB (less portable than JSONL)

### Beads Compatibility Strategy
- Use JSONL + Git sync (like Beads)
- Support dot notation for hierarchy (like Beads)
- Implement ready queue (like Beads)
- Support atomic claims (like Beads)
- Add Claude Code hooks (like Jari/Niwa)
- Keep ID format flexible (support both hash and sequential)
