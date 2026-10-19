# Taiichi v1

Label: wayfinder:map

## Destination

A v1 spec for Taiichi, ready to hand off and slice into implementation issues. Taiichi is a factory that picks up Tickets from a Tracker, registers each as a Job, moves it through the Plan → Build → Evaluate → Ship Pipeline, saves every Stage Input and Stage Output, and gives each Job its own branch and worktree. In v1, humans write the Stage Outputs; no AI is involved.

## Notes

- **Domain:** a long-running Elixir server (the Mix app in this repo) with a separate, non-Elixir CLI as its client. Use the terms in `GLOSSARY.md`; update it through the domain-modeling skill when a term changes.
- **Skills:** grilling tickets call the Skill tool for `grilling` and `domain-modeling`; prototype tickets for `prototype`; research tickets for `research`.
- **Tracker conventions:** `docs/agents/issue-tracker.md` ("Wayfinding operations").
- **Environment:** the v1 Tracker and Forge is a Forgejo instance running on the user's host. Agent sandboxes can't reach it; the user can arrange access if a ticket needs the live instance.
- **Design approach:** lean heavily on domain-driven design: aggregates, explicit invariants, and a domain model kept apart from transport and persistence.
- **Standing preferences:** put each question to the user and wait for an explicit answer before moving past it. Base recommendations only on what the user has said or what can be verified; don't invent premises such as scale. In `GLOSSARY.md`, `_Avoid_` lists hold only words people would reach for by habit.

## Decisions so far

- [Chart the v1 scope](issues/01-chart-the-v1-scope.md): the v1 boundary, server and CLI split, persistence, Pipeline rules, and Forgejo as both Tracker and Forge, settled while charting.
- [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md): yes, with four adjustments: a Report can also say Pending or Closed and is scoped to the latest Ship, Ship updates an existing Shipment, and feedback is a list of items.
- [Forgejo API for the Tracker and Forge](issues/02-forgejo-api.md): everything v1 needs exists; Refused is a request-changes review, Accepted is merged, authors can't review their own pull requests, and one filtered issues query polls cheaply.
- [HTTP over a Unix socket](issues/04-http-over-unix-socket.md): Bandit serves it with `ip: {:local, path}, port: 0`; the server must clear stale socket files and keep the socket in a private directory; curl, Go, Bun, Node and Rust clients support it, Zig's standard client doesn't.
- [Revisit the transport and protocol](issues/14-revisit-transport-and-protocol.md): kept HTTP + JSON over a Unix socket; the socket keeps out other users and browsers for free, Bandit and Plug do the server work, and a WebSocket covers two-way push if it's ever needed.
- [Stage Inputs and Stage Outputs](issues/05-stage-inputs-and-outputs.md): Publish is renamed Ship; submitters write plain Markdown plus three values (body, verdict, Shipment title) and the server writes all front matter; a Stage Input is a per-Attempt bundle of new material and references; Evaluate and Ship see the whole history after Plan.
- [CLI commands](issues/06-cli-commands.md): flat verbs with one `submit` for every Stage, Jobs numbered per repo and implied inside their worktree, `--pass`/`--fail` for the Verdict, and `$EDITOR` opening with the Stage Input below a scissors line.
- [Forgejo Tracker rules](issues/07-forgejo-tracker-rules.md): eligible means open, a ready label, and no active Job; Pickup swaps ready labels for configurable position labels and Abandon restores them; branches are `taiichi/<ticket>-j<job>-<slug>`; comments on Pickup, Done, and Abandoned; the Tracker closes the issue on Done.
- [Forgejo Forge](issues/08-forgejo-forge.md): Ship pushes as the bot with a force-with-lease and replaces the description on rework; Refused is a request for changes since the latest Ship; Closed waits at Ship for a human; Feedback runs from a cursor of Forgejo ids; polling only while a Job waits at Ship.
- [`taiichi init` and configuration](issues/09-init-and-configuration.md): a server-read `taiichi.toml` with the Forgejo URL and labels, copied into each Job at Pickup; `server.toml` holds tokens by URL and the poll interval; Repos are found by git common dir, one per Forge location; `init` is safe to rerun and relinks moved clones; `taiichi/ready` is the default and `taiichi/abandoned` is dropped.
- [HTTP API shape](issues/10-http-api-shape.md): plain RPC, `POST /v1/<method>` with JSON and synchronous calls; every Job method takes `{git_dir, job}`, and `locate` finds the Job for a worktree; one Job object everywhere, with Position as flat fields; errors carry a string `code`, with 404 for anything missing and 502 for Forgejo.
- [CLI language](issues/11-cli-language.md): Go with Charm (Cobra, Fang, Huh, Lip Gloss, Glamour, and Bubble Tea later), chosen over TypeScript on Bun for polish, with a checkpoint before the TUI; `server/` and `cli/` sit side by side; `cat` becomes `cat <stage> [job] [-n <attempt>]`.
- [Server internals](issues/12-server-internals.md): SQLite through Ecto instead of Markdown files, with worktrees on disk; Job numbers are server-wide; one process per active Job serializes writes while reads hit the database; one poller per Forge location; Ecto schemas as the domain model behind store modules; Tracker write-back is a direct call after each save.

## Not yet specified

Nothing at the moment: every patch graduated into a ticket once [Server internals](issues/12-server-internals.md) closed: [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [Watching a Job live](issues/18-watching-a-job-live.md), and [The executor seam](issues/19-executor-seam.md).

## Out of scope

- **Other Tracker and Forge adapters.** GitHub, GitLab, Linear, the repo-local `.tracker/` files, and a manual Forge. v1 ships only the Forgejo adapters; the interfaces stay general enough for these, which [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md) checks.
- **AI executing Stages.** v1 runs with humans writing Stage Outputs.
- **A CLI init wizard.** Post-v1, tracked as [CLI init wizard](../taiichi-post-v1/issues/01-cli-init-wizard.md). In v1, `taiichi init` takes flags, settled in [`taiichi init` and configuration](issues/09-init-and-configuration.md).
- **Listing and removing Repos.** Post-v1, tracked as [List and remove Repos](../taiichi-post-v1/issues/02-list-and-remove-repos.md), including what removal does to active Jobs, history, branches, and Tickets. In v1 a Repo is removed by hand, as described in [`taiichi init` and configuration](issues/09-init-and-configuration.md).
- **`--json` CLI output.** A fast-follow after v1, settled in [CLI commands](issues/06-cli-commands.md). In v1 the CLI prints for people, and scripts can call the server's HTTP API.
- **The TUI.** Post-v1, in the CLI's language. Before it starts, the CLI language gets a second look, tracked as [Revisit the CLI language before the TUI](../taiichi-post-v1/issues/03-revisit-cli-language-before-tui.md).
