# A Shipment changing under a Job

Type: grilling
Status: resolved

## Question

What happens when a Job's Shipment changes while the Job is away from Ship, in Build or Evaluate?

- The pull request is merged: the Pipeline reaches Done only from Ship, and Report only runs at Ship.
- The pull request is closed.
- Someone else pushes to the branch: the worktree falls behind, the next Build's starting commit is off, and Ship's `--force-with-lease` refuses to overwrite the push ([Forgejo Forge](08-forgejo-forge.md)).

For each case: does the server notice (the poller runs only while a Job waits at Ship, per [Server internals](12-server-internals.md)), what does the Job do, and what does the human see?

## Answer

Settled with the user. Glossary: **Done** is reached from whichever Stage the Job is in, and **Pipeline** ends in Done or Abandoned from any Stage; **Closed** no longer keeps the Job at Ship, only stops it from shipping again until a human reopens the Shipment; **Foreign commit** added. No ADR: the choices are reversible and recorded here.

Forgejo facts behind these rules were read from the Forgejo source at v13.0.5, and they match v11.0.9 and v14.0.5:
- A push to an open pull request's branch adds a timeline comment that moves the issue's `updated` time, so the poller's `since` query sees it. A push to a closed pull request does not.
- `head.sha` on `GET /pulls/{index}` is read live from the branch.
- `GET /pulls/{base}/{head}` returns pull requests in any state, in no fixed order.
- Reopening through the API skips the web page's checks for a missing branch and for another open pull request.
- A branch protection rule can limit pushes to one account, but it also blocks force-pushes for everyone.

**Noticing**
- **The poller watches every active Job that has a Shipment**, at any Stage, not only Jobs waiting at Ship. It is still one filtered query per Forge location.
- **Away from Ship**, Accepted moves the Job to Done. Closed and a reopen (back to Pending) only update the Shipment's `last_report`. New reviews are ignored: they are older than the next Ship marker, and their comments reach the next Feedback through the cursor.
- **Branch head:** every check reads the pull request's `head.sha` and saves it on the Shipment as `branch_head`. While the worktree's head doesn't contain it (`git merge-base --is-ancestor` in the local clone), `show` and `jobs` flag the Job: "The branch has commits your worktree lacks: run `git pull --rebase` in the worktree." Taiichi's own push never raises the flag.
- **New comments:** away from Ship, the poller counts comments after the feedback cursor from anyone but Taiichi's account and saves the count on the Shipment. `show` prints, for example, "3 new comments on the Shipment since the last Feedback" with the pull request's URL, and `jobs` marks the Job. Stage Inputs don't change, so the comments still reach the next Feedback on a Refusal and are never carried twice. A reviewer who doesn't Refuse the next Ship has settled them, whether or not the builder read them.

**Merged away from Ship**
- A merge at any Stage is Accepted, and the Job moves to Done from whichever Stage it is in. The usual Done steps run: labels, the Done comment, closing the issue, removing the worktree (kept if it has uncommitted changes).
- The open Attempt keeps no Stage Output, and `show` prints it as "Build 2 · unfinished". If the branch has commits beyond the last shipped head, `show` adds "2 unshipped commits on `taiichi/42-j7-…`". The branch is kept, as for every ended Job.
- **Abandon** reads the pull request whenever the Job has a Shipment, at any Stage, not only at Ship. Merged: the Job moves to Done and Abandon is refused with `409 shipment_accepted`. Unreachable: `502`. Open: the close goes through the outbox. Already closed: nothing to send. A Job that never shipped never calls Forgejo.

**Closed away from Ship**
- The Job keeps moving through Build and Evaluate, flagged "Shipment closed" in `show` and `jobs`.
- At Ship, `submit` is refused with `409 shipment_closed` until a human reopens the pull request on Forgejo or runs `taiichi abandon`. Taiichi never reopens a pull request, and never opens a second one.
- **Branch deleted:** Forgejo closes a pull request whose branch is deleted, and its web page won't reopen one while the branch is missing. So when `submit` at Ship finds the Shipment Closed and the branch missing, it first pushes the branch from the worktree (lease expecting no branch), then refuses with `409 shipment_closed`: "Branch restored. Reopen #42 on Forgejo, then submit again." The human only reopens; the next `submit` finds the branch at the worktree's head and carries on.

**Ship reads the Shipment first**
- Later Ships read the pull request by the Shipment's `number` (the `42` in `#42`) before anything else: merged is Done with `409 shipment_accepted`, closed is `409 shipment_closed`, open goes on. They never use the lookup by base and head.
- The first Ship still uses `GET /pulls/{base}/{head}` to recover after a failed try, and adopts only an open pull request opened by Taiichi's account. A closed or merged one is refused with `409 shipment_closed`; the branch name carries the Job number, so only this Job could have used it.

**Someone else pushes to the branch**
- **The branch is shared.** Commits others push belong to the Job's work, as Feedback already comes from anyone but Taiichi's account. The human brings them into the worktree (`git pull --rebase` or a merge) before the next Ship.
- **Only Ship refuses.** Build and Evaluate submit with the branch-head flag shown, so an evaluator can fail the Verdict to take the foreign commits back to Build. Ship's push becomes:

| Forgejo's branch | Ship pushes with a lease on | Effect |
|---|---|---|
| at the last commit Taiichi shipped | that commit | replaces it; only here can the Job's own rewritten history (rebase, amend, squash) overwrite what's on Forgejo |
| contained in the worktree's head | Forgejo's current head | a fast-forward; a push landing between the read and the push is still refused |
| missing | no branch | recreates it |
| anything else | n/a | refused with `409 branch_moved`; nothing saved |

- **Ship checks the judged commit.** `submit` at Ship is refused with `409 head_moved` when the worktree's head isn't the commit the passing Evaluation judged. `taiichi submit --unjudged` (`unjudged: true`) ships it anyway, for foreign commits only found at Ship, since v1 has no way back from Ship to Build. The Manifest records the judged commit next to the shipped one.

**Build Notes**
- **The range starts at the last commit the server pinned:** for Build 1, the Job's new `base_commit`; after a failing Verdict, the commit that Evaluation judged; after a Refusal, the commit that Ship pushed. Commits made between Attempts (during Plan, or while waiting at Ship) fall into the next Build's range, so every commit belongs to one Attempt. This replaces "the head when the Attempt began", which nothing recorded.
- **`base_commit`:** Pickup reads `origin/<default>`'s head before its save and stores it on the Job; the outbox step creates the branch at that commit.
- **The range holds every commit that came into the branch during the Attempt, whoever wrote it**, except commits reachable from the local `origin/<default>`. Those come from the default branch and are left out; Build Notes records the base move instead, such as `base: main@1111 → main@2222`. The range only labels commits: Evaluate still judges the head, the whole project at that commit.
- **Foreign commits** are stamped on Build Notes at submission, each with sha and author: commits in the range reachable from the local `origin/<job branch>` ref and not from any commit Taiichi pushed. `git pull` moves that ref to what Forgejo had, and Taiichi's own pushes move it too, so this needs no Forgejo call and has no gap between polls. `git pull --rebase` keeps others' commits as they were, so they still match; a foreign commit squashed into the builder's own is no longer foreign.

**Changes to earlier decisions**
- [Forgejo Forge](08-forgejo-forge.md): polling covers every Job with a Shipment; the Ship lease table above; later Ships read the Shipment by number; the first Ship adopts only an open pull request Taiichi opened; Closed no longer keeps the Job at Ship.
- [Server internals](12-server-internals.md): `shipments` gains `branch_head` and a count of new comments; `jobs` gains `base_commit`; the poller's start and stop follow "has a Shipment" instead of "waiting at Ship".
- [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md): the range's start, the base, and foreign commits on Build Notes; the judged commit on the Manifest; Ship's judged-commit check.
- [HTTP API shape](10-http-api-shape.md): `409 branch_moved` and `409 shipment_closed`; `head_moved` and `shipment_accepted` also at Ship; `unjudged` on `submit`; the Job object's Shipment carries the branch-head flag, new-comment count, and Closed flag.
- [An operation stopping partway](15-operation-stopping-partway.md): Abandon's merge check applies whenever the Job has a Shipment; Submit at Ship first reads the pull request, and may restore a deleted branch before refusing.

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** Refused now counts any request-changes review newer than the Ship marker, whatever commit it names, so a head someone else moved doesn't hide a refusal. What the Job does when the head moves is still for this ticket. Abandon at Ship already checks for a merge first; merges while the Job is away from Ship are still here.
