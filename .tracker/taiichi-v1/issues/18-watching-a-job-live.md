# Watching a Job live

Type: grilling
Status: resolved

## Question

Does the v1 CLI follow a Job's progress as it happens, and if so, what does that need from the server?

- Whether v1 needs it at all, given the CLI has no `--follow` today ([CLI commands](06-cli-commands.md)) and the TUI is post-v1.
- If it does: server-sent events (one-way) or a WebSocket (two-way) over the Unix socket, and how messages are framed, since the API is plain RPC ([HTTP API shape](10-http-api-shape.md)).
- Where notifications come from: [Server internals](12-server-internals.md) put Tracker write-back as a direct call after each save, and left domain events for features like this to bring in.

Unchecked fact a decision may wait on: whether a Go WebSocket client can dial a Unix socket ([CLI language](11-cli-language.md)).

## Answer

Settled with the user. **v1 doesn't follow a Job live. It's post-v1, tracked as [Follow a Job live](../../taiichi-post-v1/issues/05-follow-a-job-live.md).**

**Why it can wait**
- Pickup, `submit`, and `abandon` answer only once their move is done, so a person already knows the result of every change they start.
- Only the Job's checker changes a Job without a command, once every `poll_interval` (60 s by default). It moves a Job when the Shipment is Accepted (Done, from any Stage) or Refused (back to Build, from Ship), and raises flags for a Closed Shipment, a changed Ticket, new comments, commits the worktree lacks, and failing checks or outbox sends. `show` and `jobs` already print all of these, and reads come straight from SQLite, so `watch -n 30 taiichi show 7` comes close to a live view.

**What v1 reserves: nothing**
- No `--follow` command, no events table, no event bus or PubSub, and no placeholder streaming endpoint.
- Tracker write-back stays the direct call after each save, as settled in [Server internals](12-server-internals.md). That call site is where a broadcast attaches later.
- A streaming endpoint added later is a new path beside the `POST /v1/<method>` calls, so v1's API in [HTTP API shape](10-http-api-shape.md) doesn't change.

**The unchecked fact, now checked:** a Go client can dial the Unix socket for both push methods. A small program ran all three against a socket with the placeholder host `taiichi`:
- gorilla/websocket v1.5.3 through `Dialer.NetDialContext`, with `Proxy` left nil;
- coder/websocket v1.8.15 through `DialOptions.HTTPClient` with a custom `http.Transport.DialContext`;
- server-sent events through `net/http` with the same `DialContext`, reading the body line by line.

The server sees `Host: taiichi`, and no TLS is needed. Sources: [gorilla `Dialer`](https://pkg.go.dev/github.com/gorilla/websocket#Dialer), [coder `DialOptions`](https://pkg.go.dev/github.com/coder/websocket#DialOptions).

**Changes to other tickets**
- [The executor seam](19-executor-seam.md): watching a Job no longer gives v1 a reason for a feed of moves. If an executor needs notifications, it is now the only thing asking for them.

No glossary change and no ADR: no domain terms moved, and leaving the feature out is easy to reverse.

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** the server has an `outbox` of effects, sent by one process after each save, but no events: the Job's Attempts, Stage Outputs and `ended_at` record the moves. If watching a Job needs a feed of moves, this ticket brings events in, and the outbox could be filled from them.
