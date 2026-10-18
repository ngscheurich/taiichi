# An operation stopping partway

Type: grilling
Status: open

## Question

How does each multi-step operation behave when it stops partway, because Forgejo can't be reached or the server stops between steps?

- **Pickup:** the Job is saved, then the branch and worktree are created, then the Tracker write-back runs. What if the worktree or the labels never happen?
- **Ship:** push, open or update the pull request, then save the Manifest. If the pull request exists but no Manifest was saved, a retry would open a second one.
- **Abandon:** close the pull request, restore the labels, remove the worktree, record the end. In what order, and what if one step fails?
- **Tracker write-back after a move** (`Tracker.WriteBack.moved(old_job, new_job)`, called after every save): the move is kept but the labels can go stale. Is it retried, and how? An outbox table, written in the same transaction as the move and sent until it succeeds, is the likely shape.
- **Polling while Forgejo is down:** what the poller does, and what `show` and `jobs` say.
- What the CLI tells the user in each case, and which `502 forgejo_failed` responses leave work half done.

Settled constraints from [Server internals](12-server-internals.md): no transaction stays open across a git or Forgejo call; the API is synchronous for `pickup`, Ship's `submit`, and `abandon` ([HTTP API shape](10-http-api-shape.md)).
