# CLAUDE.md — gx

Go CLI for GitHub Projects management. Handles sub-issues, iterations (sprints), milestones (epics), and project board field updates — everything `gh` CLI can't do. Dual REST + GraphQL API client. Auto-JSON on pipe, gjson filtering, TTY tables.

**API**: GitHub REST API v3 + GitHub GraphQL API. Auth via `gh auth token`, `GITHUB_TOKEN`, or config file.

## Authentication

Resolution order (first non-empty wins):

1. `--token` flag
2. `GITHUB_TOKEN` env var
3. `gh auth token` (reads from gh CLI's stored credentials)
4. `~/.config/gx/config.toml` — project from `--project` flag, then `default_project`

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | `GX_OWNER` | GitHub org/user |
| `--repo` | `GX_REPO` | Repository name |
| `--token` | `GITHUB_TOKEN` / gh auth | GitHub token |
| `--project` | — | Named project from config |
| `--json` | false | Force JSON output |
| `--jq` | — | gjson filter path |
| `--limit` | 50 | Max results |
| `--verbose` | false | Print requests to stderr |
| `--quiet` | false | Suppress non-error output |

## Commands

### issues (REST)
```bash
gx issues list --label "type:bug" --state open --milestone "v2.1"
gx issues get 123
gx issues create --title "Fix" --label "type:bug" --milestone "v2.1"
gx issues create --title "Sub-task" --parent 456    # create + link as sub-issue
gx issues edit 123 --title "Updated" --add-label "must-do"
gx issues close 123 --reason "not planned"
gx issues assign 123 --user nicolasacchi
```

### sub-issues (GraphQL — the killer feature)
```bash
gx sub-issues list 123
gx sub-issues add 123 456                           # link existing
gx sub-issues add 123 --title "New task" --label "type:sub-task"  # create + link
gx sub-issues remove 123 456
gx sub-issues reorder 123 456 --after 789
```

### milestones (REST — epic equivalent)
```bash
gx milestones list
gx milestones create --title "CoMarketing" --due "2026-06-01"
gx milestones close 1
gx milestones issues 1
```

### iterations (GraphQL — sprint equivalent)
```bash
gx iterations list --project-number 1
gx iterations current --project-number 1
```

### items (GraphQL — auto-resolves field/option IDs)
```bash
gx items set 123 --project-number 1 --status "In Progress"
gx items set 123 --project-number 1 --priority "High" --points 5
gx items set 123 --project-number 1 --iteration "Sprint 46"
gx items set 123 --project-number 1 --field "Component" --value "TECH"
gx items clear 123 --project-number 1 --field "Story Points"
```

### board (GraphQL)
```bash
gx board list
gx board fields --project-number 1
```

### comments, labels, search, overview, config, open (REST)
```bash
gx comments add 123 --file context.md
gx labels list
gx search "coupon discount" --label "type:bug"
gx overview
gx config add production --owner 1000farmacie --repo 1000farmacie
gx open 123
```

## Output

- **TTY**: Tables (go-pretty)
- **Piped**: Always JSON
- `--json`: Force JSON on TTY
- `--jq`: gjson filter (NOT jq syntax). Array: `#.field`. Object: `#.{a:a,b:b}`

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | API/network error |
| 3 | Auth error (401/403) |
| 4 | Not found (404) |

## Architecture

```
cmd/gx/main.go               Entry point
internal/
  client/rest.go              REST API: issues, milestones, labels, comments
  client/graphql.go           GraphQL API: sub-issues, iterations, project fields
  client/errors.go            APIError with ExitCode()
  commands/root.go            Root command, global flags, auth
  commands/*.go               One file per command group (14 groups)
  config/config.go            TOML config, multi-project, gh auth token reuse
  output/output.go            JSON/table dispatcher, TTY detection
  output/table.go             go-pretty tables, column definitions
  output/filter.go            gjson --jq filter
```

## Dual API

- **REST** (`api.github.com/repos/{owner}/{repo}/...`): issues, milestones, labels, comments, search
- **GraphQL** (`api.github.com/graphql`): sub-issues, iterations, project fields, project items

GraphQL helpers for node ID resolution:
- `IssueNodeID(number)` → GraphQL node ID
- `ProjectNodeID(projectNumber)` → GraphQL node ID
- `getProjectFields()` → all fields with IDs and options
- `resolveOptionID(fieldName, optionName)` → auto-resolve both IDs

## Building

```bash
make build    # → bin/gx
make install  # → ~/go/bin/gx
make test     # → go test ./...
```
