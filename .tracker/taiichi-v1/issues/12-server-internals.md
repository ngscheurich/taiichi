# Server internals

Type: grilling
Status: open
Blocked by: 05

## Question

How is the server built inside?

- The process model for Jobs (for example, one `gen_statem` per Job enforcing the Pipeline) and for polling.
- The on-disk layout of a Job's data folder: a folder per Attempt, numbered within its Stage, holding the Stage Input bundle and the Stage Output (see [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md)).
- How state is rebuilt from disk after a restart, and how writes stay safe if the server stops mid-write.
- How "one active Job per Ticket" is enforced.

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
- `document` with no Attempt means the latest Attempt of that Stage with a Stage Output, and `show`'s history needs each Attempt's start time and, for Ship, its outcome and Feedback item count.
