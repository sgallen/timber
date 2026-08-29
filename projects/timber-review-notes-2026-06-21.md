# Timber review notes

Date: 2026-06-21

These notes capture the first implementation-blocking ambiguities found while reviewing the draft that became `timber-prd-v0.5.md`.

## Status after v0.5 tightening

The highest-risk ambiguities from this note were resolved directly in `timber-prd-v0.5.md`:

- canonical metadata/layout location
- `timber add` default-ref behavior
- safer `timber save` default staging behavior
- missing-ref behavior during multi-repo path creation
- `timber keep` resumable operation-state requirements

The rest of this file is retained as historical review context.

## Key findings

### 1. Canonical layout is inconsistent

The PRD currently mixes at least two layouts:

- metadata under `.timber/`
- `project.yaml` and `events.log` at the project root

This must be normalized before implementation proceeds far, otherwise commands and tests will drift.

### 2. `timber add` ref resolution is incomplete

The spec says `timber add` can reuse the path default ref if available, but that is not deterministic enough for:

- paths created from mixed refs
- forked paths
- later-added repos that were never part of the original path

The repo needs one explicit rule for how a default ref is inherited or when the command must fail and require `--from` or `repo=ref`.

### 3. `timber save` is too aggressive by default

The current draft says to stage tracked and untracked files by default.

That is risky for:

- `.env` files
- editor config
- build outputs
- caches
- accidental local artifacts

Safer likely default:

- tracked changes only by default
- explicit `--all` for untracked inclusion

### 4. Missing branch behavior is still open, but it is core behavior

The PRD needs a deterministic rule for `timber new ... --from develop --repos ...` when one included repo does not have `develop`.

Possible choices:

- fail the whole operation
- allow partial creation with rollback rules
- fall back to repo defaults
- require explicit per-repo overrides

This should not be left to implementation taste.

### 5. `timber keep` recovery state needs more detail

The PRD mentions stopping on conflict and supporting `timber keep --continue` and `timber keep --abort`, which is directionally right.

What is still needed:

- exact operation-state file shape
- which repos are marked already merged
- how resume works after a process interruption
- what abort means after some repos already merged cleanly
- how metadata and generated docs should represent partial progress

## Recommended immediate next move

Before significant code generation, tighten the spec around the five issues above. That can happen either by updating the PRD directly or by creating a repo-local implementation-rules doc that temporarily closes the gaps.
