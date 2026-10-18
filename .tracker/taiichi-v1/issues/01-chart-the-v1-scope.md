# Chart the v1 scope

Type: grilling
Status: resolved

## Question

What is the v1 boundary for Taiichi, and which foundational decisions can be settled while charting the map?

## Answer

Settled with the user while charting. Terms are defined in `GLOSSARY.md`.

**Scope**
- The destination is a v1 spec, ready to hand off.
- v1 covers pulling Tickets, registering Jobs, enforcing the Pipeline (including both loops back to Build), saving every Stage Input and Stage Output, and creating a branch and worktree per Job.
- No AI in v1: humans write the Stage Outputs. Eventually AI agents do the Stages, so Stage data must suit both machines and people.
- Remote deployment is ignored.

**Architecture**
- One long-running Elixir server per machine serves many repos. The server is started by hand in a terminal; the CLI says clearly when it can't reach it.
- The CLI is a separate client, almost certainly not written in Elixir (language: see [CLI language](11-cli-language.md)).
- Transport: HTTP + JSON over a Unix socket, if Bandit supports it (see [HTTP over a Unix socket](04-http-over-unix-socket.md)); otherwise a Unix socket with JSON Lines.
- The server is the only writer of Job data. The CLI sends Stage Output *contents*, never file paths.

**Repos**
- Taiichi works with any repo. A repo opts in with `taiichi init`, which writes a committed config file naming the repo's Tracker and Forge and their options, and registers the repo with the server for a stable id. Other commands fail in a repo that hasn't been initialized. The config file's format is open (see [`taiichi init` and configuration](09-init-and-configuration.md)).
- The CLI works out the target repo from its current directory.
- The Forgejo access token lives in the server's config or environment, never in the repo. Whose account it belongs to is the user's choice.

**Jobs and the Pipeline**
- Pickup is a human running a CLI command with a Ticket id. The factory never picks up Tickets on its own.
- Pickup order: register the Job (it enters Plan), copy the Ticket into the Plan Stage Input once (no re-sync later), then create the branch and worktree.
- Branches are named from the Ticket under a `taiichi/` prefix, branched from the tip of the repo's default branch at Pickup. Worktrees live in the server's data folder.
- Many Jobs can run in parallel; at most one active Job per Ticket.
- Submitting a valid Stage Output ends the Attempt and moves the Job on right away. An Evaluation is routed by its Verdict: pass goes to Publish, fail goes back to Build.
- After the Manifest is submitted, the Job waits at Publish until its Shipment is Accepted (the Job moves to Done) or Refused (the Job returns to Build, with the reviewer's feedback in Build's Stage Input).
- Every Attempt is kept; nothing is overwritten. A Build retry's Stage Input includes the failed Evaluation.
- A human can abandon a Job from any Stage. At Done or Abandoned, the worktree is removed; the branch and the Job's history are kept.

**Stage data**
- Saved on the filesystem, in a data folder the server owns outside every repo.
- Format: Markdown with YAML front matter.
- Each Stage Output has required front-matter fields, and the server rejects submissions missing them (fields: see [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md)).

**Trackers and Forges**
- Both are adapters. The Tracker adapter owns which Tickets are eligible for Pickup, how Job positions map to Ticket states, and how write-back happens; it is called on every Job position change. Two projects on the same Tracker can map differently, so the mapping is configurable per project.
- The core owns only rules that hold for every Tracker: one active Job per Ticket, and the Pipeline's transitions.
- A Forge does two things: publish a Job's branch as a Shipment, producing the Manifest; and report the Shipment's outcome. The factory learns the outcome by polling.
- v1 uses a locally hosted Forgejo instance as both Tracker and Forge.

## Comments

**From [Server internals](12-server-internals.md):** Stage data is no longer saved as Markdown files with YAML front matter in a data folder. The server keeps it in one SQLite database through Ecto (`$XDG_DATA_HOME/taiichi/taiichi.db`), and worktrees stay on disk. Jobs are numbered across the server, not per repo. See [SQLite for the server's data](../../../docs/adr/0001-sqlite-for-server-data.md).
