# Forgejo Forge

Type: grilling
Status: resolved
Blocked by: 02, 03

## Question

How does the Forgejo Forge ship a Shipment and Report on it?

- Pushing the branch and opening the pull request: title, body (including any closing keyword), target branch.
- What the Manifest records.
- Polling: what is polled, how often (configurable?), and what counts as Accepted and as Refused.
- How the reviewer's feedback is gathered into Build's Stage Input on a Refused Shipment, and what happens to the open pull request when the Job returns to Ship.

Build on [Forgejo API for the Tracker and Forge](02-forgejo-api.md) and keep the shape that [Does the Forge interface fit GitHub and GitLab?](03-forge-interface-fit.md) validates.

Raised by [Does the Forge interface fit GitHub and GitLab?](03-forge-interface-fit.md), also to settle here:

- How what a Report can say (Pending, Accepted, Refused, Closed) maps onto the Pipeline, especially a Shipment closed without merging.
- Ship as "ship or update" on rework, and what the Manifest stores to scope Report to the latest Ship (head commit, feedback cursor).
- Policies for comment-only reviews and drafts, and whether rework re-requests review from earlier refusers.
- Self-review limits: what happens when the Forgejo account Taiichi acts as is also the reviewer's account.

From [Forgejo Tracker rules](07-forgejo-tracker-rules.md): the Tracker closes the issue itself when the Job reaches Done, so the pull request carries no closing keyword. Whether it mentions the issue (for example `Refs #42`) is decided here.

## Answer

Settled with the user. Glossary: **Closed** added for a Shipment withdrawn without merging, and **Feedback** widened to the comments left since the last Feedback, from anyone but the factory's own account.

**Ship**
- Pushes over HTTPS as Taiichi's Forgejo account, with the token the server holds, using `--force-with-lease` against the last shipped commit. A rewrite of the Job's own history goes through; if someone else pushed in the meantime, the submission is refused with a clear error and the Job stays at Ship.
- **First Ship Attempt:** opens a pull request against the repo's default branch, read from Forgejo. The title is the submitted one, or the Ticket Snapshot's title. The body is the Manifest body word for word, followed by a footer line with the Ticket URL from the Ticket Snapshot (`Ticket: <url>`). This works across repos and Trackers without the Forge depending on the Tracker. No closing keyword, draft prefix, labels, assignees, or requested reviewers.
- **Later Ship Attempts:** the new body replaces the description, since it should describe the change as it now stands; the title changes only if one is given. Taiichi also posts a short comment marking the new round (for example "Shipped again: `abc123..def456`") and re-requests review from everyone who refused in the last round.
- **Manifest:** the title, the pull request's URL and number, the target branch, the shipped commit, the Ship marker, and the feedback cursor.

**Report**
- **Accepted:** the pull request is merged. The Job moves to Done.
- **Refused:** someone's most recent review since the latest Ship asks for changes and hasn't been dismissed. Reviews from before the latest Ship are ignored. The Job returns to Build right away.
- **Pending:** anything else, including comment-only reviews and approvals that haven't been merged.
- **Closed:** the pull request is closed without merging. The Job is not abandoned automatically, since a closed pull request may be a mistake or mean "start over": it stays at Ship, `show` and `jobs` flag it, polling carries on if the pull request is reopened, and a human decides whether to run `taiichi abandon`.

**Markers**
- The **Ship marker** is where the latest Ship happened; only reviews after it can make the Shipment Refused.
- The **feedback cursor** is the last comment already carried into a Feedback. The next Feedback gathers everything after it, including comments posted while the Job was in Build or Evaluate, so a second reviewer's late comments aren't lost.
- Both are Forgejo's own review and comment ids, which only increase, so no clocks are compared.

**Feedback**
- Everything after the feedback cursor from anyone but Taiichi's account: review bodies (including comment-only reviews and approvals with text), inline comments with their resolved flag, and conversation comments. Each item records author, location if inline, resolved flag, and text.

**Polling**
- The server polls a repo's Forge only while at least one of its Jobs waits at Ship, with one `GET /repos/{o}/{r}/issues?type=pulls&state=all&since=<last poll>`. For each changed pull request belonging to a waiting Job, it fetches the reviews, and the Feedback on a Refusal.
- On startup it checks every waiting Job once to catch up.
- The interval is a server-wide setting, defaulting to 60 seconds (for [`taiichi init` and configuration](09-init-and-configuration.md)).

**Other rules**
- **Abandoning with an open Shipment:** Taiichi closes the pull request with a comment saying the Job was abandoned and the branch is kept, so nobody later merges work with no Job behind it. Abandoned leads to Closed, never the reverse.
- **Self-review:** the token must belong to a dedicated bot account that never reviews, or Forgejo blocks that reviewer from requesting changes. Taiichi can't detect this, so it's documented, and `taiichi status` shows which Forgejo account Taiichi acts as.
- **A Shipment changing while the Job is away from Ship** (merged, closed, or pushed to during Build or Evaluate) is left to the map's fog, widened from the earlier item about commits pushed by someone else.

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):**
- The Ship marker is saved on the Shipment before the push and re-read while a push is still needed, so a retry can't skip a review. A review counts if it is newer than the marker; matching its `commit_id` to the shipped head was dropped after research.
- `--force-with-lease` always names the expected commit. A retry finds an open pull request with `GET /pulls/{base}/{head}` before opening one.
- Replacing the description, the "Shipped again" comment, re-requesting review, and Abandon's close of the pull request are outbox rows, sent after the save.
