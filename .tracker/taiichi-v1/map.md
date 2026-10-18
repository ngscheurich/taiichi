# Taiichi v1

Label: wayfinder:map

## Destination

A v1 spec for Taiichi, ready to hand off and slice into implementation issues. Taiichi is a factory that picks up Tickets from a Tracker, registers each as a Job, moves it through the Plan → Build → Evaluate → Ship Pipeline, saves every Stage Input and Stage Output, and gives each Job its own branch and worktree. In v1, humans write the Stage Outputs; no AI is involved.

## Notes

- **Domain:** a long-running Elixir server (the Mix app in this repo) with a separate, non-Elixir CLI as its client. Use the terms in `GLOSSARY.md`; update it through the domain-modeling skill when a term changes.
- **Skills:** grilling tickets call the Skill tool for `grilling` and `domain-modeling`; prototype tickets for `prototype`; research tickets for `research`.
- **Tracker conventions:** `docs/agents/issue-tracker.md` ("Wayfinding operations").
- **Environment:** the v1 Tracker and Forge is a Forgejo instance running on the user's host. Agent sandboxes can't reach it; the user can arrange access if a ticket needs the live instance.
- **Standing preferences:** put each question to the user and wait for an explicit answer before moving past it. Base recommendations only on what the user has said or what can be verified; don't invent premises such as scale. In `GLOSSARY.md`, `_Avoid_` lists hold only words people would reach for by habit.

## Decisions so far

- [Chart the v1 scope](issues/01-chart-the-v1-scope.md): the v1 boundary, server and CLI split, persistence, Pipeline rules, and Forgejo as both Tracker and Forge, settled while charting.
- [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md): yes, with four adjustments: Report adds Pending and Closed outcomes and is scoped to the latest Ship, Ship updates an existing Shipment, and feedback is a list of items.
- [Forgejo API for the Tracker and Forge](issues/02-forgejo-api.md): everything v1 needs exists; Refused is a request-changes review, Accepted is merged, authors can't review their own pull requests, and one filtered issues query polls cheaply.
- [HTTP over a Unix socket](issues/04-http-over-unix-socket.md): Bandit serves it with `ip: {:local, path}, port: 0`; the server must clear stale socket files and keep the socket in a private directory; curl, Go, Bun, Node and Rust clients support it, Zig's standard client doesn't.
- [Revisit the transport and protocol](issues/14-revisit-transport-and-protocol.md): kept HTTP + JSON over a Unix socket; the socket keeps out other users and browsers for free, Bandit and Plug do the server work, and a WebSocket covers two-way push if it's ever needed.
- [Stage Inputs and Stage Outputs](issues/05-stage-inputs-and-outputs.md): Publish is renamed Ship; submitters write plain Markdown plus three values (body, verdict, Shipment title) and the server writes all front matter; a Stage Input is a per-Attempt bundle of new material and references; Evaluate and Ship see the whole history after Plan.
- [CLI commands](issues/06-cli-commands.md): flat verbs with one `submit` for every Stage, Jobs numbered per repo and implied inside their worktree, `--pass`/`--fail` for the Verdict, and `$EDITOR` opening with the Stage Input below a scissors line.
- [Forgejo Tracker rules](issues/07-forgejo-tracker-rules.md): eligible means open, a ready label, and no active Job; Pickup swaps ready labels for configurable position labels and Abandon restores them; branches are `taiichi/<ticket>-j<job>-<slug>`; comments on Pickup, Done, and Abandoned; the Tracker closes the issue on Done.
- [Forgejo Forge](issues/08-forgejo-forge.md): Ship pushes as the bot with a force-with-lease and replaces the description on rework; Refused is a request for changes since the latest Ship; Closed waits at Ship for a human; Feedback runs from a cursor of Forgejo ids; polling only while a Job waits at Ship.
- [`taiichi init` and configuration](issues/09-init-and-configuration.md): a server-read `taiichi.toml` with the Forgejo URL and labels, copied into each Job at Pickup; `server.toml` holds tokens by URL and the poll interval; Repos are found by git common dir, one per Forge location; `init` is safe to rerun and relinks moved clones; `taiichi/ready` is the default and `taiichi/abandoned` is dropped.
- [HTTP API shape](issues/10-http-api-shape.md): plain RPC, `POST /v1/<method>` with JSON and synchronous calls; every Job method takes `{git_dir, job}`, and `locate` finds the Job for a worktree; one Job object everywhere, with Position as flat fields; errors carry a string `code`, with 404 for anything missing and 502 for Forgejo.

## Not yet specified

- **The executor seam.** Where a future Stage executor, human or AI, plugs in: how it's handed a Stage Input and how it submits a Stage Output. v1 defines the seam but doesn't use it for AI.
- **A Ticket changing under a running Job.** What happens when its Ticket is edited, closed, or relabelled mid-Job, including someone re-adding a ready label that Pickup removed.
- **Forgejo being unreachable.** How Pickup, write-back, shipping, and polling behave when the Forge or Tracker can't be reached.
- **Watching a Job live.** Whether the CLI can follow a Job's progress as it happens, and what that needs from the server. HTTP can push with server-sent events (one-way) or a WebSocket (two-way); whether the CLI's language has a WebSocket client that works over a Unix socket is unchecked. The API is plain RPC, not JSON-RPC (see [HTTP API shape](issues/10-http-api-shape.md)), so how a WebSocket's messages are framed would be decided here.
- **A Shipment changing under a Job.** While the Job is away from Ship, its Shipment is merged, closed, or pushed to on the Forge. Report only runs at Ship, and the Pipeline reaches Done only from Ship. A push by someone else also leaves the worktree behind and throws off the next Build's starting commit; Ship's force-with-lease refuses to overwrite it (see [Forgejo Forge](issues/08-forgejo-forge.md)).

## Out of scope

- **Other Tracker and Forge adapters.** GitHub, GitLab, Linear, the repo-local `.tracker/` files, and a manual Forge. v1 ships only the Forgejo adapters; the interfaces stay general enough for these, which [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md) checks.
- **AI executing Stages.** v1 runs with humans writing Stage Outputs.
- **A CLI init wizard.** Post-v1, tracked as [CLI init wizard](../taiichi-post-v1/issues/01-cli-init-wizard.md). In v1, `taiichi init` takes flags, settled in [`taiichi init` and configuration](issues/09-init-and-configuration.md).
- **Listing and removing Repos.** Post-v1, tracked as [List and remove Repos](../taiichi-post-v1/issues/02-list-and-remove-repos.md), including what removal does to active Jobs, history, branches, and Tickets. In v1 a Repo is removed by hand, as described in [`taiichi init` and configuration](issues/09-init-and-configuration.md).
- **`--json` CLI output.** A fast-follow after v1, settled in [CLI commands](issues/06-cli-commands.md). In v1 the CLI prints for people, and scripts can call the server's HTTP API.
