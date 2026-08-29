# Timber PRD

**Subtitle:** Forkable Git paths for humans and coding agents

| Field | Value |
|---|---|
| Product name | Timber |
| CLI commands | `timber` primary, `timber` long form |
| Core concept | path |
| Document type | Product Requirements Document and implementation brief |
| Status | Draft v0.5 |
| Date | 2026-06-21 |
| Audience | coding agent, implementation engineer, product/UX reviewer |


## 1. Executive summary

Timber is a command-line tool for creating isolated Git paths for humans and coding agents. It supports both single-repo and multi-repo projects. It is powered by Git bare repositories and Git worktrees, but users should not need to understand or remember raw `git worktree` commands.

The core product promise is:

> **Create isolated Git paths. Fork experiments from any clean path. Keep the winner. Drop the rest. Works for one repo or many.**

### 1.1 Product metaphor and hierarchy

The product name and the in-product nouns should stay distinct:

- **Timber** is the product.
- A **project** is the top-level container Timber manages.
- A **path** is the isolated working unit inside a project.
- A **repo** is a registered Git repository that may or may not be materialized in a given path.

The mental model should read naturally:

- Timber manages a project.
- A project knows about many repos.
- A path is a task-specific combination of one or more repos.

This framing is important because it avoids awkward language like "a timber contains timberes" while preserving the user-facing noun **path** as the central unit of work.

The user-facing concept is a **path**.

A path is an isolated checkout of one or more repos. It can be used for review, manual editing, agent work, debugging, branch-combination testing, or experiments. Timber should not split these into separate concepts such as "view," "agent path," or "state." Those are use cases of the same thing.

A project may register many repos in `.timber/repos/`, but each path may materialize any subset of them. For example, a project may have 10 registered repos while one path contains only 3, another contains 5, and another contains all 10. The registered repo catalog is not the same as path membership.

A path can also grow over time. A user may start with three repos, realize that two more repos are needed for context or changes, register those repos if necessary, and add them to the existing path without recreating it.

Example:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
timber run auth-flow -- codex
timber save auth-flow "baseline auth flow works"
timber fork auth-flow try-api try-ui try-auth-only
timber run try-auth-only -- claude
timber keep try-auth-only --into auth-flow
timber drop try-api try-ui
timber publish auth-flow
```

Internally, every path repo gets a private generated branch. The user may say "use `develop`," but Timber creates a unique branch such as `timber/auth-flow/backend-91ac` from `origin/develop`. This avoids the Git worktree rule that a branch cannot normally be checked out in multiple worktrees.


## 2. Positioning

Timber is **not** primarily "a better `git worktree` wrapper." That market is already active.

Timber should be positioned as:

> **A path manager for Git experiments and coding agents.**

Or:

> **Forkable Git paths for humans and coding agents, powered by worktrees.**

The differentiator is not merely creating a worktree. The differentiator is composing, forking, running, keeping, dropping, and resuming isolated paths across one or many repos.

### 2.1 Why `timber` / `timber`

The product name **Timber** evokes a place where work happens, where multiple pieces can be laid out side by side, and where tools are available. It fits manual work and agent work equally well.

The short command should be `timber`:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
timber fork auth-flow try-a try-b try-c
timber run try-a -- codex
timber keep try-a --into auth-flow
```

The long command should also be installed as `timber` for discoverability:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
```

If there is a command-name conflict, the install process should detect it and offer alternatives such as `git-timber` or installing only the long form.


## 3. Problem statement

Git worktrees are powerful, but the default UX has high cognitive overhead:

- Raw `git worktree add` syntax is hard to remember.
- Branch checkout collisions are confusing.
- Multi-repo work requires repeated commands and manual coordination.
- Coding agents benefit from seeing related repos, but shared directories are unsafe.
- Starting many agents from the same state is awkward.
- Coming back hours or days later is difficult: users need to reconstruct what each directory is, what it started from, and what changed.
- New worktrees often lack local setup: `.env`, dependencies, caches, editor settings, agent settings, and dev-server ports.

Timber should make these workflows feel simple:

```bash
timber new login-fix --from develop
timber new review-auth frontend=main backend=dev auth=hotfix/123
timber add review-auth audit=main
timber fork review-auth try-api try-ui try-auth
timber run try-api -- codex
timber keep try-auth --into review-auth
```


## 4. Goals

1. Provide a low-cognitive-load CLI for isolated Git paths.
2. Support single-repo and multi-repo workflows with the same commands.
3. Allow mixed repo/ref paths, such as `frontend=main backend=dev auth=hotfix/123`.
4. Allow multiple agents to work safely from the same starting point.
5. Allow a path to evolve, then be forked into multiple experiment paths.
6. Let the user keep or reject experiment results.
7. Avoid worktree branch collisions by managing private branches internally.
8. Generate human- and agent-readable path memory files.
9. Make resuming easy after hours or days away.
10. Provide shell completion that makes repos and branches discoverable.
11. Let commands infer the current project, path, and repo when the user is already inside a path.
12. Let an existing path be expanded with additional registered repos when the task scope grows.
13. Keep destructive operations conservative and explicit.


## 5. Non-goals for v0

- Replacing Git.
- Replacing GitHub/GitLab pull request workflows.
- Building a full agent orchestration UI.
- Automatically resolving complex merge conflicts.
- Supporting dirty path fork without an explicit save/checkpoint step.
- Implementing every possible Git state or advanced refspec behavior.
- Being optimized for giant monorepos on day one.
- Solving package manager cache isolation perfectly on day one.


## 6. Core mental model

Timber has three main user-facing nouns.

### 6.1 Project

A project is a directory managed by Timber.

It contains:

- one or more registered repos
- a `.timber/repos/` bare repo cache
- a `.timber/` metadata directory
- a `paths/` directory containing user paths

### 6.2 Repo

A repo is a Git repository registered with the project.

A project may contain one repo or many repos. Registered repos live in the project-level repo catalog and bare repo cache, but they are not automatically present in every path.

### 6.3 Path

A path is an isolated checkout of one or more repos selected from the project's registered repo catalog.

A path may be used for:

- reviewing
- manual coding
- coding-agent work
- testing
- debugging
- branch-combination validation
- forked experiments

There should be no separate "view" or "agent path" concept. A path can be used read-only, manually edited, or handed to an agent.


### 6.4 Repo catalog vs path membership

This is a first-class requirement.

A Timber project may have many registered repos under `.timber/repos/`, but a path should include only the repos needed for that task.

Example:

```text
myproject/
 .timber/repos/
 frontend.git
 backend.git
 auth.git
 billing.git
 search.git
 notifications.git
 analytics.git
 docs.git
 infra.git
 mobile.git

 paths/
 auth-flow/ # contains frontend, backend, auth
 billing-review/ # contains frontend, backend, billing, analytics, docs
 full-develop/ # contains all 10 repos
```

The UX implication is:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth
timber new billing-review --from main backend=dev billing=hotfix/123 --repos frontend,backend,billing,analytics,docs
timber new full-develop --from develop --all
```

Rules:

- `.timber/repos/` is the project repo catalog/cache.
- `paths/<name>/` contains only the repos selected for that path.
- `timber status`, `timber info`, `timber run`, `timber save`, `timber fork`, `timber keep`, `timber publish`, and `timber drop` operate only on repos that belong to that path unless a command explicitly targets project-level repo configuration.
- A path fork inherits the exact repo subset of its parent by default.
- A path may be expanded later with additional registered repos using `timber add`. This supports the common case where a user or agent realizes that more repo context is needed partway through the work.
- Removing repos from an existing path can wait until after MVP; adding repos is the higher-value direction because it preserves ongoing work while expanding context.



### 6.5 Context-aware commands

Timber should feel natural from inside a path. If the current working directory is inside a Timber project, path, or path repo, commands should infer as much context as possible.

Examples from inside `paths/auth-flow/` or `paths/auth-flow/backend/`:

```bash
timber status
timber info
timber save "checkpoint before experiments"
timber fork try-api try-ui try-auth
timber run -- codex
timber diff --repo backend
```

These should be equivalent to the explicit project-root forms:

```bash
timber status auth-flow
timber info auth-flow
timber save auth-flow "checkpoint before experiments"
timber fork auth-flow try-api try-ui try-auth
timber run auth-flow -- codex
timber diff auth-flow --repo backend
```

Inference rules:

- Starting from the current directory, walk upward to find `.timber/project.yaml`.
- If the path is under `paths/<name>/`, infer the current path.
- If the path is under `paths/<name>/<repo>/`, infer the current repo where applicable.
- If a command needs a path and no path can be inferred, fail with a helpful message.
- Explicit command arguments always override inferred context.

This is a major UX requirement. Developers will often be several directories deep in a repo when they realize they want to save, fork, run an agent, or inspect status.


## 7. Directory layout

Recommended layout:

```text
myproject/
 .timber/repos/
 frontend.git
 backend.git
 auth.git

 .timber/
 project.yaml
  paths/
   auth-flow.yaml
   try-api.yaml
   try-ui.yaml
  operations/
  events.log

 paths/
  auth-flow/
   frontend/
   backend/
   auth/
   PATH.md
   AGENTS.md

  try-api/
   frontend/
   backend/
   auth/
   PATH.md
   AGENTS.md
```

The user works inside `paths/<name>/`.

Timber owns `.timber/repos/` and `.timber/`. The `.timber/repos/` directory may contain more repos than any specific path uses. Path directories should be sparse by intent: they contain the repos selected for that path, not necessarily every registered repo.

Canonical rule for v0:

- `.timber/project.yaml` is the project root metadata file.
- `.timber/paths/<name>.yaml` stores path metadata.
- `.timber/events.log` is the append-only project event log.
- `.timber/operations/` stores resumable operation state such as `timber keep --continue`.
- `paths/<name>/` stores the actual checked-out path tree the user works in.

Any examples elsewhere in this PRD that appear to place `project.yaml` or `events.log` at the project root should be read as shorthand only. The canonical location is under `.timber/`.


## 8. Command design overview

### 8.1 Setup

```text
timber init <project-dir>
timber repo add <name> <url>
timber repo sync
```

### 8.2 Create

```text
timber new <name> --from <ref>
timber new <name> --from <ref> --repos <repo,repo>
timber new <name> --from <ref> --all
timber new <name> --from <ref> repo=other-ref
timber new <name> repo=ref repo=ref
timber add [<path>] --from <ref> <repo>...
timber add [<path>] repo=ref repo=ref
timber fork [<source>] <child>...
timber save [<path>] "<note>"
```

### 8.3 Use

```text
timber run [<path>] -- <command>
timber dir [<path>]
timber shell [<path>]
```

### 8.4 Understand

```text
timber ls
timber here
timber info [<path>]
timber status [<path>]
timber diff [<path>] [--repo <repo>]
timber note [<path>] "<note>"
```

### 8.5 Decide

```text
timber keep <child> --into <parent>
timber publish [<path>]
timber drop <name>
```

### 8.6 Maintain

```text
timber doctor
timber doctor --repair
timber prune
```

### 8.7 Completion internals

```text
timber complete repos --prefix <prefix>
timber complete refs --repo <repo> --prefix <prefix>
timber complete paths --prefix <prefix>
timber complete commands
```


## 9. Primary UX flows

### 9.1 Single-repo quick fix

```bash
timber init myapp
cd myapp
timber repo add app git@github.com:company/app.git
timber repo sync

timber new login-fix --from develop
timber run login-fix -- codex
timber save login-fix "fix login redirect"
timber publish login-fix
timber drop login-fix
```

In a single-repo project, the user does not need to name the repo. Timber should infer the default repo.

### 9.2 Multi-repo same branch

```bash
timber init myproject
cd myproject
timber repo add frontend git@github.com:company/frontend.git
timber repo add backend git@github.com:company/backend.git
timber repo add auth git@github.com:company/auth.git
timber repo sync

timber new auth-flow --from develop --repos frontend,backend,auth
```

Meaning:

```text
frontend -> origin/develop
backend -> origin/develop
auth -> origin/develop
```

In multi-repo projects, Timber should avoid silently materializing every registered repo. Use `--repos` for a task-specific subset or `--all` when the user explicitly wants every registered repo.

### 9.3 Multi-repo mixed branch composition

```bash
timber new review-auth-hotfix \
 frontend=main \
 backend=dev \
 auth=hotfix/123
```

Meaning:

```text
frontend -> origin/main
backend -> origin/dev
auth -> origin/hotfix/123
```

This use case should feel first-class.

### 9.4 Mostly one branch with override

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
```

Meaning:

```text
frontend -> origin/develop
backend -> origin/develop
auth -> origin/hotfix/123
```

If the user explicitly wants every registered repo included, use `--all` instead:

```bash
timber new auth-flow --from develop --all auth=hotfix/123
```

Override precedence:

1. explicit `repo=ref`
2. `--repos` filter or `--all` inclusion
3. `--from <ref>` default for included repos
4. project default repo in single-repo mode

Important distinction:

- `--from` chooses the default ref for repos that are included.
- `--repos`, `--all`, or explicit `repo=ref` tokens determine which repos are included.

### 9.5 Project with many repos, path with a subset

A project may register 10 repos, but a specific path may need only 3 of them.

```bash
timber repo add frontend git@github.com:company/frontend.git
timber repo add backend git@github.com:company/backend.git
timber repo add auth git@github.com:company/auth.git
timber repo add billing git@github.com:company/billing.git
timber repo add search git@github.com:company/search.git
timber repo add notifications git@github.com:company/notifications.git
timber repo add analytics git@github.com:company/analytics.git
timber repo add docs git@github.com:company/docs.git
timber repo add infra git@github.com:company/infra.git
timber repo add mobile git@github.com:company/mobile.git

timber new auth-flow --from develop --repos frontend,backend,auth
timber new billing-review --from main backend=dev billing=hotfix/123 --repos frontend,backend,billing,analytics,docs
timber new full-develop --from develop --all
```

Result:

```text
auth-flow contains 3 repos: frontend, backend, auth
billing-review contains 5 repos: frontend, backend, billing, analytics, docs
full-develop contains 10 repos: all registered repos
```

This keeps path startup, agent context, search scope, status output, and local setup proportional to the task.

### 9.6 Agent fanout from a base branch

```bash
timber new auth-baseline --from develop --repos frontend,backend,auth
timber fork auth-baseline auth-api auth-ui auth-tests

timber run auth-api -- codex
timber run auth-ui -- claude
timber run auth-tests -- codex
```

Each child starts from the same committed state. Each repo in each path gets a private branch.

### 9.7 Evolve, fork, keep one path

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
timber run auth-flow -- codex
timber save auth-flow "baseline auth integration works"

timber fork auth-flow try-backend-adapter try-frontend-fallback try-auth-only

timber run try-backend-adapter -- codex
timber run try-frontend-fallback -- claude
timber run try-auth-only -- codex

timber keep try-auth-only --into auth-flow
timber drop try-backend-adapter try-frontend-fallback
```

This is the most important differentiated workflow.

### 9.8 Come back later

```bash
timber ls
timber info auth-flow
timber status auth-flow
```

Expected `timber ls` output:

```text
NAME FROM AGE REPOS STATUS
auth-flow develop+override 1d frontend backend auth clean, 2 commits
try-backend-adapter auth-flow 4h frontend backend auth dirty, 1 commit
try-frontend-fallback auth-flow 4h frontend backend auth clean, 3 commits
try-auth-only auth-flow 4h frontend backend auth clean
```

Expected `timber info auth-flow` output:

```text
auth-flow
 path: paths/auth-flow
 created: 2026-06-21 14:05
 parent: origin/develop with auth=origin/hotfix/123
 purpose: baseline auth integration works

repos:
 frontend
 started from: origin/develop @ a1b2c3d
 branch: timber/auth-flow/frontend-8f3a
 status: clean, no commits

 backend
 started from: origin/develop @ e5f6a7b
 branch: timber/auth-flow/backend-91ac
 status: clean, 2 commits ahead

 auth
 started from: origin/hotfix/123 @ 91ff239
 branch: timber/auth-flow/auth-772d
 status: clean, no commits

children:
 try-backend-adapter
 try-frontend-fallback
 try-auth-only

resume:
 timber run auth-flow -- codex
 timber status auth-flow
 timber fork auth-flow try-a try-b try-c
 timber publish auth-flow
```


### 9.9 Add repos to an existing path

A user may start with a small path, then realize the task needs additional repos for context or edits. Timber should support expanding the path without forcing the user to recreate it or lose ongoing work.

Example:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth
timber run auth-flow -- codex
```

Later, the user realizes that `notifications` and `audit` are relevant. If they are not registered yet:

```bash
timber repo add notifications git@github.com:company/notifications.git
timber repo add audit git@github.com:company/audit.git
timber repo sync
```

Then add them to the existing path. If `auth-flow` has a default source ref such as `develop`, Timber can reuse it:

```bash
timber add auth-flow notifications audit
```

The user can also specify the default ref explicitly:

```bash
timber add auth-flow --from develop notifications audit
```

Or with repo-specific refs:

```bash
timber add auth-flow notifications=main audit=hotfix/456
```

From inside `paths/auth-flow/`, the context-aware form should work:

```bash
timber add notifications audit
timber add --from develop notifications audit
timber add notifications=main audit=hotfix/456
```

Expected result:

```text
paths/auth-flow/
 frontend/
 backend/
 auth/
 notifications/
 audit/
 PATH.md
 AGENTS.md
```

Rules:

- `timber add` adds registered repos to an existing path.
- If a repo is not registered, fail with a suggestion to run `timber repo add <name> <url>`.
- `timber add` should create private branches and worktrees using the same strategy as `timber new`.
- Existing dirty repos in the path should not block `timber add`; expanding context is often useful while work is in progress.
- If the repo already exists in the path, fail with a clear message.
- Update path metadata, `PATH.md`, `AGENTS.md`, and the event log.
- Future forks of this path inherit the expanded repo subset.
- Existing child paths are not automatically updated. Timber should report this clearly if the parent has children.
- `timber keep` should operate only on repos present in both source and target paths. If one side is missing a repo, it should skip with a warning or require an explicit future flag such as `--include-missing`.


## 10. CLI details

### 10.1 `timber init`

```bash
timber init myproject
```

Creates:

```text
myproject/
 .timber/repos/
 .timber/
  project.yaml
  paths/
  operations/
  events.log
 paths/
```

Optional flags:

```text
--paths-dir <path> default: paths
--repos-dir <path> default: .timber/repos
--metadata-dir <path> default: .timber
```

### 10.2 `timber repo add`

```bash
timber repo add backend git@github.com:company/backend.git
```

Expected behavior:

1. Validate project exists.
2. Validate repo name is unique and safe for paths.
3. Create `.timber/repos/backend.git` as a bare repo.
4. Add remote `origin`.
5. Configure fetch refspec.
6. Fetch remote branches and tags.
7. Update `.timber/project.yaml`.

Recommended internal setup:

```bash
git init --bare .timber/repos/backend.git
git -C .timber/repos/backend.git remote add origin git@github.com:company/backend.git
git -C .timber/repos/backend.git config remote.origin.fetch '+refs/heads/*:refs/remotes/origin/*'
git -C .timber/repos/backend.git fetch --prune --tags origin
```

Rationale: this makes `refs/remotes/origin/*` available in a bare repo cache.

### 10.3 `timber repo sync`

```bash
timber repo sync
```

Fetches all remotes:

```bash
git -C .timber/repos/frontend.git fetch --prune --tags origin
git -C .timber/repos/backend.git fetch --prune --tags origin
git -C .timber/repos/auth.git fetch --prune --tags origin
```

Important rule:

> `timber repo sync` fetches only. It must not move, rebase, or mutate existing paths.

If existing paths are now behind their source refs, `timber repo sync` may report that as information, but should not automatically change them.

### 10.4 `timber new`

Create a path.

Common forms:

```bash
timber new login-fix --from develop
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
timber new review-auth frontend=main backend=dev auth=hotfix/123
timber new backend-only --from develop --repos backend,auth
timber new full-develop --from develop --all
```

Rules:

- Path names must be unique.
- Path names must be safe for paths and branch names after sanitization.
- If the project has one repo, `--from` applies to that repo.
- If the project has multiple repos, Timber should not silently include every registered repo.
- In multi-repo mode, the user must specify path membership with `--repos`, `--all`, or explicit `repo=ref` mappings.
- `--from <ref>` chooses the default ref for included repos; it does not by itself define which repos are included in multi-repo projects.
- `repo=ref` overrides `--from` for that repo and includes that repo in the path.
- `--repos` limits which repos are materialized.
- `--all` explicitly materializes all registered repos.
- If explicit mappings are provided and no `--from`, include only explicitly mapped repos unless the user provides `--repos` or `--all`.
- If both `--repos` and `repo=ref` are provided, all explicit `repo=ref` repos must be included in the path. Prefer a clear error if a mapped repo is not listed in `--repos`, because silent auto-inclusion can be surprising in large projects.
- Path-level commands operate only on the repos recorded in that path's metadata, not on every repo registered in the project.
- In interactive mode, an underspecified multi-repo `timber new <name> --from <ref>` may prompt for repos. In non-interactive mode, it should fail and suggest `--repos` or `--all`.
- If an included repo does not contain the requested `--from` ref, the command must fail before creating the path unless that repo has an explicit `repo=ref` override.
- Default v0 behavior is all-or-nothing: if any included repo cannot resolve its requested ref, Timber should create nothing and print which repos need explicit overrides.

### 10.5 `timber add`

Add one or more registered repos to an existing path.

Common forms:

```bash
timber add auth-flow notifications audit # reuse path default ref if available
timber add auth-flow --from develop notifications audit
timber add auth-flow notifications=main audit=hotfix/456
```

Context-aware forms from inside `paths/auth-flow/`:

```bash
timber add notifications audit
timber add --from develop notifications audit
timber add notifications=main audit=hotfix/456
```

Expected behavior:

1. Resolve the target path from the argument or current directory.
2. Validate each repo is registered in the project.
3. Validate each repo is not already present in the target path.
4. Resolve refs using `--from`, any `repo=ref` overrides, or the path default source ref if available.
5. If no ref can be inferred for a bare repo name, fail with a message asking for `--from <ref>` or `repo=ref`.
6. Create a private branch for each added repo.
7. Add each worktree under `paths/<path>/<repo>/`.
8. Update path metadata.
9. Regenerate or update `PATH.md` and `AGENTS.md`.
10. Append an event to `.timber/events.log`.

Example metadata event:

```json
{"event":"path_repos_added","path":"auth-flow","repos":["notifications","audit"]}
```

Ref resolution rules for `timber add` in v0:

- `repo=ref` always wins for that repo.
- If `--from <ref>` is provided, it becomes the default ref for bare repo names in that command.
- Otherwise, Timber may reuse the path default source ref only if the path has a single unambiguous default source ref recorded in metadata.
- A path created from mixed refs without a single recorded default source ref must not guess for newly added bare repo names. In that case, `timber add auth-flow notifications` must fail and ask for `--from <ref>` or `notifications=<ref>`.
- A forked path inherits the parent's path-level default source ref if one exists. It does not invent a default from one of the forked repos.
- If the resolved ref is missing in any added repo, the whole `timber add` operation fails and rolls back any repos already added unless `--keep-partial` is specified.

Safety rules:

- Existing dirty repos in the path should not block adding new repos.
- If adding multiple repos and one fails, roll back repos already added unless `--keep-partial` is specified.
- If the path has children, warn that existing children are not automatically updated.
- `timber add` should not mutate existing repos in the path. It only adds new repo worktrees and metadata.

### 10.6 `timber save`

```bash
timber save auth-flow "checkpoint before experiments"
timber save "checkpoint before experiments" # when inside auth-flow
```

Creates local checkpoint commits in changed repos.

Default v0 behavior:

- Detect changed repos.
- Stage tracked changes by default.
- Include untracked files only when the user passes `--all`.
- Commit with message `checkpoint: <note>`.
- Skip clean repos.
- Do not push.

Rationale:

- This keeps the default aligned with the product's conservative safety posture.
- It reduces accidental commits of `.env`, caches, editor files, build outputs, or other local-only artifacts.

Potential safety prompt:

```text
backend:
 3 modified, 1 untracked

auth:
 1 modified

Create checkpoint commits? [y/N]
```

For non-interactive mode:

```bash
timber save auth-flow "checkpoint before experiments" --yes
```

### 10.7 `timber fork`

```bash
timber fork auth-flow try-api try-ui try-auth-only
timber fork try-api try-ui try-auth-only # when inside auth-flow
```

Creates child paths from the current committed state of the source path.

v0 rule:

> Source path must be clean.

If dirty:

```text
auth-flow has uncommitted changes in backend.

Choose one:
 timber save auth-flow "checkpoint before experiments"
 timber status auth-flow
 timber discard auth-flow --repo backend
```

For each repo in the source path:

1. Resolve source path repo `HEAD` commit.
2. Create new private branch for the child path at that commit.
3. Add child worktree.
4. Write child metadata.
5. Generate child `PATH.md` and `AGENTS.md`.

### 10.8 `timber run`

```bash
timber run auth-flow -- codex
timber run auth-flow -- claude
timber run auth-flow -- npm test
timber run auth-flow -- ./scripts/dev-up
timber run -- codex # when inside auth-flow
```

Runs command from the path root:

```text
paths/auth-flow/
```

The command should inherit the user environment, with optional Timber variables:

```text
TIMBER_PROJECT_ROOT=/path/to/myproject
TIMBER_PATH=auth-flow
TIMBER_PATH_ROOT=/path/to/myproject/paths/auth-flow
TIMBER_REPOS=frontend,backend,auth
```

### 10.9 `timber dir` and `timber shell`

```bash
timber dir auth-flow
```

Prints the absolute path to the path.

```bash
timber shell auth-flow
```

Starts a child shell in the path root.

Recommended shell helper:

```bash
wbcd() {
 cd "$(timber dir "$1")"
}
```

### 10.10 `timber here`

Show the Timber context inferred from the current directory.

```bash
timber here
```

Example from inside `paths/auth-flow/backend/src/main/java`:

```text
project: myproject
path: auth-flow
repo: backend
path: paths/auth-flow/backend

quick actions:
 timber status
 timber save "checkpoint note"
 timber fork try-a try-b
 timber run -- codex
```

If no Timber project is found, explain how to initialize or change directories.

### 10.11 `timber ls`

List paths.

```bash
timber ls
```

Output should be compact, glanceable, and useful after time away.

Include:

- name
- parent/source
- age
- repos
- dirty status
- commits ahead of base/fork point
- unpushed commits if known
- children count if applicable

Example:

```text
NAME FROM AGE REPOS STATUS
auth-flow develop 1d frontend api clean, 2 commits
try-api auth-flow 4h frontend api dirty, 1 commit
try-ui auth-flow 4h frontend api clean, 3 commits
```

### 10.12 `timber info`

Detailed path summary.

```bash
timber info auth-flow
```

Should answer:

- What is this?
- Where is it?
- What did it start from?
- What repos are included?
- What private branches exist?
- What changed?
- Does it have parent or child paths?
- How do I resume?
- How do I keep/drop/publish?

### 10.13 `timber status`

```bash
timber status
timber status auth-flow
```

For each repo, report:

- clean/dirty
- staged changes
- unstaged changes
- untracked files
- current private branch
- current commit
- commits ahead of source/fork point
- commits behind current upstream source ref if applicable
- merge/rebase in progress

### 10.14 `timber diff`

```bash
timber diff auth-flow
timber diff auth-flow --repo backend
```

Shows diff relative to the path source commit or fork point.

Useful modes for later:

```text
--stat
--name-only
--from-parent
--from-source
```

### 10.15 `timber keep`

```bash
timber keep try-auth-only --into auth-flow
```

Adopts child changes into parent.

v0 rules:

- Source child must be clean.
- Target parent must be clean.
- Source child must have target parent in metadata lineage, or user must pass `--force`.
- Repos with no changes relative to fork point are skipped.

Default v0 strategy: merge per repo.

Pseudo-flow per repo:

1. Identify child branch and parent branch.
2. Identify fork point commit stored in child metadata.
3. If child branch equals fork point, skip.
4. In parent path repo, merge child branch.
5. If conflicts, stop and instruct user.
6. Save operation state for `timber keep --continue`.

Operation-state requirements for v0:

- Before merging the first repo, create an operation state file under `.timber/operations/`.
- The state file must record at minimum:
  - operation type
  - source path
  - target path
  - ordered repo list
  - per-repo status: pending, merged, conflicted, skipped
  - fork point commit for each repo
  - source branch and target branch for each repo
  - created_at and last_updated timestamps
- If a conflict occurs, Timber must stop immediately after marking the conflicted repo and must not continue to later repos.
- `timber keep --continue` resumes from the first repo still marked `pending` or `conflicted` after the user resolves the conflict.
- `timber keep --abort` removes the operation state file but must not attempt to undo repos that were already merged cleanly before the conflict. Abort means "stop and clear resumable state," not "rewind Git history."
- `PATH.md` and the event log should be updated only after the keep operation fully succeeds, except that the event log may record a `keep_conflict` or `keep_started` event for recovery visibility.

Conflict message:

```text
backend:
 conflict while merging try-auth-only into auth-flow

Resolve conflicts in:
 paths/auth-flow/backend

Then run:
 timber keep --continue

Or abort:
 timber keep --abort
```

### 10.16 `timber drop`

```bash
timber drop try-api try-ui
```

Removes unwanted paths.

Safety behavior:

- If dirty, refuse unless `--force`.
- If has commits not kept or published, refuse unless `--force`.
- If has child paths, refuse unless `--recursive` or children are dropped first.
- By default, remove worktrees and keep private branches only if needed for safety.

Possible flags:

```text
--force discard dirty/unkept work
--keep-branches remove worktrees but keep private branches
--delete-branches delete private branches if safe
--recursive drop children too
```

### 10.17 `timber publish`

```bash
timber publish auth-flow
```

Pushes changed repos.

Single-repo default remote branch:

```text
origin/auth-flow
```

Multi-repo default remote branches:

```text
origin/auth-flow-frontend
origin/auth-flow-backend
origin/auth-flow-auth
```

Only repos with commits should be pushed.

Override:

```bash
timber publish auth-flow --repo backend --as scott/auth-flow
```

### 10.18 `timber note`

```bash
timber note auth-flow "Explore auth hotfix against backend dev"
```

Updates metadata and generated `PATH.md`.

Should append notes rather than overwriting history, unless `--replace` is provided.

### 10.19 `timber doctor`

```bash
timber doctor
timber doctor --repair
```

Checks:

- `.timber/repos/` exists
- `.timber/project.yaml` exists
- all configured bare repos exist
- remotes are configured
- remote-tracking refs exist
- path directories match metadata
- worktree metadata is valid
- private branches exist
- missing or stale worktree admin data
- dirty paths
- missing generated files
- parent/child metadata consistency

Repairs may include:

- regenerate `PATH.md`
- regenerate `AGENTS.md`
- run `git worktree prune`
- reconcile missing paths if safe
- suggest manual action for unsafe states


## 11. Shell completion requirements

Shell completion is a critical UX feature.

### 11.1 Repo completion

When the user types:

```bash
timber new auth-flow --from develop <TAB>
```

Timber should suggest repo assignment tokens:

```text
frontend=
backend=
auth=
```

If the user types:

```bash
timber new auth-flow --from develop au<TAB>
```

It should complete:

```bash
timber new auth-flow --from develop auth=
```

Because multi-repo `timber new --from <ref>` should not silently include all repos, completion should also make membership flags easy to discover:

```bash
timber new auth-flow --from develop --<TAB>
```

Should suggest at least:

```text
--repos
--all
```

When completing `--repos`, Timber should suggest registered repo names and support comma-separated continuation where the shell allows it.

### 11.2 Branch/ref completion after `repo=`

When the user types:

```bash
timber new auth-flow --from develop auth=<TAB>
```

Timber should show branch names for the `auth` repo:

```text
main
develop
hotfix/123
hotfix/456
release/2026-06
```

When the user types:

```bash
timber new auth-flow --from develop auth=hot<TAB>
```

Timber should narrow to:

```text
hotfix/123
hotfix/456
```

When unique:

```bash
timber new auth-flow --from develop auth=hotfix/1<TAB>
```

Completes to:

```bash
timber new auth-flow --from develop --repos frontend,backend,auth auth=hotfix/123
```

### 11.3 `--from` completion

```bash
timber new auth-flow --from <TAB>
```

Single-repo project:

- show refs for the default repo

Multi-repo project:

- show common branch names across repos first
- then possibly show deduplicated branch names from all repos

Example:

```text
main
develop
staging
release/2026-06
```

### 11.4 Path completion

Commands that take path names should complete path names:

```bash
timber info au<TAB>
timber run auth-flow -- codex
timber keep try<TAB> --into auth-flow
```

### 11.5 Completion implementation

Keep intelligence in the CLI, not in shell scripts.

Internal commands:

```bash
timber complete repos --prefix au
timber complete refs --repo auth --prefix hot
timber complete paths --prefix auth
timber complete commands
```

Completion should read local `.timber/repos/<repo>.git` refs only. It should not fetch over the network. Users run `timber repo sync` to update branch data.

Target shells:

1. zsh
2. bash
3. fish
4. PowerShell later


## 12. Git implementation details

### 12.1 Bare repo cache

Each registered repo has a bare repository under `.timber/repos/`:

```text
.timber/repos/frontend.git
.timber/repos/backend.git
.timber/repos/auth.git
```

Timber should shell out to Git rather than reimplementing Git internals.

### 12.2 Worktree creation

Do not check out shared branch names directly.

Avoid:

```bash
git -C .timber/repos/auth.git worktree add paths/auth-flow/auth develop
```

Use:

```bash
git -C .timber/repos/auth.git worktree add \
 paths/auth-flow/auth \
 -b timber/auth-flow/auth-772d \
 refs/remotes/origin/hotfix/123
```

This means many paths can start from the same remote branch without colliding.

### 12.3 Private branch naming

Format:

```text
timber/<path-name>/<repo-name>-<shortid>
```

Example:

```text
timber/auth-flow/backend-91ac
```

Rules:

- sanitize path name
- sanitize repo name
- include short random or hash suffix
- never require the user to remember this branch name
- store exact branch name in metadata

### 12.4 Ref resolution

User input examples:

```text
develop
hotfix/123
origin/develop
refs/remotes/origin/develop
<commit-sha>
```

Resolution order:

1. valid commit SHA
2. exact ref
3. `refs/remotes/origin/<input>`
4. `refs/heads/<input>`
5. tag resolution, possibly v1+

Display should prefer friendly names such as `develop` and `hotfix/123`.

Metadata should store both source ref and resolved commit.

### 12.5 Status calculations

For each path repo, track:

- `source_ref`
- `source_commit`
- `fork_point_commit`, if forked from another path
- `private_branch`
- current `HEAD`

Then compute:

- dirty status with `git status --porcelain=v1`
- ahead count with `git rev-list --count <base>..HEAD`
- behind source count with `git rev-list --count HEAD..<source_ref>` where meaningful
- unpushed commits by comparing to upstream if published

### 12.6 Keep strategy

For v0, `timber keep` should use merge by default.

Reasons:

- It preserves child experiment history.
- It maps naturally to "keep this path."
- It avoids rewriting child commits.

Possible future strategy options:

```bash
timber keep try-a --into auth-flow --squash
timber keep try-a --into auth-flow --cherry-pick
timber keep try-a --into auth-flow --repo backend
```

### 12.7 Worktree removal

Use Git worktree removal rather than deleting directories directly:

```bash
git -C .timber/repos/backend.git worktree remove paths/auth-flow/backend
```

Run `git worktree prune` only through explicit maintenance/repair commands or when safe.


## 13. Metadata

### 13.1 Project metadata

```yaml
# .timber/project.yaml
version: 1
name: myproject
created_at: "2026-06-21T14:05:00-04:00"
repos_dir: .timber/repos
paths_dir: paths
metadata_dir: .timber

default_repo: null

repos:
 frontend:
 url: git@github.com:company/frontend.git
 default_ref: main
 backend:
 url: git@github.com:company/backend.git
 default_ref: main
 auth:
 url: git@github.com:company/auth.git
 default_ref: main
```

Single-repo example:

```yaml
version: 1
name: myapp
default_repo: app
repos:
 app:
 url: git@github.com:company/app.git
 default_ref: main
```

### 13.2 Path metadata

```yaml
# .timber/paths/auth-flow.yaml
version: 1
name: auth-flow
path: paths/auth-flow
created_at: "2026-06-21T14:05:00-04:00"
purpose: "Explore auth hotfix against backend develop"

parent:
 type: remote
 description: "develop with auth=hotfix/123"
 default_source_ref: refs/remotes/origin/develop
 default_source_display: develop

repos:
 frontend:
 path: paths/auth-flow/frontend
 source_ref: refs/remotes/origin/develop
 source_display: develop
 source_commit: a1b2c3d4
 private_branch: timber/auth-flow/frontend-8f3a
 published_branch: null

 backend:
 path: paths/auth-flow/backend
 source_ref: refs/remotes/origin/develop
 source_display: develop
 source_commit: e5f6a7b8
 private_branch: timber/auth-flow/backend-91ac
 published_branch: null

 auth:
 path: paths/auth-flow/auth
 source_ref: refs/remotes/origin/hotfix/123
 source_display: hotfix/123
 source_commit: 91ff2390
 private_branch: timber/auth-flow/auth-772d
 published_branch: null

children:
 - try-backend-adapter
 - try-frontend-fallback
 - try-auth-only
```

Forked path example:

```yaml
parent:
 type: path
 name: auth-flow
 forked_at: "2026-06-21T17:15:00-04:00"
 fork_point_commits:
 frontend: a1b2c3d4
 backend: e5f6a7b8
 auth: 91ff2390
```

### 13.3 Event log

Append-only event log:

```jsonl
{"time":"2026-06-21T14:05:00-04:00","event":"path_created","path":"auth-flow"}
{"time":"2026-06-21T15:10:00-04:00","event":"save","path":"auth-flow","note":"baseline auth integration works"}
{"time":"2026-06-21T15:20:00-04:00","event":"path_forked","source":"auth-flow","children":["try-api","try-ui","try-auth-only"]}
{"time":"2026-06-21T18:40:00-04:00","event":"keep","source":"try-auth-only","target":"auth-flow"}
```


## 14. Generated files

Each path should include generated memory files at the path root.

### 14.1 `PATH.md`

Example:

````md
# auth-flow

Created: 2026-06-21 14:05
Parent: develop with auth=hotfix/123
Repos: frontend, backend, auth

## Purpose

Explore whether auth hotfix/123 works with backend develop and frontend develop.

## Repos

| Repo | Started from | Resolved commit | Private branch | Status |
|---|---|---:|---|---|
| frontend | origin/develop | a1b2c3d | timber/auth-flow/frontend-8f3a | clean |
| backend | origin/develop | e5f6a7b | timber/auth-flow/backend-91ac | clean |
| auth | origin/hotfix/123 | 91ff239 | timber/auth-flow/auth-772d | clean |

## Common commands

```bash
timber status auth-flow
timber run auth-flow -- codex
timber save auth-flow "checkpoint note"
timber add auth-flow notifications audit
timber fork auth-flow try-a try-b try-c
timber publish auth-flow
timber drop auth-flow
```

## Notes

- 2026-06-21 15:10 — baseline auth integration works.
````

`PATH.md` must be regenerated or updated by:

- `timber new`
- `timber fork`
- `timber save`
- `timber note`
- `timber keep`
- `timber publish`
- `timber doctor --repair`

### 14.2 `AGENTS.md`

Example:

````md
# Agent instructions for this path

This is an isolated Timber path.

## Rules

- Read `PATH.md` first.
- Do not edit files outside this path.
- Read repo-specific `AGENTS.md` files before editing a repo.
- Prefer targeted searches over broad scans.
- For cross-repo changes, identify producer and consumer contracts before editing.
- Before making major changes, summarize the plan.
- Before finishing, leave a concise summary of what changed.

## Repos

- `frontend`
- `backend`
- `auth`
````

This supports progressive disclosure for coding agents.

### 14.3 Optional `TASK.md`

For v0, `PATH.md` may be enough. A later version may generate `TASK.md` for agent-facing task instructions.


## 15. Hooks and setup automation

New paths are often painful because they lack local setup.

Timber should support hooks early, but keep them simple.

### 15.1 Hook phases

Potential hooks:

```text
post-repo-add
post-new
post-fork
pre-save
post-save
pre-keep
post-keep
pre-drop
post-drop
```

v0 minimum:

```text
post-new
post-fork
```

### 15.2 Project config example

```yaml
hooks:
 post-new:
 - repo: frontend
 run: pnpm install
 - repo: backend
 run: ./gradlew testClasses
 post-fork:
 - run: echo "Path ready: $TIMBER_PATH"
```

### 15.3 Preserve/copy files

Timber should optionally preserve common local config:

```yaml
preserve:
 include:
 - .env.example
 - .env.development
 - CLAUDE.local.md
 - .claude/settings.local.json
 - .aider.conf.yml
 exclude:
 - .env.production
 - .env.local.secret
```

Default behavior should avoid copying secrets unless explicitly configured.

### 15.4 Cache strategy

Do not aggressively symlink dependencies by default. Package managers vary.

Support user-defined hooks and future cache helpers. Potential later features:

- copy selected build caches
- symlink safe caches
- deterministic per-path cache dirs
- npm/pnpm/yarn/Gradle/Maven/Cargo recipes

### 15.5 Ports

Agent and parallel dev-server workflows often need unique ports.

Potential environment variables:

```text
TIMBER_PORT_BASE=43000
TIMBER_PORT_1=43123
TIMBER_PORT_2=43124
```

Deterministic port allocation can be added after v0.


## 16. Safety model

Safety should be conservative.

### 16.1 Clean requirements

For v0:

```text
timber fork requires source clean
timber keep requires source and target clean
timber drop refuses dirty or unkept work unless forced
timber repo sync fetches only
```

### 16.2 Dirty path errors

If a command requires clean state, provide next-step commands.

Example:

```text
auth-flow has uncommitted changes in backend.

Choose one:
 timber save auth-flow "checkpoint before experiments"
 timber status auth-flow
 timber diff auth-flow --repo backend
```

### 16.3 Destructive operations

Commands that destroy or hide work must require explicit flags:

```text
--force
--yes
--delete-branches
```

Defaults should preserve work.

### 16.4 Transactionality

Multi-repo operations should either complete or roll back when practical.

For `timber new`:

- If repo 3 of 5 fails, remove repo worktrees already created for that path unless `--keep-partial` is specified.
- Metadata should not be marked complete unless all required repos are created.

For `timber keep`:

- If a conflict occurs in one repo, stop and record operation state.
- Do not continue to later repos after conflict unless explicitly designed.
- Support `timber keep --continue` and `timber keep --abort`.


## 17. Competitive context and learnings

This section is research context, not direct implementation requirements.

### 17.1 Worktrunk

Worktrunk is a polished Git worktree manager focused on making worktrees easy for parallel AI-agent workflows. Its README highlights core commands such as switch, list, merge, and remove; it also advertises hooks, LLM commit messages, merge workflow, interactive picker, cache copying, PR checkout, and per-worktree dev-server support.

Learning for Timber:

- Status/list UX matters a lot.
- Hooks and cache copying are valuable.
- Merge/cleanup workflow should be smooth.
- Timber should not compete solely as a single-repo worktree wrapper.

### 17.2 CodeRabbit Git Worktree Runner / `git gtr`

CodeRabbit's `git-worktree-runner` is a Bash-based worktree manager with editor and AI integration. It emphasizes configuration copying, dependency installation, hooks, editor launch, and AI tool launch.

Learning for Timber:

- `run`/agent integration must be first-class.
- Copy/preserve rules are important.
- Simple wrappers around common workflows provide real value.

### 17.3 Grove multi-repo path orchestrator

Nick Senap's Grove (`gw`) is especially relevant because it orchestrates worktree paths across multiple repos. Its README describes "one command, one path, all repos on the same branch."

Learning for Timber:

- Multi-repo path orchestration is a real use case.
- Timber's differentiator should be mixed per-repo refs and forkable experiment lineage.

Example differentiator:

```bash
timber new review-auth frontend=main backend=dev auth=hotfix/123
timber add review-auth audit=main
timber fork review-auth try-api try-ui try-auth
```

### 17.4 wtp / Worktree Plus

`wtp` focuses on automated setup, branch tracking, and smart navigation.

Learning for Timber:

- Shell integration and navigation helpers are not optional.
- Branch tracking and status should be clear.

### 17.5 Treehouse

Treehouse manages a pool of reusable isolated worktrees for agents, preserving dependencies and build cache.

Learning for Timber:

- Cold-start cost matters.
- Timber may later need cache/pool concepts, but v0 should first nail durable named paths and experiment forking.

### 17.6 Timber's intended niche

Timber should own this flow:

```bash
timber new review-auth frontend=main backend=dev auth=hotfix/123
timber add review-auth audit=main
timber fork review-auth try-api try-ui try-auth
timber run try-auth -- codex
timber keep try-auth --into review-auth
timber drop try-api try-ui
```

That is distinct from "create a worktree for branch X."


## 18. MVP scope

### 18.1 Required commands

```text
timber init
timber repo add
timber repo sync
timber new
timber add
timber ls
timber here
timber info
timber status
timber run
timber dir
timber save
timber fork
timber keep
timber drop
```

### 18.2 Strongly recommended for MVP

```text
timber diff
timber note
timber doctor
shell completion for repos, refs, and path names
```

### 18.3 Can wait until after MVP

```text
timber publish
timber shell
advanced hooks
cache copying
port allocation
interactive picker
MCP integration
agent session tracking
PR checkout
```


## 19. MVP acceptance criteria

A user can:

1. Initialize a Timber project.
2. Register one Git repo.
3. Register multiple Git repos.
4. Fetch all remotes.
5. Create a single-repo path from a branch.
6. Create a multi-repo path where selected repos start from the same branch.
7. Create a multi-repo path with repo-specific branch overrides.
8. Register many repos in a project and create paths containing different repo subsets, such as 3 repos, 5 repos, or all repos.
9. Avoid accidentally materializing every registered repo in multi-repo projects unless the user explicitly passes `--all`.
10. Add additional registered repos to an existing path after work has already started.
11. Use context-aware commands from inside a path or repo directory.
12. Use tab completion for repo names and branch names.
13. Run a command or coding agent inside a path.
14. See all paths and their status.
15. Understand a path after returning later via `timber here`, `timber info`, and `PATH.md`.
16. Save dirty work as local checkpoint commits.
17. Fork a clean path into multiple children.
18. Keep a child path into its parent.
19. Drop unwanted paths safely.
20. Recover from common metadata/worktree issues with `timber doctor` or clear instructions.


## 20. Implementation recommendations

### 20.1 Language

Good options:

- Go
- Rust
- TypeScript/Node

Recommendation: Go first, or Rust if there is a specific reason to prefer it. Single-binary distribution and robust CLI/completion support matter more here than fastest-possible prototyping.

### 20.2 Git access

Shell out to Git.

Use stable machine-readable output where possible:

```bash
git worktree list --porcelain -z
git status --porcelain=v1
git for-each-ref --format='%(refname) %(objectname)'
git rev-parse <ref>
git rev-list --count <range>
```

### 20.3 CLI parser

Requirements:

- subcommands
- variadic `repo=ref` args
- `--` passthrough for `timber run`
- shell completion generation
- helpful errors

Good libraries:

- Go: Cobra
- Rust: clap
- Node: Commander or Clipanion

### 20.4 Data format

Use YAML or TOML for human-readable metadata. YAML is used in examples, but TOML is also acceptable.

Use JSONL for event logs.


## 21. Testing plan

### 21.1 Unit tests

- repo/ref argument parser
- ref resolution
- private branch name sanitization
- metadata read/write
- status parser
- completion handlers
- path lineage rules
- current-directory context inference
- path repo membership changes

### 21.2 Integration tests

Create temporary Git repos locally and test:

- `timber repo add`
- `timber repo sync`
- `timber new --from main`
- `timber new frontend=main backend=dev`
- `timber add` to expand an existing path with new repos
- multiple paths from same branch
- no branch checkout collisions
- dirty path fork refusal
- `timber save`
- `timber fork`
- `timber keep`
- conflict handling
- `timber drop`
- `timber doctor`

### 21.3 Shell completion tests

Test dynamic completion for:

```bash
timber new auth-flow --from develop au<TAB>
timber new auth-flow --from develop auth=<TAB>
timber new auth-flow --from develop auth=hot<TAB>
timber info au<TAB>
```

At minimum, test completion backend commands:

```bash
timber complete repos --prefix au
timber complete refs --repo auth --prefix hot
timber complete paths --prefix auth
```+


## 22. Open questions

Resolved in v0.5:

1. `timber save` stages tracked files by default. Untracked files require `--all`.
2. `timber keep` defaults to merge.
3. Missing requested refs in included repos cause all-or-nothing failure unless that repo has an explicit override.

Still open:

1. Should `timber publish` be in MVP?
2. Should Timber support presets later, such as `timber preset add develop --from develop`?
3. Should Timber support dirty fork later via automatic checkpoint commits?
4. Should generated `AGENTS.md` be customizable globally?
5. Should Timber record agent sessions and prompts?
6. Should Timber support pooled/reusable paths for faster startup?
7. How should Timber handle submodules?
8. Should Timber support read-only paths later, or is "all paths are mutable and private-branch-backed" enough?
9. What should the default publish branch naming convention be for multi-repo spaces?
10. Should v1 support removing repos from an existing path, or is adding repos enough for the core workflow?


## 23. Recommended v0 build sequence

### Milestone 1: single-repo core

- `timber init`
- `timber repo add`
- `timber repo sync`
- `timber new --from`
- private branch creation
- `timber ls`
- `timber status`
- `timber dir`
- `timber run`
- basic metadata and `PATH.md`

### Milestone 2: multi-repo composition

- multiple repos in project
- `repo=ref` parser
- `--repos`
- `--all`
- path repo subsets
- mixed repo refs
- safer multi-repo defaults that require `--repos`, `--all`, or explicit mappings
- `timber add` for expanding an existing path with additional registered repos
- multi-repo status
- generated `AGENTS.md`

### Milestone 3: experiment workflow

- `timber save`
- `timber fork`
- parent/child metadata
- `timber keep`
- `timber drop`

### Milestone 4: UX polish

- shell completion
- context-aware command inference
- `timber here`
- `timber info`
- `timber diff`
- `timber note`
- `timber doctor`
- better errors and recovery messages

### Milestone 5: integration polish

- `timber publish`
- hooks
- preserve/copy config
- shell helper installation
- cache recipes
- port allocation


## 24. Implementation contract for a coding agent

When implementing this PRD, optimize for these outcomes:

1. The CLI should feel simple even though Git mechanics are complex.
2. Private generated branches should be invisible unless useful for debugging.
3. All path creation must avoid branch checkout collisions.
4. Multi-repo operations should be safe and explain partial failures clearly.
5. Path commands must respect each path's repo subset; a project with 10 registered repos may have paths containing 3, 5, or all 10 repos.
6. Multi-repo path creation must not accidentally materialize every registered repo unless the user explicitly asks for all repos.
7. Existing paths can be expanded with additional registered repos using `timber add` without mutating existing repo worktrees.
8. Context-aware commands should work from inside a path or repo directory whenever possible.
9. The user should be able to run `timber ls`, `timber here`, or `timber info <name>` after days away and immediately know what to do next.
10. Shell completion for `repo=ref` is a primary feature, not polish.
11. Avoid broad scans of user repos. Use Git metadata and targeted commands.
12. Prefer conservative safety checks over clever destructive behavior.
13. Keep generated files useful for humans and agents.
14. Do not create extra concepts unless the UX truly requires them.


## 25. Source notes

Research checked on 2026-06-21.

- Git worktree documentation: https://git-scm.com/docs/git-worktree
- Git clone documentation: https://git-scm.com/docs/git-clone
- Worktrunk: https://github.com/max-sixty/worktrunk
- CodeRabbit Git Worktree Runner: https://github.com/coderabbitai/git-worktree-runner
- Grove multi-repo worktree orchestrator: https://github.com/nicksenap/grove
- wtp / Worktree Plus: https://github.com/satococoa/wtp
- Treehouse: https://github.com/kunchenguid/treehouse
- Grove single-repo worktree CLI: https://github.com/sQVe/grove
