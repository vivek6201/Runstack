---
name: graft
description: Codebase indexing and context graph via graft/. For ANY task in this repository—whether understanding architecture, finding where code lives, tracing call graphs/blast radius, or scoping an edit—consult the graft graph before grepping or reading raw source files.
---

# Graft: Codebase Indexing & Context Skill

`graft/` holds a structural graph of this repository: small markdown nodes that explain components in prose and link the exact `file:line` spans they cover, connected by typed symbol relationships.

## Core Rules

1. **Check Graft first:** Before running broad `grep`, `find`, or reading raw source files across the codebase, consult the local `graft/` graph.
2. **Targeted retrieval:** Fetch targeted context with structural queries rather than reading whole directory trees.
3. **Keep graph fresh:** If files are substantially changed or added, ensure the index is updated.

## Key CLI Commands

If the `graft` (or `trail`) CLI is installed:

- **Concept Search:**
  ```bash
  graft ask "<query>"
  ```
  Returns ranked conceptual matches with relevant code snippets inlined.

- **Symbol Search:**
  ```bash
  graft grep "<pattern>"
  ```
  Exhaustive pattern search grouped by symbol definitions.

- **Call Graph & Blast Radius:**
  ```bash
  graft callers "<symbol>"
  ```
  Traces exact callers and callees to calculate the impact radius of an edit.

- **File Skeleton:**
  ```bash
  graft skeleton <path/to/file>
  ```
  Displays the API signatures and exported types without the implementation body.

- **Repository Overview:**
  ```bash
  graft map
  ```
  High-level orientation map of the codebase architecture.

- **Freshness Check:**
  ```bash
  graft check
  ```
  Checks if the workspace has drifted from the indexed graph.

## Setup & Initialization

To index the repository for the first time:

```bash
npm install -g @trailhq/trail-code
# or run directly:
npx @trailhq/trail-code init
```
This scans the workspace with Tree-sitter and creates the local `graft/` directory.
