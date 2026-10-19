# Watching a Job live

Type: grilling
Status: open

## Question

Does the v1 CLI follow a Job's progress as it happens, and if so, what does that need from the server?

- Whether v1 needs it at all, given the CLI has no `--follow` today ([CLI commands](06-cli-commands.md)) and the TUI is post-v1.
- If it does: server-sent events (one-way) or a WebSocket (two-way) over the Unix socket, and how messages are framed, since the API is plain RPC ([HTTP API shape](10-http-api-shape.md)).
- Where notifications come from: [Server internals](12-server-internals.md) put Tracker write-back as a direct call after each save, and left domain events for features like this to bring in.

Unchecked fact a decision may wait on: whether a Go WebSocket client can dial a Unix socket ([CLI language](11-cli-language.md)).

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** the server has an `outbox` of effects, sent by one process after each save, but no events: the Job's Attempts, Stage Outputs and `ended_at` record the moves. If watching a Job needs a feed of moves, this ticket brings events in, and the outbox could be filled from them.
