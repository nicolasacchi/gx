# gx vs gh — when to use which

Two CLIs, one GitHub. [`gh`](https://cli.github.com) is the official GitHub CLI for
repositories, pull requests, CI, releases, and auth. **`gx`** (GitHub Explorer) is a custom CLI
that fills the gaps `gh` leaves around **issues and Projects v2 task management**. They are
complementary — most sessions use both.

> **TL;DR** — Reach for **`gx`** whenever you're *creating or updating a task*: issues (with
> type, assignees, milestone, labels, state) and project-board fields (Status, Priority, Sprint,
> Story Points, …), sub-issues, drafts, milestones, iterations. Reach for **`gh`** for everything
> *around* the code: pull requests, CI runs, repo/release/label admin, auth, and raw API.
> `gx` reuses `gh`'s auth token, so `gh auth login` is a prerequisite for both.

## What each tool is

|              | `gx` — GitHub Explorer                                                   | `gh` — GitHub CLI                                      |
|--------------|--------------------------------------------------------------------------|-------------------------------------------------------|
| **Origin**   | Custom Go CLI (`github.com/nicolasacchi/gx`)                             | Official GitHub CLI (`cli/cli`)                        |
| **Focus**    | Issues + **Projects v2** task management                                 | Repos, pull requests, CI/CD, releases, auth, raw API  |
| **Auth**     | Reuses `gh auth token` (or `GITHUB_TOKEN` / config) — no separate login  | `gh auth login` — the source of the token             |
| **Output**   | Tables on TTY, JSON when piped, `--jq` (gjson) filtering                 | Tables / JSON with `--json` + `--jq`                  |

## Quick decision — task → tool

| I want to…                                                                       | Use                                                                 |
|----------------------------------------------------------------------------------|---------------------------------------------------------------------|
| Create an issue **with an issue type** (Task/Bug/Feature/…)                      | **`gx issues create --type`** (`gh issue create` has no `--type`)   |
| Set **any** issue field on create/edit (body, assignees, milestone, labels, state) in one call | **`gx issues create` / `gx issues edit`**             |
| Add the issue to a board **and** set Status/Priority/Sprint in one call          | **`gx issues create --project-number …`** or `gx items set`         |
| Read or write project-board field values (auto-resolves field/option IDs)        | **`gx items get` / `gx items set`**                                 |
| Manage sub-issues (parent/child) or iterations/sprints                           | **`gx sub-issues` / `gx iterations`** (`gh` can't)                  |
| Bulk-edit/close many issues matching a filter                                    | **`gx bulk`**                                                       |
| Open / review / merge a **pull request**                                         | **`gh pr`** (`gx` has no PR commands)                               |
| Watch / rerun **CI**, view checks, logs                                          | **`gh run` / `gh pr checks`**                                       |
| Create a repo, a release, a gist; manage repo settings                           | **`gh repo` / `gh release` / `gh gist`**                            |
| Create a **project field** or an **org issue type** (structure, not data)        | **`gh project field-create` / `gh api … admin:org`**                |
| Authenticate / refresh scopes                                                    | **`gh auth`**                                                       |
| Hit an arbitrary REST/GraphQL endpoint                                           | **`gh api`** (gx wraps the common ones)                             |

## Feature matrix

| Capability                                                        | `gx`        | `gh`         | Notes                                                                       |
|-------------------------------------------------------------------|-------------|--------------|-----------------------------------------------------------------------------|
| Issue create with **issue type**                                  | ✅ yes      | ❌ no flag    | `gx` sends the type name via REST; `gh issue create` has no `--type`        |
| Issue edit: body / type / milestone / assignees / state in one call | ✅ yes    | ⚠️ partial   | `gh issue edit` covers title/body/labels/assignees/milestone but not type   |
| Multiple assignees, add/remove                                    | ✅ yes      | ✅ yes        | `gx` warns on GitHub's silent drop of invalid logins                        |
| **Sub-issues** (add / remove / reorder)                           | ✅ yes      | ❌ no         | GraphQL-only feature; `gh` doesn't expose it                                |
| **Project board** field read/write (auto ID resolution)          | ✅ yes      | ⚠️ manual IDs | `gh project item-edit` needs raw field/option IDs; `gx items set` takes names |
| Draft project items + convert to issue                            | ✅ yes      | ⚠️ via api    | `gx items add-draft` / `convert-draft`                                      |
| Milestones (as epics) / iterations (sprints)                      | ✅ yes      | ⚠️ limited    | Iterations are Projects-v2-only; `gh` has no first-class support            |
| Issue type discovery (`gx issues types`)                          | ✅ yes      | ⚠️ via api    | `gh api /orgs/{org}/issue-types`                                            |
| Transfer issue / create linked branch (`develop`)                 | ✅ yes      | ✅ yes        | Both: `gx issues transfer/develop` ≈ `gh issue transfer/develop`           |
| Bulk edit/close across a selector                                 | ✅ yes      | ⚠️ scripted   | `gx bulk edit/close`                                                        |
| **Pull requests** (create / review / merge / checks)              | ❌ no       | ✅ yes        | Use `gh pr …`                                                               |
| **CI/CD** runs, workflow dispatch                                 | ❌ no       | ✅ yes        | Use `gh run …`                                                              |
| Repo / release / gist admin                                       | ⚠️ labels only | ✅ yes     | `gx labels create/delete`; everything else is `gh`                          |
| Create project **fields** / org issue **types** (structure)       | ❌ no       | ✅ yes        | `gh project field-create`, `gh api` (admin:org)                             |
| Auth / token / scope refresh                                      | ⚠️ reuses gh | ✅ yes       | `gh auth login` then `gh auth refresh -s read:project,project`              |
| Robust retries (Retry-After / 5xx / network)                      | ✅ yes      | ✅ yes        | `gx` retries hint-bearing limits; never hammers a bare-403 secondary limit  |

## What only `gx` does

- **Issue type on create.** `gh issue create` has no `--type` flag — `gx issues create --type "Task"` is the clean path.
- **Sub-issues.** Parent/child links (add, remove, reorder) — `gh` has no command for them.
- **Project board fields by name.** `gx items get/set` auto-resolve field and option names to node IDs; with `gh` you must look up and pass raw `PVTF_…` / option IDs by hand.
- **Iterations / sprints, milestones-as-epics, drafts, bulk edits** — all first-class in `gx`.
- **One-call task setup.** `gx issues create --project-number N --status … --field …` creates the issue, adds it to the board, and sets fields in a single command.

## What only `gh` does

- **Pull requests** — create, review, comment, merge, checks. `gx` has no PR surface.
- **CI/CD** — `gh run` list/view/watch/rerun, workflow dispatch.
- **Repo & release admin** — `gh repo create`, `gh release`, `gh gist`, settings, secrets.
- **Structure creation** — project fields (`gh project field-create`) and org issue types (`gh api` with `admin:org`). `gx` fills field *values*, not field *definitions*.
- **Auth** — `gh auth login` / `refresh`. `gx` borrows the resulting token.
- **Raw API** — `gh api` for any endpoint `gx` doesn't wrap.

## Auth & setup

`gx` resolves its token in order: `--token` → `GITHUB_TOKEN` → `gh auth token` → `~/.config/gx/config.toml`.
In practice, if `gh` is authenticated, `gx` just works. Board mutations need the project scopes —
run `gh auth refresh -s read:project,project` once.

## Side-by-side examples

Create a fully-populated task (issue + board) — `gx` in one call vs. the `gh` dance:

```bash
# gx — one command
gx issues create --title "Fix login" --type "Bug" --assignee alice \
  --project-number 3 --status "Backlog" --priority "High" --field "Component" --value "TECH"

# gh — multiple commands + manual IDs
gh issue create --title "Fix login" --assignee alice         # no --type
gh api repos/OWNER/REPO/issues/N -X PATCH -f type=Bug
gh project item-add 3 --owner OWNER --url <issue-url>
gh project item-edit --id <ITEM_ID> --project-id <PROJ_ID> \
  --field-id <PVTSSF_…> --single-select-option-id <OPT_ID>   # × each field
```

Read an issue's current board state, then a PR for it:

```bash
gx items get 123 --project-number 3        # current Status/Priority/… (gx)
gh pr create --fill --head fix-login        # open the PR (gh)
gh pr checks                                # watch CI (gh)
```

## Rule of thumb

> **Tasks → `gx`. Code → `gh`.** Anything that lives on an issue or a Projects v2 board is `gx`'s
> job; anything that lives in the repo, a PR, or CI is `gh`'s. They share one auth token, so
> switching between them is free.
