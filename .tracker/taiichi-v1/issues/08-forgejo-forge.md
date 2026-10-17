# Forgejo Forge

Type: grilling
Status: open
Blocked by: 02, 03

## Question

How does the Forgejo Forge ship a Shipment and detect its outcome?

- Pushing the branch and opening the pull request: title, body (including any closing keyword), target branch.
- What the Manifest records.
- Polling: what is polled, how often (configurable?), and what counts as Accepted and as Refused.
- How the reviewer's feedback is gathered into Build's Stage Input on a Refused Shipment, and what happens to the open pull request when the Job returns to Ship.

Build on [Forgejo API for the Tracker and Forge](02-forgejo-api.md) and keep the shape that [Does the Forge interface fit GitHub and GitLab?](03-forge-interface-fit.md) validates.

Raised by [Does the Forge interface fit GitHub and GitLab?](03-forge-interface-fit.md), also to settle here:

- How Report's outcomes (Pending, Accepted, Refused, Closed) map onto the Pipeline, especially a Shipment closed without merging.
- Ship as "ship or update" on rework, and what the Manifest stores to scope Report to the latest Ship (head commit, feedback cursor).
- Policies for comment-only reviews and drafts, and whether rework re-requests review from earlier refusers.
- Self-review limits: what happens when the Forgejo account Taiichi acts as is also the reviewer's account.

From [Forgejo Tracker rules](07-forgejo-tracker-rules.md): the Tracker closes the issue itself when the Job reaches Done, so the pull request carries no closing keyword. Whether it mentions the issue (for example `Refs #42`) is decided here.
