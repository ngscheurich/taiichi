# Gaps found assembling the v1 spec

Type: grilling
Status: open

## Question

Assembling [the v1 spec](../spec.md) in [Assemble the v1 spec](13-assemble-the-v1-spec.md) turned up five points the settled tickets leave open or contradict. How is each settled?

1. **Reading the Ticket Snapshot and Feedback.** `document` takes `{job, stage, attempt}` and returns that Attempt's Stage Output, and `cat <stage>` calls it. Nothing else reads a stored document. So once `input` has moved past an Attempt, nothing reads the Ticket Snapshot (Build's Input leaves it out) or an earlier Attempt's Feedback, and on an ended Job `input` is `409 job_ended`. For example, after Job 7 is Done, `taiichi cat` can print its Blueprint but not the Ticket it was built from, nor the Feedback that sent Ship 1 back. Should `document` and `cat` reach them, and how? ([HTTP API shape](10-http-api-shape.md), [The executor seam](19-executor-seam.md))
2. **The Job number in Pickup's branch check.** Pickup checks the branch name `taiichi/<ticket>-j<job>-<slug>` is free before its save, but the Job number is the `jobs.id` the database assigns at that save. How does the check know the number? ([`taiichi init` and configuration](09-init-and-configuration.md), [Server internals](12-server-internals.md), [An operation stopping partway](15-operation-stopping-partway.md))
3. **A worktree kept at Done.** Done keeps a worktree with uncommitted changes, since "Taiichi never deletes uncommitted work unless a human says so". But an ended Job takes no commands, so the Operator has no way to say so: the worktree and the Job's `worktree` field stay forever unless removed by hand. What removes it? ([An operation stopping partway](15-operation-stopping-partway.md))
4. **The account in `status`.** [Forgejo Forge](08-forgejo-forge.md) says `taiichi status` shows which Forgejo account Taiichi acts as, so a self-review clash can be spotted. The `status` method in [HTTP API shape](10-http-api-shape.md) returns `{server: {version, api}, repo}` with no account, and getting it means a Forgejo call (`GET /user`). Only `init` returns `account`. Which holds?
5. **`init`'s exclusive-scope prompt over the API.** [A Ticket changing under a running Job](17-ticket-changing-under-a-job.md) has `init` warn and ask when a configured exclusive label shares its scope with labels outside the config, with `--yes` skipping. The missing-labels prompt travels as `409 labels_missing` and a second call with `create_labels: true`; the scope warning has no code or param. For example, with `ready_labels = ["status/ready"]` and a `status/blocked` label in the repo, how does the server's warning reach the CLI's question, and how does the answer come back?

On resolution, update each **Open** note in [the v1 spec](../spec.md) and remove it, and update the spec's other sections the answers touch.
