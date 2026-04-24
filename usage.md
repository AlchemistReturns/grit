# grít — Full Usage Reference

This document is the complete reference for every command, flag, configuration option, and behavior in grít.

---

## Table of Contents

1. [Installation](#installation)
2. [Initialization](#initialization)
3. [Commit interview](#commit-interview)
4. [Real-time file watcher](#real-time-file-watcher)
5. [Viewing your log](#viewing-your-log)
6. [Recording decisions](#recording-decisions)
7. [Revert post-mortems](#revert-post-mortems)
8. [End-of-day reflection](#end-of-day-reflection)
9. [Statistics and analytics](#statistics-and-analytics)
10. [Exporting data](#exporting-data)
11. [Configuration reference](#configuration-reference)
12. [Database reference](#database-reference)
13. [Git hooks reference](#git-hooks-reference)
14. [Answer tagging](#answer-tagging)
15. [How complexity is scored](#how-complexity-is-scored)
16. [Windows notes](#windows-notes)

---

## Installation

### Prerequisites

| Requirement | Notes |
|-------------|-------|
| Go 1.21+ | `go version` to check |
| GCC | Required for CGO / SQLite. See below. |

**GCC installation:**

- **Windows** — install via [MSYS2](https://www.msys2.org/):
  ```sh
  pacman -S mingw-w64-ucrt-x86_64-gcc
  ```
- **macOS** — `xcode-select --install`
- **Linux** — `sudo apt install build-essential` (or distro equivalent)

### Build from source

```sh
git clone https://github.com/alchemistreturns/grit
cd grit

# Windows — gcc must be on PATH
PATH="/c/msys64/ucrt64/bin:$PATH" go build -o grit.exe .

# macOS / Linux
go build -o grit .
```

Add the resulting binary to your PATH.

### go install

```sh
# Windows
PATH="/c/msys64/ucrt64/bin:$PATH" go install github.com/alchemistreturns/grit@latest

# macOS / Linux
go install github.com/alchemistreturns/grit@latest
```

> The first build takes 30–60 seconds. The SQLite C amalgamation is compiled once and then cached — subsequent builds are fast.

---

## Initialization

```sh
cd your-project
grit init
```

`grit init` must be run inside a git repository. It creates:

| Path | Description |
|------|-------------|
| `.grit.yaml` | Project-level config (question pool, thresholds, watch settings) |
| `~/.grit/store.db` | Global SQLite database shared across all your projects |
| `.git/hooks/pre-commit` | Runs `grit commit` before every commit |
| `.git/hooks/post-rewrite` | Detects revert commits and triggers post-mortems |

**Idempotency:** safe to run multiple times. If `.grit.yaml` already exists, it is left untouched. If a pre-commit hook already exists, the grít invocation is appended rather than replacing the file.

---

## Commit interview

The commit interview runs automatically every time you `git commit`. You never need to invoke it manually.

### How it works

1. grít reads the staged diff to understand the nature of the change.
2. It selects 2 questions from the configured pool, avoiding questions asked in the last N commits (the "window").
3. Each question is displayed as an interactive prompt with a 30-second timeout.
4. Answers are stored in `~/.grit/store.db` and optionally tagged.

### Context-aware questions

grít analyzes the staged diff and injects a context-specific question when it detects:

| Condition | Injected question |
|-----------|-------------------|
| Diff > 200 lines | "Should this commit be split into smaller pieces?" |
| New files added | "What's the contract / responsibility of the new file(s)?" |
| Only test files changed | "What edge cases are you not testing yet?" |
| Config files changed | "What value would break this if set incorrectly?" |

### Commit types that are silently skipped

grít reads the commit message and skips the interview for:

- Merge commits (`Merge branch …`, `Merge pull request …`)
- Fixup / squash commits (`fixup!`, `squash!`)
- WIP commits (message starts with `WIP` or `wip:`)
- Amend commits (`--amend` reuses the message, grít detects this)

### Revert commits

When grít detects a commit message starting with `Revert "…"`, it skips the normal interview and instead runs the **post-mortem interview** (3 questions). See [Revert post-mortems](#revert-post-mortems).

### TTY / CI behavior

If there is no TTY available (CI, piped commands, SSH without terminal allocation), grít detects this, stores a skipped event, and exits 0. Your pipeline never breaks.

### Exit code guarantee

`grit commit` **always exits 0**. A commit is never blocked by grít, even if grít itself panics.

---

## Real-time file watcher

```sh
grit watch
```

Run this in a second terminal while you code. grít watches your project directory recursively and prompts you when it detects friction signals.

### Watched file types

Controlled by `watch.extensions` in `.grit.yaml` (default: `.go`, `.js`, `.ts`, `.py`, `.rs`, `.java`, `.c`, `.cpp`).

### Excluded directories

`.git`, `node_modules`, `vendor` are always excluded.

### Debouncing

Each file path has a 200ms debounce. Burst saves (e.g. VSCode's rename-then-create pattern) are coalesced into a single analysis event.

### Friction signals

#### Complexity threshold

When a file's complexity score exceeds `thresholds.complexity` (default `10.0`), grít prints:

- The file's current score and your running average for that file
- A per-function breakdown showing which functions contribute most

No question is asked — this is informational.

#### Vague naming

When new lines (since the previous save) contain identifiers matching the weak-name list, grít shows a prompt:

```
Weak name detected: "handleData" in auth.go
→ What would a better name be?
```

The built-in list includes: `result`, `temp`, `tmp`, `data`, `stuff`, `thing`, `value`, `item`, `obj`, `res`, `resp`, `req`, and functions starting with `handle`, `process`, `do`, `get`, `set`, `manage`, `run`.

Additional per-language weak names can be configured in `.grit.yaml` under `watch.language_names`.

#### AI-assist prompt (15+ new lines)

When 15 or more new lines appear since the last save, grít waits 2 seconds and then asks:

```
→ Was any of this AI-assisted? What did you verify?
```

#### Paste comprehension prompt (30+ new lines)

When 30 or more new lines appear, grít asks:

```
→ Did you paste this? Do you fully understand what it does?
```

#### Undo spike prompt (20+ lines deleted)

When 20 or more lines are removed, grít asks:

```
→ Wrong turn or intentional cleanup?
```

#### Dead time prompt (40+ minutes in same file)

When grít detects multiple writes to the same file over a 40+ minute window without meaningful forward progress, it asks:

```
→ You've been in this file a while. Is the problem still clear?
```

All thresholds are configurable. See [Configuration reference](#configuration-reference).

---

## Viewing your log

```sh
grit log
```

Displays all friction events in reverse-chronological order, grouped by day. Each event shows:
- Time
- Event type (hook)
- Commit message snippet (for commit-time events)
- All questions and answers

### Filters

```sh
# filter by event type
grit log --hook interview
grit log --hook file_complexity
grit log --hook naming
grit log --hook ai_reflect
grit log --hook paste
grit log --hook undo_spike
grit log --hook dead_time
grit log --hook decision
grit log --hook revert

# show events from a date onward
grit log --since 2025-06-01

# show only events where you skipped all questions
grit log --skipped

# combine filters
grit log --hook interview --since 2025-06-01
```

### Event types

| Hook value | When it's recorded |
|------------|--------------------|
| `interview` | Commit-time friction interview |
| `file_complexity` | Complexity threshold exceeded |
| `naming` | Weak identifier detected |
| `ai_reflect` | 15+ lines added |
| `paste` | 30+ lines added |
| `undo_spike` | 20+ lines deleted |
| `dead_time` | 40+ min inactivity in file |
| `decision` | `grit decision` interview |
| `revert` | Revert commit post-mortem |

---

## Recording decisions

```sh
grit decision
```

Runs a structured 4-question interview to record an architectural decision:

1. **Context** — What situation or constraint forces this decision?
2. **Alternatives** — What alternatives did you evaluate?
3. **Decision** — What did you decide and why?
4. **Consequences** — What do you give up? What could go wrong?

Decisions are stored as regular events in the database with hook type `decision`.

### Listing decisions

```sh
grit decision list
```

Prints all recorded decisions with their dates and decision title (extracted from the decision answer).

### Exporting to ADR markdown

```sh
grit decision export
```

Exports each decision as an ADR (Architecture Decision Record) markdown file:

```
decisions/
├── 2025-06-01-use-sqlite-for-local-storage.md
├── 2025-06-15-cobra-for-cli-framework.md
└── ...
```

Each file is formatted with sections:
- **Context**
- **Options Considered**
- **Decision**
- **Consequences**

---

## Revert post-mortems

When you `git revert` a commit, the post-rewrite hook automatically triggers a 3-question post-mortem:

1. What went wrong with `<original commit message>`?
2. Was this caught in review or did it reach production?
3. What would have caught this earlier?

grít extracts the original commit hash from the revert message and stores it in `events.related_commit` so you can trace the revert back to its source.

### Manual invocation

You rarely need this, but you can also run:

```sh
grit revert --check
```

This is what the post-rewrite hook calls internally.

---

## End-of-day reflection

```sh
grit reflect
```

Shows your daily stats and asks 2 questions from the reflection pool:

**Stats shown:**
- Total friction events logged today
- Completed interviews vs. skipped
- Consecutive days streak (days with at least one completed interview)

**Reflection questions** (2 selected from the pool, avoiding recent repeats):
- "What problem did you think you were solving this morning versus what you actually solved?"
- "What would you tell a teammate who picks up this code tomorrow?"
- "What would you have done differently if you had one more hour?"
- "What's the most surprising thing you learned today?"
- "What decision today might you regret in 6 months?"
- "Where did your estimate go wrong today, and why?"

### Deep reflection export

If `deep_reflect.enabled: true` in `.grit.yaml`, answers are also written to a dated markdown file:

```
~/.grit/reflections/
├── 2025-06-01.md
├── 2025-06-02.md
└── ...
```

---

## Statistics and analytics

### Weekly summary

```sh
grit stats week
```

Shows the past 7 days:
- Total commits and interview completion rate
- Consecutive days streak
- Top friction tags with bar chart visualization
- Most complex files touched (peak complexity scores)

### File complexity trend

```sh
grit stats file path/to/file.go
```

Shows:
- Sparkline of complexity scores over time for that file
- All friction answers that mention the file

### Contribution heatmap

```sh
grit stats heatmap
```

12-week GitHub-style heatmap of friction density:

```
     Jun        Jul        Aug
Mon  ░ ░ ▒ ▓ █ ░ ░ ▒ ▒ ░ ░ ░
Wed  ░ ▒ ▒ ▓ ▒ ░ ▒ ▓ █ ▒ ░ ░
Fri  ░ ░ ░ ▒ ░ ░ ░ ▒ ▒ ░ ░ ░
```

| Character | Events/day |
|-----------|------------|
| `░` | 0 |
| `▒` | 1–2 |
| `▓` | 3–4 |
| `█` | 5+ |

### Tagged digest

```sh
grit stats digest
```

Groups all friction answers by their `[tag]` prefix. Untagged answers appear under `[general]`. Useful for identifying recurring themes.

---

## Exporting data

```sh
# export to Markdown
grit push --md

# export to JSON
grit push --json

# export a specific date range
grit push --md --since 2025-01-01
```

Output is written to `~/.grit/exports/grit-friction-{period}.{format}`.

**Markdown export:** organizes events by day with formatted question/answer pairs.

**JSON export:** array of event objects with ISO 8601 timestamps and nested answers.

Default range is the last month if `--since` is not specified.

---

## Configuration reference

grít looks for `.grit.yaml` in the current working directory. All sections and fields are optional — defaults apply when absent.

```yaml
# .grit.yaml — full example with all options

questions:
  # The pool of questions grít draws from at commit time.
  # You can add, remove, or rewrite any of these.
  pool:
    - "What's the hardest part of this change?"
    - "What would you do differently next time?"
    - "What assumption are you most uncertain about?"
    - "What did you learn that surprised you?"
    - "What would break first if this code were wrong?"
    - "What did you have to look up or re-learn to write this?"
    - "What's the least obvious thing about this diff?"
    - "If you came back to this in 6 months, what would confuse you?"
    - "What did you almost do instead?"
    - "What corner case are you intentionally ignoring?"
    - "What would make this change unnecessary?"
    - "What conversation should this commit have started?"
  # Number of recent questions to avoid repeating. Set to 0 to disable rotation.
  window: 5

thresholds:
  # Complexity score (keyword count) that triggers a watcher report.
  complexity: 10.0
  # Number of new lines added that triggers the AI-assist prompt.
  ai_reflect_lines: 15
  # Minutes of inactivity in a file before the dead-time prompt.
  dead_time_minutes: 40
  # Number of lines deleted that triggers the undo-spike prompt.
  undo_spike_lines: 20
  # Number of new lines that triggers the paste-comprehension prompt.
  paste_lines: 30

watch:
  # File extensions the watcher pays attention to.
  extensions:
    - .go
    - .js
    - .ts
    - .py
    - .rs
    - .java
    - .c
    - .cpp
  # Additional weak names to flag per language.
  # Keys must match the language name grít derives from the file extension.
  language_names:
    go:
      - result
      - tmp
      - data
      - obj
    python:
      - data
      - stuff
      - res
      - val

export:
  # Directory where `grit push` writes export files.
  path: "~/.grit/exports"

deep_reflect:
  # If true, `grit reflect` writes answers to a dated markdown file.
  enabled: false
  # Directory for dated reflection markdown files.
  output_dir: "~/.grit/reflections"
```

---

## Database reference

The SQLite database lives at `~/.grit/store.db` and is shared across all projects. It is opened in WAL mode with a 5-second busy timeout so `grit watch` and `grit commit` can write simultaneously without contention.

### `events` table

```sql
CREATE TABLE events (
    id             TEXT PRIMARY KEY,   -- 16-byte random hex ID
    hook           TEXT NOT NULL,      -- event type (see below)
    occurred_at    INTEGER NOT NULL,   -- Unix timestamp
    skipped        INTEGER DEFAULT 0,  -- 1 if all questions were skipped/timed out
    commit_msg     TEXT,               -- commit message (commit-time events only)
    related_commit TEXT               -- for revert events: hash of the reverted commit
);
```

**Hook values:** `interview`, `file_complexity`, `naming`, `ai_reflect`, `paste`, `undo_spike`, `dead_time`, `decision`, `revert`

### `answers` table

```sql
CREATE TABLE answers (
    id         TEXT PRIMARY KEY,
    event_id   TEXT NOT NULL REFERENCES events(id),
    question   TEXT NOT NULL,
    answer     TEXT NOT NULL,
    tag        TEXT DEFAULT ''   -- extracted from [tag] prefix in the answer text
);
```

### `complexity_history` table

```sql
CREATE TABLE complexity_history (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL,
    score       REAL NOT NULL,
    recorded_at INTEGER NOT NULL
);
```

### Querying the database directly

```sh
sqlite3 ~/.grit/store.db

# all events from the last 7 days
SELECT hook, occurred_at, commit_msg
FROM events
WHERE occurred_at > strftime('%s', 'now', '-7 days')
ORDER BY occurred_at DESC;

# tag frequency
SELECT tag, COUNT(*) as n
FROM answers
WHERE tag != ''
GROUP BY tag
ORDER BY n DESC;

# complexity trend for a file
SELECT score, recorded_at
FROM complexity_history
WHERE path LIKE '%auth.go'
ORDER BY recorded_at;
```

---

## Git hooks reference

### pre-commit

```sh
#!/bin/sh
if command -v grit >/dev/null 2>&1; then
    grit commit
fi
```

- Runs before every `git commit`
- If `grit` is not on PATH (e.g. a team member who hasn't installed it), the hook silently no-ops
- Always exits 0 — commits are never blocked

### post-rewrite

```sh
#!/bin/sh
if command -v grit >/dev/null 2>&1; then
    grit revert --check "$@"
fi
```

- Runs after `git commit --amend` and `git rebase`
- grít inspects the arguments to detect revert commits specifically
- Silently no-ops for non-revert rewrites

### Manually installing / repairing hooks

```sh
# re-run init — hooks are appended idempotently
grit init
```

If you need to remove grít from a hook, delete the block between the grít markers in `.git/hooks/pre-commit`.

---

## Answer tagging

Any answer can be prefixed with a `[tag]` to categorize the friction:

```
[debug]    spent 45 minutes on a nil pointer that a type assertion would have caught
[scope]    started with auth, ended up refactoring the whole session layer
[design]   not sure if this belongs in the handler or the service
[test]     no easy way to test this without a real database
```

Tags are extracted and stored in `answers.tag`. They surface in:
- `grit stats digest` — grouped by tag
- `grit stats week` — top tags bar chart
- `grit push --md` / `--json` exports

You can use any tag name. No predefined taxonomy is enforced.

---

## How complexity is scored

grít uses a keyword-counting heuristic that works across all supported languages without per-language parsers.

**Algorithm:**
1. Start with a base score of **1**
2. For each non-comment line, add 1 for each occurrence of:
   `if`, `else`, `for`, `switch`, `case`, `select`, `&&`, `||`, `catch`, `while`, `do`
3. Comment lines (starting with `//`, `#`, `*`, `--`) are skipped entirely

**Per-function breakdown (`ScoreByFunction`):**

grít also parses function boundaries using a multi-language detection approach:

| Language | Function detection |
|----------|--------------------|
| Go | `func ` prefix |
| Python | `def ` prefix |
| JavaScript / TypeScript | `function `, arrow functions |
| Rust | `fn ` prefix |
| Java / C# | method signatures |
| C / C++ | `{` after parameter list |

When the watcher detects a file above the threshold, it reports per-function scores so you can pinpoint the offending function rather than the whole file.

---

## Windows notes

- **GCC required** — install MSYS2 and run `pacman -S mingw-w64-ucrt-x86_64-gcc`
- **PATH for build/install** — `/c/msys64/ucrt64/bin` must be on PATH during `go build` and `go install`
- **TTY in git hooks** — bubbletea prompts use `CONIN$` on Windows via `tea.WithInputTTY()`, which works correctly when invoked from a git hook inside Windows Terminal, Git Bash, and VS Code's integrated terminal
- **Binary name** — build produces `grit.exe`; add the directory containing it to your Windows PATH or MSYS2 PATH

---

## Typical daily workflow

```sh
# morning: start your watcher
cd your-project
grit watch          # keep this running in a second terminal

# during the day: work normally
# grít prompts appear automatically at commits and on friction signals

# end of day
grit reflect        # review stats and answer 2 reflection questions
grit stats week     # see your weekly friction picture
grit log --since $(date +%Y-%m-%d)  # review today's events

# when you make a significant architectural choice
grit decision

# export everything for a retrospective
grit push --md --since 2025-06-01
```
