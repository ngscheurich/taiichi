# Server internals

Type: grilling
Status: resolved
Blocked by: 05

## Question

How is the server built inside?

- The process model for Jobs (for example, one `gen_statem` per Job enforcing the Pipeline) and for polling.
- The on-disk layout of a Job's data folder: a folder per Attempt, numbered within its Stage, holding the Stage Input bundle and the Stage Output (see [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md)).
- How state is rebuilt from disk after a restart, and how writes stay safe if the server stops mid-write.
- How "one active Job per Ticket" is enforced.

## Answer

Settled with the user. Glossary: a Repo no longer numbers its Jobs; each Job has a number unique across the server and never reused. ADR: [SQLite for the server's data](../../../docs/adr/0001-sqlite-for-server-data.md).

**Scope of this ticket**
- This ticket covers the server's own data: each write lands whole or not at all, and a restart reads back a consistent picture.
- An operation that stops partway across git and Forgejo (Pickup, Ship, Abandon) is left to the map's fog, widened from "Forgejo being unreachable" to "an operation stopping partway".

**Storage: SQLite through Ecto**
- Replaces the charted Markdown files with YAML front matter. One database, `$XDG_DATA_HOME/taiichi/taiichi.db`, with no YAML anywhere: a document's `meta` is a JSON column, which the API already returns as `meta`.
- Worktrees stay on disk at `$XDG_DATA_HOME/taiichi/worktrees/<job>`.
- Connection settings, set explicitly: WAL journal mode, a busy timeout so writers queue, `synchronous = NORMAL` (after a power loss the newest commit may be missing, never a corrupt database), foreign keys on.
- **No transaction stays open across a git or Forgejo call.** For Ship: push and open the pull request first, then save the Manifest in one transaction.
- PostgreSQL was weighed and rejected (see the ADR).

**Tables**
- Every table has an integer `id` primary key, `AUTOINCREMENT` so ids are never reused.
- **`repos`:** `git_dir` (unique), main worktree path, Forge and Tracker locations, `registered_at`. Unique on Forge URL plus Forge repo.
- **`jobs`:** `id` *is* the Job number people see, unique across the server. `repo_id`, Ticket id, Tracker URL and repo, `settings` (JSON copy of the Tracker and Forge settings from Pickup), branch, worktree (unique, null once removed), `picked_up_at`, `ended_at`, `ended_as` (`done`, `abandoned`, or null). A partial unique index on (Tracker URL, Tracker repo, Ticket id) `WHERE ended_at IS NULL` enforces one active Job per Ticket across the whole server.
- **`attempts`:** `job_id`, `seq` (order within the Job), `stage`, `number` (within the Stage), `started_at`, and on Ship Attempts `last_report` (the latest Report: `pending`, `accepted`, `refused`, or `closed`) and `reported_at`, written by the poller on each check. Unique on (Job, `seq`) and on (Job, Stage, number). `CHECK (stage = 'ship' OR (last_report IS NULL AND reported_at IS NULL))` keeps both off every other Stage's Attempts.
- **`documents`:** `kind`, `meta` (JSON), `body`, `created_at`, and either `job_id` (the Ticket Snapshot) or `attempt_id`. An Attempt has at most one Stage Output. Feedback hangs off the Build Attempt it opens, and its `meta` names the Ship Attempt it came from. The Ship marker and the starting feedback cursor are in the Manifest's `meta`; each Feedback records the cursor it advanced to, and the current cursor is the latest of these. `body` is left out of queries unless asked for (`load_in_query: false`), so list queries don't load large bodies.
- **`input_parts`:** (`attempt_id`, `position`, `document_id`), written in the same transaction that creates the Attempt. Every Stage Input is stored, not recomputed, so a later change to the rules never rewrites what an old Attempt received.
- **Position is derived, not stored:** the current Attempt is the Job's highest `seq`; whether it has a Stage Output tells "awaiting Stage Output" from "waiting on the Shipment"; `ended_as` gives Done or Abandoned.
- A Ship Attempt's `last_report` in `show`'s history is derived too: Refused if a Build Attempt follows it, Accepted if the Job is Done, otherwise the stored `last_report`.

**Rules across aggregates**
- One Repo per Forge location, one active Job per Ticket, and lookups by git common dir and by worktree path are unique indexes. A violation becomes a domain error such as `ticket_has_active_job`, never a raw constraint error.
- **Pickup:** (1) outside any transaction, fetch the Ticket, check eligibility, and check the branch name on the Forge; (2) one transaction started with `BEGIN IMMEDIATE`: insert the Job, the Ticket Snapshot, Plan 1, and its `input_parts`; (3) start the Job's process, which creates the branch and worktree and does the Tracker write-back.

**Processes**
- **One GenServer per active Job**, under a DynamicSupervisor, serializes that Job's writes: `submit`, `abandon`, and moves reported by the poller. Ended Jobs have no process.
- **Reads go straight to the database:** `jobs`, `show`, `input`, `document`, and `locate` never wait behind a slow Ship push.
- **One poller per Forge location** (instance URL plus `owner/repo`), taken from each waiting Job's own settings copy, so a Job keeps polling its own Shipment after `taiichi init` re-points its Repo. It starts when the first Job at that location starts waiting at Ship and stops when none are left. Its `since` value lives in memory; the startup check covers a restart. It tells the Job's process what a Report says, and the process makes the move.
- A `gen_statem` per Job and a process per Repo were rejected: a Position is data either way, and a per-Repo process would make one Job's push hold up another Job's calls.

**Startup**
1. Read `server.toml` and clear a stale socket file.
2. Open the database and run pending Ecto migrations, each in its own transaction; a failed migration stops the server with the error.
3. Start a process for each Job with `ended_at IS NULL`.
4. Start the pollers, which first check every Job waiting at Ship.
5. Open the socket last, so no request sees a half-started server.

**Structure (domain-driven design)**
```
Taiichi.Repos       Repo aggregate: init, registration, lookup by git dir
Taiichi.Jobs        Job aggregate: pickup, submit, abandon, and every read
  Jobs.Pipeline     pure rules: Position, moves, the parts of a Stage Input
  Jobs.Server       the per-Job process
  Jobs.Poller       one per Forge location; asks the Forge port, then moves Jobs
Taiichi.Tracker     port (behaviour) + Tracker.Forgejo adapter
Taiichi.Forge       port (behaviour) + Forge.Forgejo adapter
Taiichi.Git         branches, worktrees, push
Taiichi.API         Plug router: JSON and error codes; calls only Repos and Jobs
```
- **Job aggregate:** the Job with its Attempts, documents, and input parts. It changes only through its process, and each change is one transaction. **Repo aggregate:** the registration and its locations.
- **Ecto schemas are the domain model**, under one rule: only each context's store module calls `Repo`. The Pipeline is pure functions over the schema structs and returns them unsaved, so its tests build structs by hand with no database. Changesets stay on the schemas. Small functions per document kind read `meta` into atoms (for example, an Evaluation's verdict). A separate set of domain structs with a mapper was weighed and rejected: Ecto schemas are plain structs that can't save themselves, so the mapper bought only typed documents and storage isolation, at the cost of two structs per concept and casting outside the schemas.
- **Tracker write-back is a direct call, not domain events.** After every save, the Job's process calls `Tracker.WriteBack.moved(old_job, new_job)`, with `nil` as the old Job at Pickup. Write-back compares the two Positions and does the label swap, comments, and closing. It runs before the process replies, which keeps `pickup` and `abandon` synchronous as [HTTP API shape](10-http-api-shape.md) requires. Events were weighed: with one synchronous consumer they add indirection for nothing yet. The fog items that need "this happened to Job 7" (the executor seam, watching a Job live, retrying write-back) can attach at the same call site, or bring events in then.

**Changes to earlier decisions**
- [Chart the v1 scope](01-chart-the-v1-scope.md) and [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md): stored Stage data is rows in SQLite, not Markdown files with front matter.
- [CLI commands](06-cli-commands.md): Jobs are numbered across the server, not per Repo.
- [`taiichi init` and configuration](09-init-and-configuration.md): a Repo is no longer one folder. Removing it by hand is: abandon its active Jobs, stop the server, `DELETE FROM repos WHERE id = …` (cascades), delete those Jobs' worktrees, run `git worktree prune` in the clone, start the server. Pickup's branch-name check stays as a guard, though numbers no longer restart.
- [HTTP API shape](10-http-api-shape.md): Job methods (`show`, `input`, `document`, `submit`, `abandon`) take `{job, …}` without `git_dir`; `locate` takes `{worktree}`; `jobs`, `tickets`, `pickup`, `init`, and `status` keep `git_dir`. `taiichi show 12` works from any folder. Worktree paths are `…/taiichi/worktrees/<job>`.

## Comments

**From [`taiichi init` and configuration](09-init-and-configuration.md):**
- Keep everything about a Repo (registration, Jobs, history, worktrees) in one folder under the data folder, so a Repo can be removed by hand: stop the server, delete the folder, run `git worktree prune` in the clone.
- "At most one active Job per Ticket" holds across the whole server, keyed by the Tracker's location plus the Ticket id, not per Repo.
- A Repo is looked up by its clone's absolute git common dir, and the server holds at most one Repo per Forge location.
- Each Job stores a copy of its Repo's Tracker and Forge settings from Pickup.

**From [HTTP API shape](10-http-api-shape.md):**
- `submit` carries the Stage and Attempt it was written for; refuse a mismatch with the Job's current Position as `409 job_moved`.
- `locate` matches a worktree path exactly against Job worktrees, so the server needs a lookup from worktree path to Job.
- `init` reads the clone's `origin` remote and finds its main working tree from the common dir; the CLI sends neither.
- `document` with no Attempt means the latest Attempt of that Stage with a Stage Output, and `show`'s history needs each Attempt's start time and, for Ship, its `last_report` and Feedback item count.
