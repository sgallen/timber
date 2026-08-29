# Timber

Timber is a path manager for Git experiments and coding agents.

It turns one or more Git repos into isolated development paths you can create, fork,
compare, keep, and throw away as a unit.

## TL;DR

Git worktrees make parallel development possible.

Then the bookkeeping starts.

Which branch belongs to which experiment? Where are its worktrees? What happens when the
change spans three repos? How do you fork competing approaches, keep the winner, and
remove the rest without leaving branches and directories everywhere?

Timber manages that workflow.

```text
Create a path -> Fork alternatives -> Work independently -> Keep one -> Drop the rest
```

A **path** is the thing you work in. It may contain one repo, every repo in a project, or
only the repos needed for that experiment. Each repo keeps its own Git history. Timber
manages the branches, worktrees, directories, metadata, and relationships around them.

The path is the unit of the experiment.

## One path, several repos

```bash
timber init myproject
cd myproject

timber repo add app git@github.com:company/app.git
timber repo add auth git@github.com:company/auth.git
timber repo sync

timber new auth-flow --from main --repos app,auth
timber fork auth-flow try-api try-ui try-auth-only

# Work happens independently in each child path.

timber keep try-auth-only --into auth-flow
timber drop try-api try-ui
```

A **project** is the top-level Timber directory. It holds the repo catalog, paths,
relationships, and operation state.

A **repo** is a Git repository registered with that project.

A **path** is an isolated working directory containing one worktree for each participating
repo. The developer works with `auth-flow` or `try-api` as a unit. Timber coordinates the
Git machinery inside it.

A path can start from shared or per-repo refs, fork into child paths, and later be kept or
dropped as one environment. The individual worktrees remain ordinary Git repositories.

## How Timber works

### Use only the repos a path needs

Multi-repo does not mean every path needs every repo.

```bash
timber new auth-flow --from main --repos app,auth
```

Start repos from different refs when the work requires it:

```bash
timber new release-check --from main --repos app,auth auth=hotfix/session
```

Add another registered and synchronized repo later:

```bash
timber add auth-flow billing
```

Or choose its starting ref explicitly:

```bash
timber add auth-flow billing=master
```

No silent materialization of the whole repo catalog. A path contains the repos you choose.

### Fork, keep, and drop whole approaches

```bash
timber fork auth-flow try-library try-custom
```

Each child gets its own private branches, worktrees, directory, and path metadata. A human
or coding agent can work there without sharing a checkout with another approach.

When one wins:

```bash
timber keep try-custom --into auth-flow
```

Timber merges the child into the target repo by repo. Interrupted conflict resolution is
explicit:

```bash
timber keep --continue
timber keep --abort
```

Remove an abandoned path with:

```bash
timber drop try-library
```

Dropping is conservative. Dirty paths, child relationships, and managed branches require
explicit handling.

### Use the current directory as context

From somewhere inside:

```text
paths/auth-flow/app/src/...
```

Timber can infer the current project, path, and repo:

```bash
timber here
timber status
timber info
timber diff
timber dir
timber run -- make test
```

At the project root, provide the path name when needed:

```bash
timber status auth-flow
timber run auth-flow -- make test
```

`timber run` executes the command once from the path root. It provides
`TIMBER_PROJECT_ROOT`, `TIMBER_PATH`, `TIMBER_PATH_ROOT`, and `TIMBER_REPOS` to the
process.

## Install

You need Git and Go 1.22 or later.

### Build from this repo

```bash
git clone https://github.com/sgallen/timber.git
cd timber

make build
mkdir -p ~/.local/bin
cp ./bin/timber ~/.local/bin/timber

export PATH="$HOME/.local/bin:$PATH"
timber version
```

### Install with Go

```bash
go install github.com/sgallen/timber/cmd/timber@latest
```

Make sure your Go bin directory is on `PATH`.

## Commands

```text
# Project and repos
timber init <project-dir>
timber repo add <name> <url>
timber repo ls
timber repo sync
timber repo rm <name>

# Paths
timber new <name> --from <ref>
timber add [<path>] [--from <ref>] <repo>... [repo=ref ...]
timber fork [<source>] <child>...
timber keep [<child>] --into <target>
timber drop <path>...

# Inspect and use
timber ls
timber here
timber info [path]
timber status [path]
timber diff [path] [--repo <name>]
timber dir [path]
timber run [<path>] -- <command>
```

Run `timber <command> --help` for command-specific options and examples.

## Shell completion

Timber provides path, repo, and ref completion for Bash and zsh.

```bash
# Bash
source <(timber completion bash)

# Zsh
source <(timber completion zsh)
```

Add the appropriate line to your shell configuration to enable it permanently.

## Git still does Git

Timber manages project setup, repo synchronization, path creation, private branches and
worktrees, path lineage, reconciliation, cleanup, context, and inspection.

Git still owns editing, staging, commits, history, pushing, pulling, merge-conflict
resolution, pull requests, branch protection, code review, and releases.

Timber manages the environment around the work. It does not replace the repository
workflow inside it.
