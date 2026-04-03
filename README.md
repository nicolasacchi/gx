# gx

CLI for [GitHub Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects) management. Handles sub-issues, iterations (sprints), milestones (epics), and project board field updates — everything `gh` CLI can't do. Designed for AI agent integration.

## Install

```bash
go install github.com/nicolasacchi/gx/cmd/gx@latest
```

Or build from source:

```bash
git clone https://github.com/nicolasacchi/gx.git
cd gx
make install
```

## Why gx?

The `gh` CLI handles basic issue CRUD but can't:

| Gap | gh | gx |
|-----|----|----|
| **Sub-issues** | No support | `gx sub-issues add/remove/list` |
| **Iterations (sprints)** | No commands | `gx iterations list/current` |
| **Milestones** | Only a flag on issue create | `gx milestones list/create/close/issues` |
| **Project field updates** | Requires manual ID lookups | `gx items set 123 --status "In Progress"` |
| **Overview** | Impossible | `gx overview` (parallel health snapshot) |

## Quick Start

```bash
# Auth: reuses gh CLI token automatically
gh auth login

# Or set explicitly
export GITHUB_TOKEN=ghp_...

# List issues
gx issues list --owner myorg --repo myrepo --label "type:bug"

# Sub-issues (the killer feature)
gx sub-issues list 123
gx sub-issues add 123 456
gx sub-issues add 123 --title "New sub-task"

# Milestones (epic equivalent)
gx milestones create --title "v2.1" --due "2026-06-01"
gx milestones issues 1

# Project board fields (auto-resolves IDs)
gx items set 123 --project-number 1 --status "In Progress" --priority "High"

# Iterations (sprint equivalent)
gx iterations list --project-number 1

# Overview (parallel fetch)
gx overview
```

## Command Groups

| Group | Subcommands | API |
|-------|-------------|-----|
| `issues` | list, get, create, edit, close, reopen, assign, timeline, linked-prs, pin/unpin, lock/unlock | REST + GraphQL |
| `sub-issues` | list, add, remove, reorder | **GraphQL** |
| `milestones` | list, get, create, edit, close, reopen, issues | REST |
| `iterations` | list, current | **GraphQL** |
| `items` | add, set, clear, archive | **GraphQL** |
| `board` | list, fields | **GraphQL** |
| `bulk` | edit, close | REST |
| `comments` | list, add | REST |
| `labels` | list, create, delete | REST |
| `search` | (text + filters) | REST |
| `overview` | (parallel health snapshot) | REST |
| `config` | add, remove, list, use, current | local |
| `open` | (browser / --url) | local |

## License

MIT
