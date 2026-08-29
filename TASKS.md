# TASKS.md

Repo-local implementation backlog for `timber`.

---

## Agent guidance

- Read `AGENTS.md` first.
- Read the active PRD and review notes in `projects/` before acting.
- Keep tasks explicit, small, and reviewable.
- Update task status when meaningful work starts or finishes.

---

## Backlog

### In Progress

### Ready

### Blocked
- None.

### Done

#### WB-0040 Stop repo sync from pruning path branches
- **Outcome:** Reworked repo-cache syncing to use a safe bare-cache fetch layout instead of mirror semantics that could prune private `timber/...` path branches. Sync now fetches remote heads into `refs/remotes/origin/*`, preserves local private branches/worktrees across repeated syncs, and upgrades existing cache remote config on the fly. Also reordered ref resolution to prefer remote-tracking refs and added regression coverage proving a path stays clean after `timber repo sync`. Validated via `make fmt`, `make test`, `make build`, and a manual reproduction of the original sync-after-new workflow.

#### WB-0039 Add command-specific help output
- **Outcome:** Added command-specific `--help` output across the CLI so commands like `timber init`, `timber repo add`, `timber fork`, `timber keep`, `timber drop`, `timber status`, `timber info`, `timber dir`, `timber diff`, `timber run`, and `timber completion` are self-discoverable without falling back to top-level help. Also reused command-family help for missing/unknown `timber repo` cases. Validated via `make fmt`, `make test`, `make build`, and direct smoke checks for representative command help output.

#### WB-0038 Clarify path read-side UX
- **Outcome:** Tightened the read-side path UX so `timber ls` shows an accurate `SOURCE` summary for mixed-ref paths, `timber info` focuses on structural metadata (path, created time, source, per-repo starting refs/branches), and `timber status` focuses on live path state. Also replaced vague `dirty` status labels with clearer `modified` wording in path/repo summaries. Validated via `make fmt`, `make test`, `make build`, and a manual mixed-ref path smoke check.

#### WB-0037 Harden status/info against missing baseline commits
- **Outcome:** Hardened path ahead-count calculation so `timber status` and `timber info` no longer fail hard when a repo's recorded `source_commit` or `source_ref` is stale or missing. Timber now falls back safely instead of aborting the whole command, with regression coverage for invalid stored baseline metadata. Validated via `make fmt`, `make test`, and `make build`.

#### WB-0036 Add repo-subcommand help
- **Outcome:** Added `timber repo --help` with repo-specific subcommands, explanations, and examples, and reused that help for missing/unknown `timber repo` invocations so the repo command family is self-discoverable. Validated via `make fmt`, `make test`, `make build`, and a direct `timber repo --help` smoke check.

#### WB-0035 Add repo removal for trial-time cleanup
- **Outcome:** Added conservative `timber repo rm <name>` support so trial users can correct mistaken repo registrations without editing metadata by hand. Removal now refuses repos still used by any path, deletes the cached mirror when safe, updates default-repo bookkeeping, and is documented in top-level help plus the README. Validated via Go tests, `make fmt`, `make test`, `make build`, and a manual `timber repo rm` smoke check.

#### WB-0034 Fix `timber add` discoverability and mixed-ref path metadata bugs
- **Outcome:** Made `timber add` easier to discover by tightening top-level help, adding dedicated `timber add --help` usage/examples, and expanding the README with the register → sync → add flow. Also fixed path metadata refresh after mixed-ref repo additions so `timber ls`/`timber info` report the correct `from` summary, and hardened ahead-count reads so older metadata without stored source commits no longer breaks `timber info`/`timber status`. Validated via `make fmt`, `make test`, `make build`, and a manual mixed-ref add smoke check.

#### WB-0033 Polish repo registration and sync UX
- **Outcome:** Normalized `timber version` to a clean user-facing version string, moved syncing under `timber repo sync`, added `timber repo ls`, improved `timber repo add` to show unsynced state and next steps, and made repo syncing print immediate per-repo progress plus completion results. Updated the README to match the revised repo UX. Validated via `make fmt`, `make test`, `make build`, and manual repo add/list/sync smoke checks.

#### WB-0003 Implement Milestone 1 single-repo core
- **Outcome:** Closed the last obvious Milestone 1 gap by implementing `timber run [<path>] -- <command>` with path-root execution, inferred path context when run inside a path, `TIMBER_*` environment variables, updated help/completion/`timber here` guidance, and `PATH.md` command hints. Validated via `make fmt`, `make test`, and `make build`.

#### WB-0032 Polish README mental model and cohesion
- **Outcome:** Reworked the README mental model around the product/container/working-unit layers and tightened the whole document so it reads as one cohesive, product-facing story rather than a pile of accurate notes. Validated via `make fmt`, `make test`, and `make build`.

#### WB-0031 Refresh README voice and product story
- **Outcome:** Rewrote the README to better sell the product story, explain the mental model, show the fork/keep/drop workflow, and give a clearer install and try-it-now path without overstating maturity. Validated via `make fmt`, `make test`, and `make build`.

#### WB-0030 Add zsh completion support
- **Outcome:** Added `timber completion zsh` alongside the existing bash completion path, using the same repo/path/ref completion backends and documenting the zsh install flow in the README. Validated via `make fmt`, `make test`, `make build`, and completion output smoke checks for both shells.

#### WB-0029 Update docs and install path
- **Outcome:** Updated repo/module references from `github.com/scott/timber` to `github.com/sgallen/timber`, added a clear README install path for local binary install and `go install`, refreshed the bootstrap-status wording, and pointed the local git remote `origin` at `https://github.com/sgallen/timber.git`. Validated via `make fmt`, `make test`, `make test-race`, `make build`, and local `go install ./cmd/timber`.

#### WB-0019 Improve CLI-layer test coverage and Go QA checks
- **Outcome:** Added direct `internal/cli` tests for argument parsing and helper behavior around `timber new` / `timber add`, and added a repo-local `make test-race` target so lightweight Go QA is part of the normal workflow on this machine. Validated via `make fmt`, `make test`, `make test-race`, `make build`, and `go test ./... -cover`.

#### WB-0028 Polish lineage and recovery UX
- **Outcome:** Polished the read-side path UX by surfacing parent/child lineage and active keep-recovery guidance in `timber info` and `timber status`, plus a small context-aware `timber here` suggestion improvement. Validated via Go tests, `make fmt`, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0027 Implement bootstrap `timber drop`
- **Outcome:** Added bootstrap `timber drop <path>...` with conservative safety checks for dirty or ahead paths, child-lineage refusal unless `--recursive`, and branch-handling controls via `--force`, `--keep-branches`, and `--delete-branches`. Also updated CLI completion/help and the README command inventory. Validated via Go tests, `make fmt`, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0026 Implement bootstrap `timber keep`
- **Outcome:** Added bootstrap `timber keep [<child>] --into <target>` with source inference inside a child path, repo-by-repo merge keep behavior, and resumable `.timber/operations/keep.yaml` state plus `timber keep --continue` and `timber keep --abort`. Conflicts now stop safely with clear recovery instructions, and the README command inventory/completion guidance were refreshed. Validated via Go tests, `make fmt`, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0025 Implement bootstrap `timber fork`
- **Outcome:** Added bootstrap `timber fork [<source>] <child>...` with clean-source enforcement, source-path inference when run inside a path, child path creation from source `HEAD` commits, and stored parent/fork-point metadata needed for later `timber keep` work. Also updated `timber here` suggestions and the README command inventory. Validated via Go tests, `make fmt`, `make test`, `make build`, and a manual CLI smoke check.

#### WB-0024 Polish shell completion and install UX
- **Outcome:** Polished completion setup UX by clarifying CLI help, adding bash setup guidance directly to `timber completion bash`, and documenting current-shell plus persistent installation in the README. Also refreshed the README bootstrap command inventory so completion guidance stays grounded in the current product surface. Validated via `make fmt`, `make test`, `make build`, and CLI smoke checks.

#### WB-0023 Expand repo and ref completion
- **Outcome:** Expanded shell completion beyond path names by adding completion backends for registered repos and synced refs, plus bash completion wiring for `--repo`, `--repos`, and `repo=ref` flows in commands like `timber new`, `timber add`, and `timber diff`. Validated via Go tests, `make test`, `make build`, and completion smoke checks.

#### WB-0022 Implement `timber diff`
- **Outcome:** Added `timber diff [path] [--repo <name>]` with path inference, optional repo scoping, and per-repo diff sections that respect the path/repo model. Clean repos are shown explicitly as clean when included in scope. Validated via Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0021 Support multi-repo `timber status` and `timber info`
- **Outcome:** Extended `timber status` and `timber info` to summarize all repos in a path rather than assuming a single-repo view. The commands now show path-level rollups plus per-repo branch/source/status details, and the behavior was validated with new multi-repo tests, `make test`, `make build`, and CLI smoke checks.

#### WB-0020 Implement `timber add`
- **Outcome:** Added bootstrap `timber add` for expanding an existing path with additional registered repos using either a shared default ref or repo-specific `repo=ref` mappings. The command supports context-aware use from inside a path, updates path metadata and generated docs, and creates one private-branch-backed worktree per added repo. Validated via Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0018 Implement `timber new --all`
- **Outcome:** Added `timber new --all` so users can explicitly create a path from every registered repo while preserving the rule that multi-repo expansion must be explicit. Validated parsing/creation with Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0017 Implement mixed-ref path creation
- **Outcome:** Extended bootstrap `timber new` to accept explicit `repo=ref` mappings, both as mapping-only path creation and as overrides on top of `--from` plus `--repos`. Added validation for inclusion and registration rules, persisted per-repo source metadata, and validated via Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0016 Implement multi-repo `timber new --from --repos`
- **Outcome:** Extended bootstrap `timber new` to support explicit multi-repo selection with `--repos`, creating one private-branch-backed worktree per selected repo from a shared ref, plus multi-repo path metadata and generated docs. Added all-or-nothing validation for repo selection/ref resolution and validated via Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0015 Implement path-name completion
- **Outcome:** Added a completion backend for path names plus `timber completion bash` shell integration so commands like `timber info`, `timber status`, and `timber dir` can complete path names from the current project. Also documented the explicit-at-root plus completion UX rule in the README. Validated via Go tests, `make test`, `make build`, and completion smoke checks.

#### WB-0014 Implement bootstrap `timber info`
- **Outcome:** Added `timber info [path]` with path-name inference from the current directory, backed by path metadata plus current Git status for bootstrap single-repo paths. The output now summarizes path, creation time, source, repo branch, current status, and suggested resume commands. Validated via Go tests, `make test`, `make build`, and manual CLI smoke checks.

#### WB-0013 Tighten product terminology and metaphor
- **Outcome:** Updated the README, AGENTS guidance, and PRD to explicitly anchor the hierarchy: Timber is the product, a project is the top-level container, and a path is the isolated working unit. Also refreshed the README bootstrap command list to match the current CLI.

#### WB-0012 Polish bootstrap CLI UX
- **Outcome:** Clarified bootstrap help text for `timber repo`, `timber new`, and `timber status`; removed misleading unimplemented suggestions from `timber here`; and added direct `timber repo add` usage guidance for missing/unknown repo subcommands. Validated via `make test`, `make build`, and CLI smoke checks.

#### WB-0011 Implement bootstrap `timber dir`
- **Outcome:** Added `timber dir [path]` with path-name inference from the current directory, backed by path metadata lookup and absolute path output. Covered the helper with Go tests and validated via `make test`, `make build`, and manual CLI smoke checks.

#### WB-0010 Implement bootstrap `timber status`
- **Outcome:** Added `timber status [path]` with path-name inference from the current directory, backed by path metadata plus Git cleanliness/ahead checks for bootstrap single-repo paths. Covered clean/dirty+ahead cases with Go tests and validated via `make test`, `make build`, and manual CLI smoke checks.

#### WB-0009 Implement bootstrap `timber ls`
- **Outcome:** Added `timber ls` backed by path metadata discovery under `.timber/paths/`, with sorted bootstrap table output showing path name, source, repos, and path. Covered the listing behavior with Go tests and validated via `make test`, `make build`, and a manual CLI smoke check.

#### WB-0008 Implement single-repo `timber new --from`
- **Outcome:** Added bootstrap `timber new <name> --from <ref>` for single-repo projects, including private-branch-backed worktree creation, path metadata under `.timber/paths/`, generated `PATH.md` and `AGENTS.md`, and append-only path-created event logging. Covered the flow with Go tests and validated via `make test`, `make build`, and a manual CLI smoke check.

#### WB-0007 Implement `timber repo sync` bootstrap support
- **Outcome:** Added `timber repo sync` with project-root inference and Git-backed mirror clone/fetch behavior for registered repos under `.timber/repos/`. Covered clone/fetch behavior with local Git-backed tests and validated via `make test`, `make build`, and a manual CLI smoke check.

#### WB-0006 Implement `timber repo add` bootstrap support
- **Outcome:** Added `timber repo add` with project-root inference, duplicate-name validation, repo metadata persistence, and default-repo handling for single-repo vs multi-repo projects. Covered the behavior with Go tests and validated via `make test`, `make build`, and a manual CLI smoke check.

#### WB-0005 Harden Go bootstrap project initialization
- **Outcome:** `timber init` now rejects nested Timber projects and invalid non-file `.timber/events.log` paths, with regression coverage for both cases and validation via `make test` and `make build`.

#### WB-0004 Recreate the bootstrap in Go and make it the default entry path
- **Outcome:** Added a Go-first scaffold with `go.mod`, `cmd/timber/main.go`, core project metadata/init/context logic under `internal/`, and basic Go tests. Installed Go locally, validated the scaffold with `go test`, `go build`, and command-level smoke checks, and added a `Makefile` that uses repo-local Go caches on this machine.

#### WB-0002b Decide the long-term implementation language direction
- **Outcome:** Scott chose Go as the starting direction for the real product path.

#### WB-0002 Choose the initial implementation stack and scaffold the CLI
- **Outcome:** An earlier Python bootstrap existed briefly to validate the CLI shape before the repo pivoted fully to Go. That prototype has now been removed.

#### WB-0001 Tighten the spec where it is ambiguous enough to block clean implementation
- **Outcome:** PRD tightened to v0.5 with canonical layout, explicit `timber add` ref rules, safer `timber save` defaults, deterministic missing-ref behavior, and resumable `timber keep` operation-state requirements.

#### WB-0000 Initialize Timber repo scaffold
- **Outcome:** Created repo-local README, agent instructions, task tracker, imported PRD, and initial review notes so another agent can start from a durable local project home.
