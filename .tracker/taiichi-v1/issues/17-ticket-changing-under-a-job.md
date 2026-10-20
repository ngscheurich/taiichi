# A Ticket changing under a running Job

Type: grilling
Status: resolved

## Question

What happens when a Job's Ticket changes on the Tracker while the Job is active?

- The Ticket is edited: the Ticket Snapshot never changes, so is the edit ignored, reported, or a reason to stop?
- The Ticket is closed.
- Someone relabels it, including re-adding a ready label that Pickup removed, or removing or swapping a position label.
- Taiichi's own writes coming back: it swaps labels on every move and closes the Ticket on Done, so a Report on a Ticket could be its own change. Feedback faces the same problem and leaves out the factory's own account. If the Tracker reports on Tickets, its values get their own group in `GLOSSARY.md`, `### Reports on a Ticket`, beside `### Reports on a Shipment`.

Does the server ever look at the Ticket after Pickup? [Forgejo Tracker rules](07-forgejo-tracker-rules.md) settled what the Tracker writes; this settles what it does about changes it didn't make.

## Answer

Settled with the user. Glossary: no change. A Job doesn't wait on its Ticket and nothing about the Ticket moves the Job, so there is no `### Reports on a Ticket` group; the Ticket's changes are flags `show` prints. No ADR: the choices are reversible and recorded here.

Forgejo facts were read from the Forgejo source at v17.0.0, spot-checked against v13.0.5 (findings: `docs/research/forgejo-issue-changes.md` on branch `research/forgejo-issue-changes`, commit `55658b9`; local only, since the push failed without `ssh`):
- Title and body edits, closing, reopening, label changes and new comments move an issue's `updated` time. Editing or deleting a comment doesn't. A repo admin can backdate `updated`.
- A deleted issue is hard-deleted with no trace and no webhook; every read or write on it returns 404. Issue numbers are never reused.
- `POST …/labels` adds only the labels given, `DELETE …/labels/{id}` removes one (204 if it wasn't there), and `PUT …/labels` replaces the set. Adding an exclusive label removes every other *exclusive* label in its scope (the name before the last `/`), including labels the caller didn't name.
- `GET …/issues/{n}/comments?since=` filters on a comment's `updated` time, so it sees new and edited comments.

Other Trackers were checked from their docs (findings: `docs/research/tracker-label-exclusivity.md` on branch `research/tracker-label-exclusivity`, commit `aa7699d`; local only): GitLab's scoped labels and Linear's label groups are exclusive too, GitHub has none, and all three can add or remove one label at a time. GitLab, Linear and the local markdown tracker each have a native single-valued status.

**Closed or deleted**
- A Ticket closed while its Job is active is flagged "Ticket closed" in `show` and `jobs`. The Job carries on, and a human decides whether to run `taiichi abandon`. The flag clears if the Ticket is reopened. Taiichi never abandons a Job on its own, as with a Closed Shipment.
- A deleted Ticket is flagged "Ticket deleted" the same way. When a Tracker outbox row (labels, a comment, closing) gets a 404, the sender reads the repo: if the repo answers, the issue is gone and the row ends `skipped`; if not, the row keeps failing as usual, since Forgejo also answers 404 when a token loses access. Done and Abandon still complete, and Abandon has no issue to restore ready labels on.

**Edits and comments**
- The title, the body, and the labels Taiichi doesn't name are compared with the Ticket Snapshot. Stage Inputs never change; the Snapshot stays what the Job is held to. Comparing state rather than watching for events means an edit that is undone clears itself.
- Comments from anyone but Taiichi's account, created or edited after the seen point, are counted.
- **Taiichi's own writes never raise a flag.** Its labels are left out of the comparison, its comments come from its own account, and it closes the Ticket only at Done, when the Job has ended and its checks have stopped.

**`ack`**
- `taiichi ack [job]` (API method `ack` with `{job}`) marks what's new on the Job as seen: the Ticket's edits and comments, and the Shipment's new-comment count. It records the state as the last check saw it, so it never calls Forgejo and never returns `502`.
- It never touches the feedback cursor: the next Refusal still carries every comment since the last Feedback.
- "Ticket closed", "Ticket deleted", "Shipment closed" and the branch-head flag aren't acked; they stay until their cause goes away.
- Without it, a second edit to an already-flagged Ticket would read exactly like the first.

**Labels (Forgejo adapter)**
- Every label write sets Taiichi's labels exactly: it removes every ready label and every other position label, then adds the new position label. Abandon removes every position label and adds back the ready labels from the Snapshot. A human's relabel lasts until the next move. Labels are never commands: adding a position label by hand doesn't move the Job, and a ready label re-added to an active Job's Ticket still meets "already has an active Job" at Pickup.
- Labels Taiichi doesn't name are never touched. Taiichi only adds (`POST`) or removes (`DELETE`) labels named in the Job's settings copy, and never replaces the whole set with `PUT`. The Snapshot still records every label.
- `init` warns and asks when a configured exclusive label shares its scope with exclusive labels outside the config, naming the labels Forgejo would strip at every move. `--yes` skips the prompt. It doesn't refuse, since dropping such a label may be wanted. The defaults are safe: `taiichi/ready` and the five position labels are the only exclusive labels in the `taiichi` scope.

**The Tracker port**
- The port names Positions, not labels: write-back marks the Ticket for a Position, as `moved(old_job, new_job)` already does. It gains `check`, which returns the Ticket's state (`open`, `closed`, or `deleted`), what differs from the Snapshot and from the seen point (title, body, labels), and the new comments. Each adapter leaves its own markers out of the comparison.
- The label rules above belong to the Forgejo adapter. Another adapter could use a native status instead, such as a Linear workflow state or a `Stage:` line in a markdown file.

**Checks**
- **One checker process per active Job**, started and stopped with the Job beside its Job process, under the same supervisor. Every `poll_interval` it reads the Ticket (`GET /issues/{n}` and `GET /issues/{n}/comments?since=<seen point>`) and, if the Job has a Shipment, the Shipment (the pull request, its reviews, and its comments after the cursor). It hands what it finds to the Job's process, which saves it and makes any move; the checker never writes the database. Startup's step that starts each Job's process starts its checker too, and the first check runs at once.
- **The Forge's repo-wide `since` query is dropped.** It existed only because Forgejo's API made it the cheapest poll. Reading each Shipment directly makes both sides one shape and removes the in-memory `since` and its boundary, backdating and restart cases. The cost is two calls per Ticket and three per Shipment, every tick, even when nothing changed. [Oban or the hand-rolled outbox](21-oban-or-hand-rolled-outbox.md) may revisit who drives the checks.
- A failed check changes nothing: the flags keep their last values and `checks_failing_since` is set until a check succeeds, as `reports_failing_since` does for Shipments.

**Data**
- A new **`tickets`** table, one row per Job inside the Job aggregate: `state`, the current title, body and other labels, `checked_at`, `checks_failing_since`, and the seen point (`seen_at` with the title, body and labels at the last `ack`, starting as the Snapshot and Pickup's time), and `new_comments`. It holds the Ticket as the last check saw it, not the Ticket itself.
- `shipments` gains a seen point: the latest comment id at the last `ack`. Its new-comment count counts after the later of that id and the feedback cursor.
- "Differs from Snapshot" and "edited since seen" are worked out from stored values, never saved separately.

**Display**
- `jobs` marks a Job for: Ticket closed or deleted, Ticket edited since seen, new comments on the Ticket or the Shipment, and checks failing. Once acked, an edited Ticket no longer marks it.
- `show` prints all of it, plus "differs from Snapshot: body, labels" for as long as it's true:

```
Ticket #42 · closed · checked 12:04
  differs from Snapshot: body
  edited since seen: body
  2 new comments
```

- When a Stage Input includes the Ticket Snapshot and the Ticket now differs from it, the CLI prints a note above the scissors line in `$EDITOR`, such as "Ticket #42 has changed since this Snapshot: body. See <url>." It is rendered by the CLI only; the stored Stage Input doesn't change.
- The Job object gains `ticket: {url, state, differs_from_snapshot, edited_since_seen, new_comments, checked_at, checks_failing_since}`.

**New tickets**
- [Does Oban fit the outbox and the pollers?](20-oban-fit.md) and [Oban or the hand-rolled outbox](21-oban-or-hand-rolled-outbox.md), raised by the user: Oban in place of the outbox settled in [An operation stopping partway](15-operation-stopping-partway.md).

**Changes to earlier decisions**
- [Forgejo Tracker rules](07-forgejo-tracker-rules.md): each label write sets Taiichi's labels exactly, not only "removes the previous position label"; labels are only added or removed one at a time; a 404 on a deleted issue ends a row `skipped`.
- [Forgejo Forge](08-forgejo-forge.md) and [A Shipment changing under a Job](16-shipment-changing-under-a-job.md): the Shipment is read directly each tick by the Job's checker; the repo-wide `since` query and the per-location poller are gone; the new-comment count is acked.
- [Server internals](12-server-internals.md): per-Job checkers replace the per-location pollers; new `tickets` table; `shipments` gains a seen comment id.
- [An operation stopping partway](15-operation-stopping-partway.md): the sender's repo check on a 404, and `skipped` for a deleted issue.
- [`taiichi init` and configuration](09-init-and-configuration.md): the exclusive-scope warning.
- [CLI commands](06-cli-commands.md): the `ack` verb and the editor's Snapshot note.
- [HTTP API shape](10-http-api-shape.md): the `ack` method and the Job object's `ticket` field.
