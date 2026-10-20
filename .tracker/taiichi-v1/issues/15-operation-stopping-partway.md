# An operation stopping partway

Type: grilling
Status: resolved

## Question

How does each multi-step operation behave when it stops partway, because Forgejo can't be reached or the server stops between steps?

- **Pickup:** the Job is saved, then the branch and worktree are created, then the Tracker write-back runs. What if the worktree or the labels never happen?
- **Ship:** push, open or update the pull request, then save the Manifest. If the pull request exists but no Manifest was saved, a retry would open a second one.
- **Abandon:** close the pull request, restore the labels, remove the worktree, record the end. In what order, and what if one step fails?
- **Tracker write-back after a move** (`Tracker.WriteBack.moved(old_job, new_job)`, called after every save): the move is kept but the labels can go stale. Is it retried, and how? An outbox table, written in the same transaction as the move and sent until it succeeds, is the likely shape.
- **Polling while Forgejo is down:** what the poller does, and what `show` and `jobs` say.
- What the CLI tells the user in each case, and which `502 forgejo_failed` responses leave work half done.

Settled constraints from [Server internals](12-server-internals.md): no transaction stays open across a git or Forgejo call; the API is synchronous for `pickup`, Ship's `submit`, and `abandon` ([HTTP API shape](10-http-api-shape.md)).

## Answer

Settled with the user. Glossary: **Attempt** now says why the word fits: its Stage Output may be sent back. No ADR: the choices are reversible and recorded here.

**Operations**
- An operation is one server action that moves a Job with a single save and also does work in git or on Forgejo: Pickup, Submit at Plan, Build and Evaluate, Submit at Ship, a Refused or Accepted Report, and Abandon.

**The rule: one save, with steps either side of it**
- A step whose result the save needs runs **before** the save and is safe to repeat. If one fails, nothing is saved, the call returns the error, and the user runs the same command again.
- Every other step runs **after** the save, as a row in an **outbox**, written in the save's own transaction. The user's call succeeds once the save lands.
- A retry knows a step is done by asking git or Forgejo, not from a record of progress, so a step that succeeded just before the server stopped is still found.

| Operation | Before the save | The save | After the save (outbox) |
|---|---|---|---|
| Pickup | fetch the Ticket, check eligibility and the branch name | Job, Ticket Snapshot, Plan 1 | create the branch and worktree; labels; Pickup comment |
| Submit at Plan, Build, Evaluate | git checks | Stage Output, next Attempt | labels |
| Submit at Ship | save the Ship marker if a push is needed; push; find or open the pull request | Manifest | labels; on a later Ship, replace the description, the "Shipped again" comment, re-request review |
| Report: Refused | read the reviews and Feedback | Feedback, Build Attempt | labels |
| Report: Accepted | read the pull request | Job ends as Done | labels; Done comment; close the issue; remove the worktree |
| Abandon | at Ship, read the pull request; refuse a dirty worktree | Job ends as Abandoned | close the pull request; restore ready labels; Abandoned comment; remove the worktree |

**Ship**
- Pushes with `--force-with-lease=refs/heads/<branch>:<sha>`, always with an explicit expected commit (empty on the first Ship), never the bare flag. When no force is needed, this is a normal fast-forward push. A remote already at the local head is "Everything up-to-date" with exit 0, so a retry after a push that landed needs no extra code. A remote at any other commit is refused. Checked with git 2.54.
- Rewriting shipped history stays allowed, as [Forgejo Forge](08-forgejo-forge.md) settled. Old commit ids in the history (earlier Build Notes ranges, judged commits, earlier Manifests) may stop resolving once git cleans up; the server never acts on them.
- Before opening a pull request, Ship looks one up with `GET /pulls/{base}/{head}` and adopts an open one, so a retry never opens a second.
- **The Ship marker is saved before the push**, on the Shipment, and is re-read and re-saved whenever a push is still needed. Once the push has landed, a retry keeps the saved marker. A review counts toward Refused if it is newer than the marker, whatever commit it names. Matching a review's `commit_id` to the shipped head was weighed and dropped: `commit_id` records whatever commit the reviewer's page or script sends, which misses reviews from the Commits tab, tabs loaded before the push, and a head someone else moved (findings: `docs/research/forgejo-review-commit-id.md` on branch `research/forgejo-review-commit-id`, commit `6b0c9e1`; local only, since the push failed without `ssh`).

**Abandon**
- At Ship with a Shipment, Abandon reads the pull request first. If it is merged, the Job moves to Done and the call refuses with `409 shipment_accepted`. Any other answer is not recorded, and the Job is abandoned. If Forgejo can't be reached, Abandon at Ship fails with `502`; at any other Stage it never needs Forgejo.
- A dirty worktree is refused with `409 worktree_dirty`. `taiichi abandon --discard` sends `discard: true`, and the removal then uses `--force`. The confirmation prompt mentions uncommitted changes.
- Closing the pull request is an outbox row. While Forgejo is down nobody can merge, and once it is back the close follows within one `poll_interval`.

**Done**
- A worktree with uncommitted changes is kept: its removal row ends `skipped`, `worktree` stays set, and `show` says "worktree kept: uncommitted changes". Taiichi never deletes uncommitted work unless a human says so.

**The outbox**
- A table, `outbox`: `id` (the order), `job_id`, `effect` (such as `tracker_labels`, `tracker_comment`, `tracker_close`, `forge_close`, `git_worktree_add`, `git_worktree_remove`), `payload` (JSON, worked out at save from the Job's settings copy), `created_at`, `tries`, `last_error`, `last_tried_at`, `succeeded_at`, `given_up_at`.
- Rows are effects (instructions for the outside world), not events. The Job's Attempts, Stage Outputs and `ended_at` already record every move, so an events table would store each one twice. A feed of moves is left to [Watching a Job live](18-watching-a-job-live.md) and [The executor seam](19-executor-seam.md).
- **One sender process** in the server sends rows, standing apart from the Job processes because ended Jobs have none and Done still has work after its save. After each save, the Job's process tells the sender to go.
- **Order:** a failing row holds back only the later rows of the same Job aimed at the same place (Tracker, Forge, or git).
- **Retries:** every `poll_interval`. A row that has failed for 24 hours is given up, with the rows waiting behind it. Every server start retries all given-up rows, so a fixed token or a recovered Forgejo takes effect on restart, with no new command.
- **Kept:** finished rows stay until their Job is deleted, as a record of what Taiichi did outside.
- Every comment Taiichi posts, on the Tracker or the Forge, is skipped if Taiichi's account already posted the same text on that issue or pull request.

**Replies and the CLI**
- `pickup`, `submit` and `abandon` wait for the first try of their Job's new rows, then reply `200` either way. The Job object gains `outbox`: `null` when nothing is pending, otherwise `{pending, failing_since, last_error}`, with given-up rows marked. The CLI prints a warning under its normal output, and `jobs` marks the Job.
- When `submit` fails and the body came from the editor or stdin, the CLI saves it to `$XDG_STATE_HOME/taiichi/submit/<job>-<stage>-<attempt>.md` and prints `taiichi submit <job> -f <path>`. Nothing is saved on the server; a draft Manifest would break write-once documents and the way Position is derived.
- **`502 forgejo_failed` now always means nothing was saved.** Only Ship's `submit` can leave work done on Forgejo (a push, a pull request), and running it again finishes it. Steps after the save never produce a `502`.

**Polling while Forgejo is down**
- The poller keeps its interval and moves `since` forward only after a successful poll. A failed check leaves no Report: `reported_at` stays the last Report, and `reports_failing_since` is set until a check succeeds. `show` and `jobs` print, for example, `Shipment pending · reported 11:58 · reports failing since 12:03`.

**Model changes**
- **`shipments`**, one row per Job inside the Job aggregate: `url`, `number`, `ship_marker`, `last_report`, `reported_at`, `reports_failing_since`. A Shipment is the pull request itself: it belongs to the Job across every Ship Attempt, changes with every Report, and is in no Stage Input, unlike a Stage Output. Created with the first Manifest's save; the first Ship's marker is empty, since a new pull request has no reviews.
- **`attempts`** drops `last_report`, `reported_at` and their `CHECK`. `show`'s history gives a Ship Attempt's Report as Refused if a Build follows it, Accepted if the Job is Done, otherwise the Shipment's `last_report`.
- The Manifest's `meta` no longer carries the Ship marker. Each Manifest still records its own Ship and is written once.
- **Startup** gains a step before the socket opens: start the outbox sender, which first retries given-up rows, then sends unsent ones.

**Attempts and gates**
- "Attempt" was questioned, since a failing Verdict isn't Evaluate failing, and a rename to Visit was weighed and dropped. The word fits once each Attempt fails in one way only: its own Stage Output is sent back. In v1 that happens to Build, through a failing Verdict or a Refused Shipment.
- Plan stays numbered like every Stage in the data and the API. While a Stage has only one Attempt, the CLI prints it plainly ("Plan", not "Plan 1").
- **Gates are post-v1:** a human check of a Stage Output before the Job moves on, able to send it back with notes, so the Stage repeats. They come with a second operator, such as an AI agent. Tracked as [Gates](../../taiichi-post-v1/issues/04-gates.md). In v1 a Job never returns to Plan; a wrong Blueprint means Abandon and Pickup again.

**Changes to earlier decisions**
- [Server internals](12-server-internals.md): the `shipments` and `outbox` tables, the marker's new home, and the startup step.
- [Forgejo Forge](08-forgejo-forge.md): the marker is saved before the push; Ship's later-Attempt extras and Abandon's close are outbox rows.
- [HTTP API shape](10-http-api-shape.md): `outbox` and `reports_failing_since` on the Job, `409 shipment_accepted`, and `discard` on `abandon`.

## Comments

**From [A Shipment changing under a Job](16-shipment-changing-under-a-job.md):**
- Abandon reads the pull request whenever the Job has a Shipment, at any Stage, not only at Ship.
- Submit at Ship first reads the pull request by number: merged ends the Job as Done with `409 shipment_accepted`; closed is `409 shipment_closed`, after restoring a deleted branch.

**From [A Ticket changing under a running Job](17-ticket-changing-under-a-job.md):**
- When a Tracker row gets a 404, the sender reads the repo. If the repo answers, the issue was deleted and the row ends `skipped`. If not, the row keeps failing as usual.
- [Oban or the hand-rolled outbox](21-oban-or-hand-rolled-outbox.md) reopens whether the outbox is hand-rolled.
