# Follow a Job live

Status: needs-triage

A person may want to see a Job change as it happens, without running `taiichi show` again. For example: Job 7 waits at Ship, a reviewer requests changes at 14:02, the Job's checker sees it within one `poll_interval`, and the CLI prints "Job 7: Shipment Refused, now at Build 2 (2 Feedback items)" as soon as the change is saved. v1 leaves this out, as settled in [Watching a Job live](../../taiichi-v1/issues/18-watching-a-job-live.md). The post-v1 TUI is the likely first user, but a `--follow` in the plain CLI could come first.

What v1 already gives:
- **Only the Job's checker changes a Job without a command.** Pickup, `submit`, and `abandon` answer only once their move is done. The checker can move a Job (Accepted to Done from any Stage, Refused back to Build from Ship) or raise a flag (a Closed Shipment, a changed Ticket, new comments, commits the worktree lacks, failing checks or outbox sends). See [A Ticket changing under a running Job](../../taiichi-v1/issues/17-ticket-changing-under-a-job.md).
- **Where events would attach:** the Job's process calls Tracker write-back after every save, with the old and new Job. A broadcast can go at the same call site. See [Server internals](../../taiichi-v1/issues/12-server-internals.md).
- **The API can grow without breaking:** a streaming endpoint is a new path beside the `POST /v1/<method>` calls, so no existing method changes. See [HTTP API shape](../../taiichi-v1/issues/10-http-api-shape.md).
- **A Go client works over the Unix socket**, checked with a small program in [Watching a Job live](../../taiichi-v1/issues/18-watching-a-job-live.md): gorilla/websocket through `Dialer.NetDialContext`, coder/websocket through `DialOptions.HTTPClient` with a custom `http.Transport.DialContext`, and server-sent events through `net/http` with the same `DialContext`. The server sees `Host: taiichi`.

What to settle then:
- **Server-sent events or a WebSocket.** Bandit serves both. One-way push needs only server-sent events.
- **How messages are framed**, given the API is plain RPC with JSON.
- **Where events come from:** a broadcast at the write-back call site, or domain events, which would let the outbox be filled from them (see the comment from [An operation stopping partway](../../taiichi-v1/issues/15-operation-stopping-partway.md) on [Watching a Job live](../../taiichi-v1/issues/18-watching-a-job-live.md)).
- **Whether a future Stage executor shares the feed** to learn when a Stage Input is ready, as asked in [The executor seam](../../taiichi-v1/issues/19-executor-seam.md).
- **What a client sees after reconnecting:** missed changes, or only the Job's current state.
