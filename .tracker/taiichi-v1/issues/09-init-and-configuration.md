# `taiichi init` and configuration

Type: grilling
Status: resolved
Blocked by: 07, 08

## Question

What does configuration look like, per repo and per server?

- The committed repo config file: its name and format (TOML was suggested but not chosen; YAML is already used for Stage data front matter, and JSON is built into Elixir), and its fields (Tracker, Forge, their options, and the Tracker's per-project mapping).
- What `taiichi init` does step by step, and how a repo is registered with the server and identified on later commands.
- The server's own config: data folder location, socket path, Forgejo URL and token.

From [Forgejo Tracker rules](07-forgejo-tracker-rules.md): the Tracker's per-project fields are its `owner/repo` (defaulting to the Forge's), the ready-label list, the position-label map (defaults `taiichi/plan` through `taiichi/abandoned`, each optional), and on/off switches for the Pickup, Done, and Abandoned comments. The server checks that configured labels exist but never creates them; whether `taiichi init` creates them is decided here.

From [Forgejo Forge](08-forgejo-forge.md): the server's config adds the Forge poll interval, server-wide (default 60 seconds). The token must belong to a dedicated bot account that never reviews, and `taiichi status` shows which Forgejo account Taiichi acts as.

## Answer

Settled with the user. Glossary: **Repo** added (a git repository that has opted in with `taiichi init`), with "project" to avoid.

**The repo config: `taiichi.toml`**
- Lives at the Repo root, in TOML. TOML allows comments and is what this repo already uses for tool config. Elixir needs one Hex package for it.
- Only the server reads it. Polling and write-back run without a CLI request, so the CLI never parses the file; it only finds the Repo and sends requests.
- Unknown keys are an error, so a typo fails loudly instead of being ignored the way Forgejo ignores unknown label names.

```toml
[forge]
type = "forgejo"
url  = "https://forgejo.example"
repo = "ngs/widgets"

[tracker]
type = "forgejo"
# url and repo default to the Forge's
ready_labels = ["taiichi/ready"]          # the default

[tracker.position_labels]   # each defaults to taiichi/<position>; false turns it off
plan     = "taiichi/plan"
build    = "taiichi/build"
evaluate = "taiichi/evaluate"
ship     = "taiichi/ship"
done     = "taiichi/done"

[tracker.comments]          # each defaults to true
pickup    = true
done      = true
abandoned = true
```

- `type` selects the adapter; only `forgejo` exists in v1.
- The Forgejo instance URL lives here, next to `owner/repo`: together they say where the Repo lives. Tokens stay on the server.
- **No `abandoned` position label.** Dropped from [Forgejo Tracker rules](07-forgejo-tracker-rules.md): an issue showing both a ready label and `taiichi/abandoned` is confusing, and the Abandoned comment and the Job history already record it. Abandon removes the current position label, restores the ready labels, and posts the Abandoned comment.
- `taiichi/done` stays: nothing replaces it, and it marks a closed issue as shipped through Taiichi.

**When the config is read**
- Each Job keeps a copy of the Tracker and Forge settings from Pickup, as it does with the Ticket Snapshot. The Job runs under that copy, so background work never depends on what is checked out in the clone. Fixing a running Job's config means abandoning it and picking the Ticket up again.
- Only `init`, `tickets`, and `pickup` read `taiichi.toml`, fresh from the clone's working tree.

**The server's config**
- `$XDG_CONFIG_HOME/taiichi/server.toml` holds only `poll_interval` (seconds, default 60) and `[tokens]`, keyed by instance URL. URLs are compared after normalizing (lowercase scheme and host, no trailing slash).
- The server refuses to start if `server.toml` can be read by group or others, since it holds tokens. Tokens can't be set through an environment variable.
- The data folder is `$XDG_DATA_HOME/taiichi` (default `~/.local/share/taiichi`), with no override. `XDG_DATA_HOME` already moves it, for tests too.
- The socket path comes from `TAIICHI_SOCKET`, else `$XDG_RUNTIME_DIR/taiichi/taiichi.sock`, else `~/.local/state/taiichi/taiichi.sock`. CLI and server work it out the same way, and `server.toml` can't set it, so the CLI never has to read the server's config. The server creates the socket's folder with mode `0700`.

**Identifying a Repo**
- The CLI sends the clone's absolute git common dir (`git rev-parse --path-format=absolute --git-common-dir`), and the server maps it to the Repo. Every worktree of a clone shares the common dir, so this works from the main checkout, from Taiichi's Job worktrees, and from any worktree the user makes. The server's Repo id can't go in `taiichi.toml`, because that file is committed and shared.
- A server holds at most one Repo per Forge location (instance URL plus `owner/repo`). Two clones of one Forgejo repo would compete for the same Tickets and Shipments.
- "At most one active Job per Ticket" holds across the whole server, keyed by the Tracker's location plus the Ticket id, because two Repos can share one Tracker repo.

**`taiichi init`, step by step**
1. **Write the file if it's missing.** The server writes it, so the schema lives in one place, with every default spelled out. `forge.url` and `forge.repo` come from the `origin` remote when it's an HTTP(S) URL; otherwise `--forge-url` and `--forge-repo` are required. `--ready-label` (repeatable) is optional and replaces the default `taiichi/ready`. An existing file is used as is.
2. **Check, in order, stopping at the first failure:** the file matches the schema; `server.toml` has a token for every instance the file names; Forgejo answers, returning the token's account (`GET /user`) and the repo with its default branch (`GET /repos/{o}/{r}`); every configured label exists. Token scopes aren't checked, since Forgejo doesn't report them.
3. **Labels.** `init` offers to create missing position labels, and `taiichi/ready` when the ready list is just the default, all as exclusive labels in the `taiichi` scope. It asks first; `--yes` skips. Any other missing ready label is refused by name, since it's more likely a typo. The server itself never creates labels.
4. **Register, by case:**
   - *Already registered* (the common dir is known): changes nothing, or updates the Repo's stored Forge location if `taiichi.toml` now names a different one. A location another Repo holds is refused.
   - *Moved clone* (unknown common dir, but a Repo with this Forge location exists and its recorded path is gone): updates the path and runs `git worktree repair`, so active Jobs' worktrees work again. Moving or renaming a parent folder is the same case.
   - *New:* registers a new Repo. If another Repo with this Forge location still exists at its path, it's refused.
5. **Report:** prints the Repo, the Forgejo account Taiichi acts as, and a reminder to commit `taiichi.toml`. `init` never commits it.

There's no separate re-init command; running `init` again is safe. If `taiichi.toml` names a different Forge location than the one stored, `tickets` and `pickup` refuse with "Forge location changed; run `taiichi init`".

**Pickup checks the branch name**
- Before registering a Job, Pickup checks that its branch name is free, both in the clone and on the Forge. If it isn't, Pickup refuses and names the branch, instead of failing halfway through. This matters after a Repo is removed by hand (below), since a fresh Repo's Job numbers restart at 1.

**Removing a Repo by hand**
- v1 has no command for it. For it to work by hand, [Server internals](12-server-internals.md) must keep everything about a Repo (registration, Jobs, history, worktrees) in one folder under the data folder. Then removal is:
  1. abandon each active Job;
  2. stop the server;
  3. delete the Repo's folder;
  4. run `git worktree prune` in the clone;
  5. start the server.

**Post-v1** (outside this map, as `needs-triage` issues):
- [CLI init wizard](../../taiichi-post-v1/issues/01-cli-init-wizard.md)
- [List and remove Repos](../../taiichi-post-v1/issues/02-list-and-remove-repos.md)

## Comments

**From [Server internals](12-server-internals.md):** a Repo is no longer one folder; the server's data is in SQLite. Removing a Repo by hand is: abandon its active Jobs, stop the server, `DELETE FROM repos WHERE id = …` (cascades to its Jobs and their history), delete those Jobs' worktrees under `$XDG_DATA_HOME/taiichi/worktrees/`, run `git worktree prune` in the clone, start the server. Job numbers are server-wide and never reused, so they no longer restart after a removal; Pickup's branch-name check stays as a guard.

**From [A Ticket changing under a running Job](17-ticket-changing-under-a-job.md):** `init` warns and asks when a configured exclusive label shares its scope with exclusive labels outside the config, naming the labels Forgejo would strip at every move. `--yes` skips the prompt. It never refuses.
