# Taiichi v1

Label: wayfinder:map

## Destination

A v1 spec for Taiichi, ready to hand off and slice into implementation issues. Taiichi is a factory that picks up Tickets from a Tracker, registers each as a Job, moves it through the Plan → Build → Evaluate → Publish Pipeline, saves every Stage Input and Stage Output, and gives each Job its own branch and worktree. In v1, humans write the Stage Outputs; no AI is involved.

## Notes

- **Domain:** a long-running Elixir server (the Mix app in this repo) with a separate, non-Elixir CLI as its client. Use the terms in `GLOSSARY.md`; update it through the domain-modeling skill when a term changes.
- **Skills:** grilling tickets call the Skill tool for `grilling` and `domain-modeling`; prototype tickets for `prototype`; research tickets for `research`.
- **Tracker conventions:** `docs/agents/issue-tracker.md` ("Wayfinding operations").
- **Environment:** the v1 Tracker and Forge is a Forgejo instance running on the user's host. Agent sandboxes can't reach it; the user can arrange access if a ticket needs the live instance.
- **Standing preferences:** put each question to the user and wait for an explicit answer before moving past it. Base recommendations only on what the user has said or what can be verified; don't invent premises such as scale. In `GLOSSARY.md`, `_Avoid_` lists hold only words people would reach for by habit.

## Decisions so far

- [Chart the v1 scope](issues/01-chart-the-v1-scope.md): the v1 boundary, server and CLI split, persistence, Pipeline rules, and Forgejo as both Tracker and Forge, settled while charting.
- [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md): yes, with four adjustments: Report adds Pending and Closed outcomes and is scoped to the latest Publish, Publish updates an existing Shipment, and feedback is a list of items.
- [Forgejo API for the Tracker and Forge](issues/02-forgejo-api.md): everything v1 needs exists; Refused is a request-changes review, Accepted is merged, authors can't review their own pull requests, and one filtered issues query polls cheaply.
- [HTTP over a Unix socket](issues/04-http-over-unix-socket.md): Bandit serves it with `ip: {:local, path}, port: 0`; the server must clear stale socket files and keep the socket in a private directory; curl, Go, Bun, Node and Rust clients support it, Zig's standard client doesn't.
- [Revisit the transport and protocol](issues/14-revisit-transport-and-protocol.md): kept HTTP + JSON over a Unix socket; the socket keeps out other users and browsers for free, Bandit and Plug do the server work, and a WebSocket covers two-way push if it's ever needed.

## Not yet specified

- **The executor seam.** Where a future Stage executor, human or AI, plugs in: how it's handed a Stage Input and how it submits a Stage Output. v1 defines the seam but doesn't use it for AI.
- **A Ticket changing under a running Job.** What happens when its Ticket is edited, closed, or relabelled mid-Job.
- **Forgejo being unreachable.** How Pickup, write-back, publishing, and polling behave when the Forge or Tracker can't be reached.
- **Watching a Job live.** Whether the CLI can follow a Job's progress as it happens, and what that needs from the server. HTTP can push with server-sent events (one-way) or a WebSocket (two-way); whether the CLI's language has a WebSocket client that works over a Unix socket is unchecked.

## Out of scope

- **Other Tracker and Forge adapters.** GitHub, GitLab, Linear, the repo-local `.tracker/` files, and a manual Forge. v1 ships only the Forgejo adapters; the interfaces stay general enough for these, which [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md) checks.
- **AI executing Stages.** v1 runs with humans writing Stage Outputs.
