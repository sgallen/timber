# AGENTS.md

This file is the working contract for agents contributing to the `timber` repo.

## Product context

Timber is a path manager for Git experiments and coding agents.

Use the product metaphor consistently:

- **Timber** is the product name.
- A **project** is the top-level container.
- A **path** is the isolated working unit.
- A **repo** is a registered Git repository that may participate in a path.

It is not just a thin `git worktree` wrapper. The differentiated product behavior is:

- sparse multi-repo paths
- mixed per-repo refs
- safe private branch creation
- forkable experiment paths
- keep/drop workflows
- context-aware commands from inside a path

## How to work here

Before making substantial changes:

1. Read this file.
2. Read `TASKS.md`.
3. Read `projects/timber-prd-v0.5.md`.
4. Read `projects/timber-review-notes-2026-06-21.md`.
5. Treat the PRD as the source of truth, except where the review notes identify unresolved ambiguities that need tightening.

## Workflow expectations

- Prefer small, reviewable slices over broad speculative rewrites.
- Keep implementation aligned with the product UX, not just Git internals.
- Shell out to Git rather than reimplementing Git behavior.
- Preserve the user-facing noun `path`.
- Do not introduce extra concepts unless they materially improve UX.
- Keep destructive operations conservative and explicit.
- Update `TASKS.md` when meaningful work starts, blocks, or finishes.
- On this machine, prefer `make fmt`, `make test`, and `make build` so Go uses the repo-local cache configuration.

## Early implementation priorities

- Normalize the spec enough that another agent can implement without inventing behavior.
- Keep generated files useful for both humans and coding agents.
- Make branch-collision avoidance and path metadata correctness first-class.
- Treat shell completion as an MVP feature, not optional polish.
- Build forward from the Go scaffold.

## Boundaries

- Do not broaden into hosted orchestration, PR management, or a GUI.
- Do not optimize for giant-monorepo performance before the core workflows work.
- Do not silently materialize all repos in multi-repo projects unless explicitly requested.
- Do not hide ambiguous behavior behind “smart defaults” where the PRD has not yet made the rule explicit.
- Do not spend time debating Python vs Go further unless a new constraint appears. Scott explicitly chose Go.
