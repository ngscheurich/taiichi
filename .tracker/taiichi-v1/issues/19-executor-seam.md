# The executor seam

Type: grilling
Status: resolved

## Question

Where does a future Stage executor, human or AI, plug in, and what does v1 define for it without using it for AI?

- How an executor learns a Stage Input is ready, and how it fetches it (the `input` method returns the parts as documents, per [HTTP API shape](10-http-api-shape.md)).
- How it submits a Stage Output, and whether the `submit` method as it stands is the seam.
- What v1 must build now so the future harness (a long-running TypeScript process hosting pi durable, a client of the server's API like the CLI, per [CLI language](11-cli-language.md)) needs no server change, and what can wait.
- Whether it needs notifications, and how that relates to [Watching a Job live](18-watching-a-job-live.md).

## Answer

Settled with the user. Glossary: **Operator** added, the human in the loop. No ADR: nothing here is hard to reverse.

**The bar: nothing that would be costly to add later**
- The ticket asked what v1 must build so the harness needs no server change. That bar was dropped, because it would pull post-v1 work into v1, and Gates change the server anyway.
- v1 builds nothing for a future executor unless adding it later would be costly, meaning it would break stored data or change an existing API method. Anything that can come later as a new method, a new nullable field or a new endpoint waits.

**v1 changes nothing for an executor**
- **Finding and reading work:** `jobs` or `show` (`awaiting: "stage_output"`), then `input` for the current Attempt's parts, with the work done in the Job's `worktree`. It's the same route the CLI takes. The harness runs on the same host as the same user, so it reaches the socket and the worktree like the CLI does.
- **Handing back:** `submit` as it stands, with `stage` and `attempt` taken from `input`'s reply. If a human and an executor race on the same Attempt, the first save wins and the second gets `409 job_moved`. Build's clean-worktree check, Evaluate's moved-head check and Ship's push behave the same whoever calls.
- **`input` answers after the Output is in.** At Ship, once the Manifest is in, `input` still returns Ship's Input while the Shipment is under review. A second Manifest is refused with `409 not_awaiting_output`, so a caller checks `awaiting` on the Job first, as the CLI does.

**One gap filled for v1**
- `input` on a Done or Abandoned Job is `409 job_ended`. Such a Job has no current Attempt (`attempt` is `null`), and its earlier Outputs stay readable through `document`.

**Held for the post-v1 AI session**

The user wants every AI-related decision held for a separate grilling session after v1, with no ticket written for now. These are the open items:
- An `awaiting` field on `input`'s reply, so an executor needs one call, and its shape (the Job's values, or a boolean).
- How an executor learns a Stage Input is ready. Moves it makes itself come back in `submit`'s reply; Pickup and the checker's moves (Refused back to Build, Accepted to Done) would need polling `jobs` or a feed. `jobs` takes a `git_dir`, so serving several Repos needs a server-wide listing. A feed would join [Follow a Job live](../../taiichi-post-v1/issues/05-follow-a-job-live.md).
- Recording who wrote an Output. That would be a new key in new documents' `meta`; documents are write-once, so old ones lack it, and a missing key means a human wrote it.
- The word for whatever writes Stage Outputs. Executor (this ticket), submitter ([Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md)) and harness (the TypeScript process, [CLI language](11-cli-language.md)) are in use. Operator is taken: it means the human in the loop.

**Operator**
- The user defined an Operator as the human in the loop, now in `GLOSSARY.md`. [Gates](../../taiichi-post-v1/issues/04-gates.md), the map's Out of scope line for Gates, and [An operation stopping partway](15-operation-stopping-partway.md) called an AI agent "a second operator". All three now say "once an AI agent drives some Stages".

**Changes to earlier decisions**
- [HTTP API shape](10-http-api-shape.md): `input` on an ended Job is `409 job_ended`.

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** no events exist yet; see the comment on [Watching a Job live](18-watching-a-job-live.md). An executor that needs to learn a Stage Input is ready would be the second consumer of a feed of moves.

**From [Watching a Job live](18-watching-a-job-live.md):** v1 doesn't follow a Job live. That's post-v1, tracked as [Follow a Job live](../../taiichi-post-v1/issues/05-follow-a-job-live.md), and v1 reserves nothing for it. So watching no longer gives v1 a reason for a feed of moves; if an executor needs notifications, it is now the only thing asking for them. A Go client can use server-sent events or a WebSocket over the Unix socket, checked there.
