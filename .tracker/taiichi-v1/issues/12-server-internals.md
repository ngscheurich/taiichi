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
