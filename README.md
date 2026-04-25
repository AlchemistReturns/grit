# grít

<div align="center">

  <a href="https://github.com/alchemistreturns/grit">
    <img
      src="public/icon.svg"
      alt="Grít"
      width="120"
      height="120"
    />
  </a>

  <h2>Friction where it matters; data where you own it</h2>

  <h3><strong>Grít captures the thinking behind your code—not just the diffs</strong></h3>

  <p>
    <strong>grít</strong> is a <strong>CLI</strong> that turns your commits into a <strong>knowledge base</strong> on disk: small, purposeful friction at the moments that matter - a <strong>commit</strong>, a <strong>dependency</strong> change, or a <strong>revert</strong>. You get context-aware prompts, a real-time <code>watch</code>, <code>decision</code> records, <code>stats</code>, and exports. All of it lives in a SQLite file under <code>.grit/</code> in your repository
    <br>
    <strong>no cloud required</strong>.
  </p>

  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.21 or newer" /></a>
    &nbsp;
    <a href="https://www.sqlite.org/"><img src="https://img.shields.io/badge/SQLite-local%20%28WAL%29-003B57?logo=sqlite&logoColor=white" alt="SQLite" /></a>
    &nbsp;
    <a href="https://git-scm.com/"><img src="https://img.shields.io/badge/Git-hooks-F05032?logo=git&logoColor=white" alt="Git hooks" /></a>
    &nbsp;
    <img src="https://img.shields.io/badge/scope-per%20repository-3f3f46" alt="Per-repository" />
  </p>

  <p>
    <a href="#website">Website</a>
    &nbsp;·&nbsp;
    <a href="#what-it-does">What it does</a>
    &nbsp;·&nbsp;
    <a href="#prerequisites">Prerequisites</a>
    &nbsp;·&nbsp;
    <a href="#installation">Install</a>
    &nbsp;·&nbsp;
    <a href="#quick-start">Quick start</a>
    &nbsp;·&nbsp;
    <a href="#commands">Commands</a>
    &nbsp;·&nbsp;
    <a href="#configuration">Configuration</a>
    &nbsp;·&nbsp;
    <a href="#file-structure">File structure</a>
  </p>

</div>

---

## Website

A companion project runs the **official site** in the browser describing the story, how it works, feature tour, and install CTAs, and full documentation.

|                 |                                                                                                                                                 |
| :-------------- | :---------------------------------------------------------------------------------------------------------------------------------------------- |
| **Live**        | [**grit-cli.vercel.app**](https://grit-cli.vercel.app/) — home page, and [**/docs**](https://grit-cli.vercel.app/docs) - complete documentation |
| **Source Code** | [**github.com/rawadhossain/grit-cli**](https://github.com/rawadhossain/grit-cli)                                                                |

Use the site for **discovery and the full /docs experience**; use this README for **install, commands, and reference** in your editor and at the terminal.

---

## What it does

**At every commit**, grít intercepts the pre-commit hook and asks adaptive questions from a rotating pool:

- Questions are context-aware: large diffs get split-commit prompts, new files get contract questions, test-only commits get edge-case questions
- Answers can be tagged with `[tag]` prefixes for categorization
- Revert commits automatically trigger a 3-question post-mortem

**While you write code**, `grit watch` monitors your files and surfaces friction in real time:

- Files above a complexity threshold trigger a per-function score breakdown
- Vague identifiers (`handleData`, `tmp`, `result`) trigger a language-aware naming prompt
- Pasting 30+ lines triggers a question about whether you understood what you pasted
- Adding 15+ lines triggers a question about AI assistance
- Deleting 20+ lines triggers a question about wrong turns vs. cleanup
- 40+ minutes in the same file without progress triggers a focus check-in

**Beyond commits and watching**, grít offers:

- `grit decision` — record architectural decisions (ADR-style) with structured interviews
- `grit reflect` — end-of-day reflection with daily stats and optional markdown export
- `grit stats` — weekly analytics, file complexity trends, heatmaps, and tagged digests
- `grit push` — export your friction data to Markdown or JSON

All answers are stored locally in a SQLite database at `.grit/store.db` inside each project.

---

## Prerequisites

- **Go 1.21+**
- **GCC** (required for CGO / SQLite driver)
    - Windows: install via [MSYS2](https://www.msys2.org/) — `pacman -S mingw-w64-ucrt-x86_64-gcc`
    - macOS: `xcode-select --install`
    - Linux: `sudo apt install build-essential` or equivalent

---

## Installation

### From source

```sh
git clone https://github.com/alchemistreturns/grit
cd grit

# Windows (MSYS2 gcc)
PATH="/c/msys64/ucrt64/bin:$PATH" go build -o grit.exe .

# macOS / Linux
go build -o grit .
```

Add the binary to your PATH, then run `grit init` inside any git repository.

### go install

```sh
# Windows
PATH="/c/msys64/ucrt64/bin:$PATH" go install github.com/alchemistreturns/grit@latest

# macOS / Linux
go install github.com/alchemistreturns/grit@latest
```

> **Note:** The first build takes 30–60 seconds — this is normal. The SQLite C amalgamation is being compiled once and then cached.

---

## Quick start

```sh
cd your-project
grit init

# make a change, then commit
git add .
git commit -m "feat: add auth"
# grít prompts you with adaptive questions in the terminal

# view your friction timeline
grit log

# start the real-time watcher in a second terminal
grit watch

# end-of-day reflection
grit reflect

# see weekly analytics
grit stats week
```

---

## Commands

### `grit init`

Initializes grít in the current git repository.

```sh
grit init
```

Creates:

- `.grit.yaml` — project-level configuration with question pool, thresholds, and watch settings
- `.grit/store.db` — per-repository SQLite database (WAL mode)
- `.git/hooks/pre-commit` — calls `grit commit` before every commit
- `.git/hooks/post-rewrite` — detects revert commits and triggers post-mortems
- `.git/hooks/post-commit` — records the commit hash against the interview event

Safe to re-run. Existing hooks are appended to, not overwritten. Existing `.grit.yaml` is not touched.

---

### `grit commit`

Runs the adaptive friction interview. Called automatically by the git pre-commit hook.

```sh
grit commit
```

- Reads the staged diff to ask context-aware questions
- Draws from a rotating question pool (configurable, avoids recently-asked questions)
- Skips silently for merge commits, fixups, squashes, WIP, and amend commits
- Detects missing TTY and skips gracefully — CI never breaks
- Triggers a 3-question post-mortem for revert commits
- Always exits with code 0

**Answer tagging:** prefix any answer with `[tag]` to categorize it:

```
[debug] spent 45 minutes on a nil pointer that a type check would have caught
```

---

### `grit snooze` / `grit disable` / `grit resume`

Temporarily or permanently pause friction interviews — useful when you need to push rapidly without interruption.

```sh
grit snooze          # pause for 1 hour (default)
grit snooze 30m      # pause for 30 minutes
grit snooze 2h30m    # pause for 2.5 hours
grit disable         # pause indefinitely
grit resume          # re-enable interviews
```

While paused, `grit commit` records a skipped event and exits immediately — commits are never blocked. Pause state is stored in `.grit/pause` and expires automatically when a snooze duration elapses.

---

### `grit log`

Displays your friction timeline in reverse-chronological order, grouped by day.

```sh
grit log

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

# show events from a specific date onward
grit log --since 2025-01-15

# show only skipped events
grit log --skipped

# look up the interview linked to a specific commit
grit log --commit abc1234
```

| Flag              | Description                                                        |
| ----------------- | ------------------------------------------------------------------ |
| `--hook <type>`   | Filter by event type (see list above)                              |
| `--since <date>`  | Show events on or after this date (`YYYY-MM-DD`)                   |
| `--skipped`       | Show only events where the prompt was skipped or timed out         |
| `--commit <hash>` | Show the interview linked to a specific commit hash (prefix match) |

---

### `grit watch`

Starts the real-time file watcher. Run in a second terminal while you code.

```sh
grit watch
```

| Trigger                            | Default threshold | Action                                          |
| ---------------------------------- | ----------------- | ----------------------------------------------- |
| File save                          | —                 | Debounced 200ms, then analyzed                  |
| Complexity score exceeds threshold | `10.0`            | Prints per-function breakdown + running average |
| Vague identifier in new lines      | Any match         | Naming prompt (30s timeout)                     |
| Lines added                        | ≥ 15              | AI-assist prompt after 2s delay                 |
| Lines added                        | ≥ 30              | Paste-comprehension prompt                      |
| Lines deleted                      | ≥ 20              | Undo-spike prompt (wrong turn or cleanup?)      |
| Inactivity in same file            | ≥ 40 min          | Focus check-in prompt                           |

Watches subdirectories recursively, skipping `.git`, `node_modules`, and `vendor`. All thresholds are configurable in `.grit.yaml`.

---

### `grit decision`

Records an architectural decision with a structured 4-question interview.

```sh
grit decision
```

Questions:

1. What situation or constraint forces this decision?
2. What alternatives did you evaluate?
3. What did you decide and why?
4. What do you give up? What could go wrong?

#### `grit decision list`

Lists all recorded architectural decisions with dates.

#### `grit decision export`

Exports decisions to ADR-formatted markdown files in `decisions/YYYY-MM-DD-slug.md`.

---

### `grit revert`

Records a post-mortem for a reverted commit. Triggered automatically by the post-rewrite hook.

```sh
# called automatically — you rarely need to run this manually
grit revert --check
```

Questions:

1. What went wrong with the original commit?
2. Was this caught in review or did it reach production?
3. What would have caught this earlier?

---

### `grit reflect`

End-of-day reflection. Shows daily stats and asks 2 questions drawn from the reflection pool.

```sh
grit reflect
```

Displays:

- Total events logged today, completed vs. skipped interviews
- 2 reflection questions from a rotating pool
- Optionally writes answers to `.grit/reflections/YYYY-MM-DD.md` (if `deep_reflect.enabled: true`)

---

### `grit stats`

Analytics subcommands.

```sh
# past 7 days summary
grit stats week

# complexity trend for a specific file
grit stats file path/to/file.go

# 12-week contribution heatmap
grit stats heatmap

# friction digest grouped by tags
grit stats digest
```

#### `grit stats week`

- Total commits and interview completion rate
- Consecutive days streak
- Top friction tags (bar chart)
- Most complex files touched (peak scores)

#### `grit stats file <path>`

- Sparkline of complexity over time
- All friction notes mentioning the file

#### `grit stats heatmap`

- 12-week GitHub-style heatmap of friction density
- `░` = 0, `▒` = 1–2, `▓` = 3–4, `█` = 5+ events/day

#### `grit stats digest`

- All friction answers grouped by `[tag]`
- Untagged answers appear under `[general]`

---

### `grit push`

Exports friction data to Markdown or JSON.

```sh
grit push --md
grit push --json
grit push --md --since 2025-01-01
```

| Flag             | Description                                        |
| ---------------- | -------------------------------------------------- |
| `--md`           | Export to Markdown                                 |
| `--json`         | Export to JSON                                     |
| `--since <date>` | Export from this date onward (default: last month) |

Output is written to `.grit/exports/grit-friction-{period}.{format}`.

---

### `grit remove`

Removes grít hooks and configuration from the current repository.

```sh
grit remove          # remove hooks and .grit.yaml
grit remove --all    # also delete the entire .grit folder (database included)
```

| Flag    | Description                                                            |
| ------- | ---------------------------------------------------------------------- |
| `--all` | Completely remove the `.grit` directory, including the SQLite database |

- Cleanly unregisters grít from all three hooks (`pre-commit`, `post-rewrite`, `post-commit`) without touching any other hook logic you may have
- Removes `.grit.yaml`
- Without `--all`, the `.grit/` folder (database, exports, reflections) is preserved
- Safe to run even if hooks were never installed

---

## Configuration

grít looks for `.grit.yaml` in the current directory. All fields are optional.

```yaml
# .grit.yaml

questions:
    pool:
        - "What's the hardest part of this change?"
        - "What would you do differently next time?"
        - "What assumption are you most uncertain about?"
        - "What did you learn that surprised you?"
        # ... add your own
    window: 5 # how many recent questions to avoid repeating

thresholds:
    complexity: 10.0 # complexity score that triggers a report
    ai_reflect_lines: 15 # new lines that trigger AI-assist prompt
    dead_time_minutes: 40 # inactivity before focus check-in
    undo_spike_lines: 20 # deleted lines that trigger undo-spike prompt
    paste_lines: 30 # new lines that trigger paste-comprehension prompt

watch:
    extensions:
        - .go
        - .js
        - .ts
        - .py
        - .rs
        - .java
        - .c
        - .cpp
    language_names:
        go: ["result", "tmp", "data", "obj"]
        python: ["data", "stuff", "res", "val"]

export:
    path: ".grit/exports"

deep_reflect:
    enabled: false
    output_dir: ".grit/reflections"
```

---

## File structure

```
grit/
├── main.go
├── .grit.yaml
├── go.mod / go.sum
│
├── cmd/
│   ├── root.go           cobra root, Execute()
│   ├── init.go           grit init
│   ├── commit.go         grit commit — always exits 0
│   ├── log.go            grit log + filters (incl. --commit hash lookup)
│   ├── watch.go          grit watch — debounce, channel event loop
│   ├── decision.go       grit decision / list / export
│   ├── revert.go         grit revert --check
│   ├── reflect.go        grit reflect
│   ├── stats.go          grit stats week / file / heatmap / digest
│   ├── push.go           grit push --md / --json
│   └── remove.go         grit remove [--all]
│
└── internal/
    ├── config/
    │   └── config.go         viper config, GritDir(), DBPath()
    ├── store/
    │   ├── store.go           SQLite open + WAL schema migration
    │   ├── events.go          InsertEvent, QueryEvents, Filter
    │   └── answers.go         InsertAnswer, QueryAnswers, InsertComplexity,
    │                          AvgComplexity, RecentQuestions, TagCounts, etc.
    ├── hooks/
    │   └── installer.go       idempotent pre-commit + post-rewrite hook writer
    ├── prompt/
    │   ├── single.go          bubbletea single-line prompt (TTY, timeout, Esc)
    │   └── interview.go       sequential multi-question driver
    └── analysis/
        ├── complexity.go      Score(content), ScoreByFunction(content)
        └── naming.go          FindWeakName, FindWeakNameWithExtra, DiffLines
```

---

## Database schema

`.grit/store.db` — WAL mode, 5-second busy timeout (watch and commit can write simultaneously).

```sql
CREATE TABLE events (
    id             TEXT PRIMARY KEY,   -- 16-byte random hex
    hook           TEXT NOT NULL,      -- interview | file_complexity | naming |
                                       -- ai_reflect | paste | undo_spike |
                                       -- dead_time | decision | revert
    occurred_at    INTEGER NOT NULL,   -- Unix timestamp
    skipped        INTEGER DEFAULT 0,
    commit_msg     TEXT,
    commit_hash    TEXT,               -- SHA of the commit (set by post-commit hook)
    related_commit TEXT               -- for revert events: hash of reverted commit
);

CREATE TABLE answers (
    id         TEXT PRIMARY KEY,
    event_id   TEXT NOT NULL REFERENCES events(id),
    question   TEXT NOT NULL,
    answer     TEXT NOT NULL,
    tag        TEXT DEFAULT ''         -- extracted from [tag] prefix in answer
);

CREATE TABLE complexity_history (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL,
    score       REAL NOT NULL,
    recorded_at INTEGER NOT NULL
);
```

---

## Git hooks

`grit init` installs three hooks, all minimal and safe:

**pre-commit** — friction interview on every commit:

```sh
#!/bin/sh
if command -v grit > /dev/null 2>&1; then
    grit commit
fi
```

**post-rewrite** — post-mortem on reverts:

```sh
#!/bin/sh
if command -v grit > /dev/null 2>&1; then
    grit revert --check "$@"
fi
```

**post-commit** — records the commit hash against the interview:

```sh
#!/bin/sh
if command -v grit > /dev/null 2>&1; then
    grit post-commit
fi
```

This links each friction interview to the exact commit SHA, enabling `grit log --commit <hash>` lookups.

If `grit` is not on PATH, hooks silently no-op. Your commits are never blocked.

---

## Windows notes

- Requires GCC from MSYS2 (UCRT64): `pacman -S mingw-w64-ucrt-x86_64-gcc`
- Build and install commands must have `/c/msys64/ucrt64/bin` on `PATH`
- bubbletea prompts use `CONIN$` on Windows via `tea.WithInputTTY()` — works correctly inside git hook context

---

## How complexity is scored

grít uses a keyword-counting heuristic that works across all supported languages without per-language parsers:

- Base score: **1**
- Each occurrence of `if`, `else`, `for`, `switch`, `case`, `select`, `&&`, `||`, `catch`, `while`, `do` adds **1**
- Comment lines are skipped
- `ScoreByFunction` breaks the total down per function for pinpointing the offender

This approximates cyclomatic complexity well enough to surface genuinely tangled code.

---

## Minimum viable demo

```sh
cd /tmp/demo-project && git init
grit init

echo 'package main\nfunc main() {}' > main.go
git add .
git commit -m "feat: initial commit"
# answer the interview questions

grit log
grit stats week
```
