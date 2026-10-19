# The executor seam

Type: grilling
Status: open

## Question

Where does a future Stage executor, human or AI, plug in, and what does v1 define for it without using it for AI?

- How an executor learns a Stage Input is ready, and how it fetches it (the `input` method returns the parts as documents, per [HTTP API shape](10-http-api-shape.md)).
- How it submits a Stage Output, and whether the `submit` method as it stands is the seam.
- What v1 must build now so the future harness (a long-running TypeScript process hosting pi durable, a client of the server's API like the CLI, per [CLI language](11-cli-language.md)) needs no server change, and what can wait.
- Whether it needs notifications, and how that relates to [Watching a Job live](18-watching-a-job-live.md).

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** no events exist yet; see the comment on [Watching a Job live](18-watching-a-job-live.md). An executor that needs to learn a Stage Input is ready would be the second consumer of a feed of moves.
